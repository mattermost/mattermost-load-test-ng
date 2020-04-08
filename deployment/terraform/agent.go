package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-server/v5/mlog"
)

func (t *Terraform) generateLoadtestAgentConfig(output *terraformOutput) (*loadtest.Config, error) {
	cfg, err := loadtest.ReadConfig("")
	if err != nil {
		return nil, err
	}

	cfg.ConnectionConfiguration.ServerURL = "http://" + output.Proxy.Value.PrivateIP
	cfg.ConnectionConfiguration.WebSocketURL = "ws://" + output.Proxy.Value.PrivateIP
	cfg.ConnectionConfiguration.AdminEmail = t.config.AdminEmail
	cfg.ConnectionConfiguration.AdminPassword = t.config.AdminPassword

	return cfg, nil
}

func (t *Terraform) configureAndRunAgents(extAgent *ssh.ExtAgent, output *terraformOutput) error {
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

	for _, val := range output.Agents.Value {
		sshc, err := extAgent.NewClient(val.PublicIP)
		if err != nil {
			return err
		}
		mlog.Info("Configuring agent", mlog.String("ip", val.PublicIP))
		if uploadBinary {
			dstFile := "/home/ubuntu/tmp.tar.gz"
			mlog.Info("Uploading binary", mlog.String("file", packagePath))
			if out, err := sshc.UploadFile(packagePath, dstFile, false); err != nil {
				return fmt.Errorf("error uploading file %q, output: %q: %w", packagePath, out, err)
			}
			commands := []string{
				"rm -rf mattermost-load-test-ng",
				"tar xzf tmp.tar.gz",
				"mv mattermost-load-test-ng* mattermost-load-test-ng",
				"rm tmp.tar.gz",
			}
			cmd := strings.Join(commands, " && ")
			if out, err := sshc.RunCommand(cmd); err != nil {
				return fmt.Errorf("error running command, got output: %q: %w", out, err)
			}
		}

		mlog.Info("Uploading agent service file")
		rdr := strings.NewReader(strings.TrimSpace(agentServiceFile))
		if out, err := sshc.Upload(rdr, "/lib/systemd/system/ltagent.service", true); err != nil {
			return fmt.Errorf("error uploading file, output: %q: %w", out, err)
		}

		mlog.Info("Uploading coordinator service file")
		rdr = strings.NewReader(strings.TrimSpace(coordinatorServiceFile))
		if out, err := sshc.Upload(rdr, "/lib/systemd/system/ltcoordinator.service", true); err != nil {
			return fmt.Errorf("error uploading file, output: %q: %w", out, err)
		}

		mlog.Info("Starting agent")
		cmd := "sudo service ltagent start"
		if out, err := sshc.RunCommand(cmd); err != nil {
			return fmt.Errorf("error running command, got output: %q: %w", out, err)
		}
	}
	return nil
}

func (t *Terraform) initLoadtest(extAgent *ssh.ExtAgent, output *terraformOutput) error {
	if len(output.Agents.Value) == 0 {
		return fmt.Errorf("there are no agents to initialize load-test")
	}
	ip := output.Agents.Value[0].PublicIP
	sshc, err := extAgent.NewClient(ip)
	if err != nil {
		return err
	}
	mlog.Info("Generating load-test config")
	cfg, err := t.generateLoadtestAgentConfig(output)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	dstPath := "/home/ubuntu/mattermost-load-test-ng/config/config.json"
	mlog.Info("Uploading updated config file")
	if out, err := sshc.Upload(bytes.NewReader(data), dstPath, false); err != nil {
		return fmt.Errorf("error uploading file, output: %q: %w", out, err)
	}

	mlog.Info("Populating initial data for load-test", mlog.String("agent", ip))
	cmd := "cd mattermost-load-test-ng && ./bin/ltagent init"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command, output: %q, error: %w", out, err)
	}
	return nil
}
