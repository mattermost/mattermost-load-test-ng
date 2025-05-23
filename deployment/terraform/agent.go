package terraform

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

const dstUsersFilePath = "/home/%s/users.txt"

func (t *Terraform) generateLoadtestAgentConfig() (*loadtest.Config, error) {
	cfg, err := loadtest.ReadConfig("")
	if err != nil {
		return nil, err
	}

	url := getServerURL(t.output, t.config)

	cfg.ConnectionConfiguration.ServerURL = "http://" + url
	cfg.ConnectionConfiguration.WebSocketURL = "ws://" + url
	cfg.ConnectionConfiguration.AdminEmail = t.config.AdminEmail
	cfg.ConnectionConfiguration.AdminPassword = t.config.AdminPassword

	if t.config.UsersFilePath != "" {
		cfg.UsersConfiguration.UsersFilePath = fmt.Sprintf(dstUsersFilePath, t.Config().AWSAMIUser)
	}

	return cfg, nil
}

func (t *Terraform) configureAndRunAgents(extAgent *ssh.ExtAgent) error {
	var uploadBinary bool
	var packagePath string
	if strings.HasPrefix(t.config.LoadTestDownloadURL, filePrefix) {
		packagePath = strings.TrimPrefix(t.config.LoadTestDownloadURL, filePrefix)
		info, err := os.Stat(packagePath)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("load-test package path %s has to be a regular file", packagePath)
		}
		uploadBinary = true
	}

	commands := []string{
		"rm -rf mattermost-load-test-ng*",
		"tar xzf tmp.tar.gz",
		"mv mattermost-load-test-ng* mattermost-load-test-ng",
		"rm tmp.tar.gz",
	}
	if !uploadBinary {
		commands = append([]string{"wget -O tmp.tar.gz " + t.config.LoadTestDownloadURL}, commands...)
	}

	// If UsersFilePath is present, split the user credentials among all the agents,
	// so that the logged in users don't clash
	splitFiles := make([][]string, 0, len(t.output.Agents))
	if t.config.UsersFilePath != "" {
		f, err := os.Open(t.config.UsersFilePath)
		if err != nil {
			return fmt.Errorf("error opening UsersFilePath %q", t.config.UsersFilePath)
		}
		scanner := bufio.NewScanner(f)
		for range t.output.Agents {
			splitFiles = append(splitFiles, []string{})
		}
		i := 0
		for scanner.Scan() {
			splitFiles[i] = append(splitFiles[i], scanner.Text())
			i = (i + 1) % len(splitFiles)
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("error reading UsersFilePath %q: %w", t.config.UsersFilePath, err)
		}
	}

	wg := sync.WaitGroup{}
	var foundErr atomic.Bool

	for i, val := range t.output.Agents {
		wg.Add(1)
		agentNumber := i
		instance := val

		go func() {
			defer wg.Done()

			sshc, err := extAgent.NewClient(t.Config().AWSAMIUser, instance.GetConnectionIP())
			if err != nil {
				mlog.Error("error creating ssh client", mlog.Err(err), mlog.Int("agent", agentNumber))
				foundErr.Store(true)
				return
			}
			mlog.Info("Configuring agent", mlog.String("ip", instance.GetConnectionIP()), mlog.Int("agent", agentNumber))
			if uploadBinary {
				dstFilePath := fmt.Sprintf("/home/%s/tmp.tar.gz", t.Config().AWSAMIUser)
				mlog.Info("Uploading binary", mlog.String("file", packagePath), mlog.Int("agent", agentNumber))
				if out, err := sshc.UploadFile(packagePath, dstFilePath, false); err != nil {
					mlog.Error("error uploading file", mlog.String("path", packagePath), mlog.String("output", string(out)), mlog.Err(err), mlog.Int("agent", agentNumber))
					foundErr.Store(true)
					return
				}
			}

			cmd := strings.Join(commands, " && ")
			if out, err := sshc.RunCommand(cmd); err != nil {
				mlog.Error("error running command", mlog.Int("agent", agentNumber), mlog.String("output", string(out)), mlog.Err(err))
				foundErr.Store(true)
				return
			}

			tpl, err := template.New("").Parse(apiServiceFile)
			if err != nil {
				mlog.Error("could not parse agent service template", mlog.Err(err), mlog.Int("agent", agentNumber))
				foundErr.Store(true)
				return
			}

			tplVars := map[string]any{
				"blockProfileRate": t.config.PyroscopeSettings.BlockProfileRate,
				"execStart":        fmt.Sprintf(baseAPIServerCmd, t.Config().AWSAMIUser),
				"User":             t.Config().AWSAMIUser,
			}
			if t.config.EnableAgentFullLogs {
				tplVars["execStart"] = fmt.Sprintf("/bin/bash -c '%s &>> /home/%s/ltapi.log'", t.Config().AWSAMIUser, baseAPIServerCmd)
			}
			buf := bytes.NewBufferString("")
			tpl.Execute(buf, tplVars)

			otelcolConfig, err := renderAgentOtelcolConfig(instance.Tags.Name, t.output.MetricsServer.GetConnectionIP())
			if err != nil {
				mlog.Error("unable to render otelcol config", mlog.Int("agent", agentNumber), mlog.Err(err))
				foundErr.Store(true)
				return
			}

			otelcolConfigFile, err := fillConfigTemplate(otelcolConfig, map[string]any{
				"User": t.Config().AWSAMIUser,
			})
			if err != nil {
				mlog.Error("unable to render otelcol config", mlog.Int("agent", agentNumber), mlog.Err(err))
				foundErr.Store(true)
				return
			}

			batch := []uploadInfo{
				{srcData: strings.TrimPrefix(buf.String(), "\n"), dstPath: "/lib/systemd/system/ltapi.service", msg: "Uploading load-test api service file"},
				{srcData: strings.TrimPrefix(clientSysctlConfig, "\n"), dstPath: "/etc/sysctl.conf"},
				{srcData: strings.TrimPrefix(limitsConfig, "\n"), dstPath: "/etc/security/limits.conf"},
				{srcData: strings.TrimPrefix(prometheusNodeExporterConfig, "\n"), dstPath: "/etc/default/prometheus-node-exporter"},
				{srcData: strings.TrimSpace(otelcolConfigFile), dstPath: "/etc/otelcol-contrib/config.yaml"},
			}

			if t.config.UsersFilePath != "" {
				batch = append(batch, uploadInfo{srcData: strings.Join(splitFiles[agentNumber], "\n"), dstPath: fmt.Sprintf(dstUsersFilePath, t.Config().AWSAMIUser), msg: "Uploading list of users credentials"})
			}

			// If SiteURL is set, update /etc/hosts to point to the correct IP
			if t.config.SiteURL != "" {
				appHostsFile, err := t.getAppHostsFile(agentNumber)
				if err != nil {
					mlog.Error("error getting output", mlog.Err(err), mlog.Int("agent", agentNumber))
					foundErr.Store(true)
					return
				}

				batch = append(batch, uploadInfo{srcData: appHostsFile, dstPath: "/etc/hosts", msg: "Updating /etc/hosts to point to the correct IP"})
			}

			if err := uploadBatch(sshc, batch); err != nil {
				mlog.Error("error uploading batch", mlog.Err(err), mlog.Int("agent", agentNumber))
				foundErr.Store(true)
				return
			}

			if err := t.setupPrometheusNodeExporter(sshc); err != nil {
				mlog.Error("error setting up prometheus node exporter", mlog.Err(err), mlog.Int("agent", agentNumber))
			}

			cmd = "sudo systemctl restart otelcol-contrib && sudo systemctl restart prometheus-node-exporter"
			if out, err := sshc.RunCommand(cmd); err != nil {
				mlog.Error("error running ssh command", mlog.Int("agent", agentNumber), mlog.String("cmd", cmd), mlog.String("out", string(out)), mlog.Err(err))
				foundErr.Store(true)
				return
			}

			if out, err := sshc.RunCommand("sudo sysctl -p"); err != nil {
				mlog.Error("error running sysctl", mlog.String("output", string(out)), mlog.Err(err), mlog.Int("agent", agentNumber))
				foundErr.Store(true)
				return
			}

			mlog.Info("Starting load-test api server", mlog.Int("agent", agentNumber))
			if out, err := sshc.RunCommand("sudo systemctl daemon-reload && sudo systemctl restart ltapi"); err != nil {
				mlog.Error("error starting load-test api server", mlog.String("output", string(out)), mlog.Err(err), mlog.Int("agent", agentNumber))
				foundErr.Store(true)
				return
			}
		}()
	}

	wg.Wait()

	if foundErr.Load() {
		return errors.New("error configuring agents, check above logs for more information")
	}

	return nil
}

func (t *Terraform) initLoadtest(extAgent *ssh.ExtAgent, initData bool) error {
	if len(t.output.Agents) == 0 {
		return errors.New("there are no agents to initialize load-test")
	}
	ip := t.output.Agents[0].GetConnectionIP()
	sshc, err := extAgent.NewClient(t.Config().AWSAMIUser, ip)
	if err != nil {
		return err
	}
	mlog.Info("Generating load-test config")
	cfg, err := t.generateLoadtestAgentConfig()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	dstPath := fmt.Sprintf("/home/%s/mattermost-load-test-ng/config/config.json", t.Config().AWSAMIUser)
	mlog.Info("Uploading updated config file")
	if out, err := sshc.Upload(bytes.NewReader(data), dstPath, false); err != nil {
		return fmt.Errorf("error uploading file, output: %q: %w", out, err)
	}

	if initData && t.config.TerraformDBSettings.ClusterIdentifier == "" {
		mlog.Info("Populating initial data for load-test", mlog.String("agent", ip))
		cmd := fmt.Sprintf("cd mattermost-load-test-ng && ./bin/ltagent init --user-prefix '%s' --server-url 'http://%s:8065'",
			t.output.Agents[0].Tags.Name, t.output.Instances[0].GetConnectionIP())
		if out, err := sshc.RunCommand(cmd); err != nil {
			// TODO: make this fully atomic. See MM-23998.
			// ltagent init should drop teams and channels before creating them.
			// This needs additional delete actions to be added.
			if strings.Contains(string(out), "with that name already exists") {
				return nil
			}
			return fmt.Errorf("error running ssh command, output: %q, error: %w", out, err)
		}
	}

	return nil
}

func (t *Terraform) getAppHostsFile(index int) (string, error) {
	output, err := t.Output()
	if err != nil {
		return "", err
	}

	// The new entry in /etc/hosts will make SiteURL point to:
	// - The first instance's IP if there's a single app node
	// - The IP of one of the proxy nodes if there's more than one app node
	proxyHost := ""
	if output.HasProxy() {
		// This allows a bare-bones multi-proxy setup
		// where we simply round-robin between the proxy nodes amongst the agents.
		// Not a perfect solution, because ideally we would have used
		// a dedicated DNS service (like Route53), but it works for now.
		proxyIndex := index % len(output.Proxies)
		proxyHost += fmt.Sprintf("%s %s\n", output.Proxies[proxyIndex].GetConnectionIP(), t.config.SiteURL)
	} else {
		proxyHost = fmt.Sprintf("%s %s\n", output.Instances[0].GetConnectionIP(), t.config.SiteURL)
	}

	return fmt.Sprintf(appHosts, proxyHost), nil
}
