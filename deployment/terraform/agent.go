package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/agent"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/logger"
	"github.com/mattermost/mattermost-server/v5/mlog"
)

func (t *Terraform) generateLoadtestAgentConfig(output *terraformOutput) loadtest.Config {
	return loadtest.Config{
		ConnectionConfiguration: loadtest.ConnectionConfiguration{
			ServerURL:                   "http://" + output.Proxy.Value.PrivateIP,
			WebSocketURL:                "ws://" + output.Proxy.Value.PrivateIP,
			AdminEmail:                  t.config.AdminEmail,
			AdminPassword:               t.config.AdminPassword,
			IdleConnTimeoutMilliseconds: 90000,
		},
		UserControllerConfiguration: loadtest.UserControllerConfiguration{
			Type: "simple",
			Rate: 1.0,
		},
		InstanceConfiguration: loadtest.InstanceConfiguration{
			NumTeams: 2,
		},
		UsersConfiguration: loadtest.UsersConfiguration{
			InitialActiveUsers: 4,
			MaxActiveUsers:     1000,
		},
		LogSettings: logger.Settings{
			EnableFile:   true,
			FileLevel:    "INFO",
			FileLocation: "loadtest.log",
		},
	}
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
		mlog.Info("configuring agent", mlog.String("ip", val.PublicIP))
		if uploadBinary {
			// upload the local file with a -local prefix to avoid collision
			// with the previously downloaded file
			dstFile := "/home/ubuntu/mattermost-load-test-ng-local.tar.gz"
			mlog.Info(fmt.Sprintf("Uploading binary file %s", packagePath))
			if out, err := sshc.UploadFile(packagePath, dstFile, false); err != nil {
				return fmt.Errorf("error uploading file %q, output: %q: %w", packagePath, string(out), err)
			}
			if out, err := sshc.RunCommand("rm -rf mattermost-load-test-ng && " +
				"tar xzf mattermost-load-test-ng-local.tar.gz && " +
				"mv $(ls -d */ | grep mattermost-load-test-ng) mattermost-load-test-ng"); err != nil {
				return fmt.Errorf("error running command, got output: %q: %w", string(out), err)
			}
		}

		mlog.Info("Uploading agent service file")
		rdr := strings.NewReader(strings.TrimSpace(agentServiceFile))
		if out, err := sshc.Upload(rdr, "/lib/systemd/system/ltagent.service", true); err != nil {
			return fmt.Errorf("error uploading file, output: %q: %w", string(out), err)
		}

		mlog.Info("Uploading coordinator service file")
		rdr = strings.NewReader(strings.TrimSpace(coordinatorServiceFile))
		if out, err := sshc.Upload(rdr, "/lib/systemd/system/ltcoordinator.service", true); err != nil {
			return fmt.Errorf("error uploading file, output: %q: %w", string(out), err)
		}

		mlog.Info("Starting agent")
		cmd := fmt.Sprintf("sudo service ltagent start")
		if out, err := sshc.RunCommand(cmd); err != nil {
			return fmt.Errorf("error running command, got output: %q: %w", string(out), err)
		}
	}
	return nil
}

func (t *Terraform) initLoadtest(extAgent *ssh.ExtAgent, ip string, output *terraformOutput) error {
	sshc, err := extAgent.NewClient(ip)
	if err != nil {
		return err
	}
	mlog.Info("Populating DB", mlog.String("agent", ip))
	cfg := t.generateLoadtestAgentConfig(output)
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	dstPath := "/home/ubuntu/mattermost-load-test-ng/config/config.json"
	mlog.Info("Uploading updated config file")
	if out, err := sshc.Upload(bytes.NewReader(data), dstPath, false); err != nil {
		return fmt.Errorf("error uploading file, output: %q: %w", string(out), err)
	}

	mlog.Info("Running init command")
	cmd := "cd mattermost-load-test-ng && ./bin/ltagent init"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command, out: %s, error: %w", out, err)
	}
	return nil
}

func (t *Terraform) configureAndRunCoordinator(extAgent *ssh.ExtAgent, ip string, output *terraformOutput) error {
	loadtestConfig := t.generateLoadtestAgentConfig(output)

	var loadAgentConfigs []agent.LoadAgentConfig
	for _, val := range output.Agents.Value {
		loadAgentConfigs = append(loadAgentConfigs, agent.LoadAgentConfig{
			Id:             val.Tags.Name,
			ApiURL:         "http://" + val.PrivateIP + ":4000",
			LoadTestConfig: loadtestConfig,
		})
	}

	mlog.Info("Setting up coordinator", mlog.String("ip", ip))
	sshc, err := extAgent.NewClient(ip)
	if err != nil {
		return err
	}

	clusterConfig := coordinator.Config{
		ClusterConfig: cluster.LoadAgentClusterConfig{
			Agents:         loadAgentConfigs,
			MaxActiveUsers: 100,
		},
		MonitorConfig: performance.MonitorConfig{
			PrometheusURL:    "http://" + output.MetricsServer.Value.PrivateIP + ":9090",
			UpdateIntervalMs: 2000,
			Queries: []prometheus.Query{
				{
					Description: "Request Duration",
					Query:       "rate(mattermost_http_request_duration_seconds_sum[1m])/rate(mattermost_http_request_duration_seconds_count[1m])",
					Threshold:   2.0,
					Alert:       true,
				},
			},
		},
	}

	data, err := json.Marshal(clusterConfig)
	if err != nil {
		return err
	}
	mlog.Info("Uploading updated config file")
	dstPath := "/home/ubuntu/mattermost-load-test-ng/config/coordinator.json"
	if out, err := sshc.Upload(bytes.NewReader(data), dstPath, false); err != nil {
		return fmt.Errorf("error running ssh command: output: %s, error: %w", out, err)
	}

	mlog.Info("Starting the coordinator")
	cmd := "sudo service ltcoordinator start"
	if err := sshc.StartCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: %s, error: %w", cmd, err)
	}

	return nil
}
