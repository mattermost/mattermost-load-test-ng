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

func (t *Terraform) initLoadtest(extAgent *ssh.ExtAgent, ip string, output *terraformOutput) error {
	sshc, err := extAgent.NewClient(ip)
	if err != nil {
		return err
	}

	cfg := t.generateLoadtestAgentConfig(output)
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	dstPath := "/opt/mattermost/config/config.json"
	if out, err := sshc.Upload(bytes.NewReader(data), dstPath, true); err != nil {
		return fmt.Errorf("error running ssh command: output: %s, error: %w", out, err)
	}

	mlog.Info("Populating DB", mlog.String("ip", ip))
	cmd := "cd /opt/mattermost && sudo ./bin/lt-agent init"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: %s, out: %s, error: %w", cmd, out, err)
	}
	return nil
}

func (t *Terraform) configureAndRunAgent(extAgent *ssh.ExtAgent, ip string) error {
	sshc, err := extAgent.NewClient(ip)
	if err != nil {
		return err
	}

	file, err := os.Open("/Users/ibrahim/lt-agent")
	if err != nil {
		return err
	}
	dstPath := "/opt/mattermost/bin/lt-agent"
	if out, err := sshc.Upload(file, dstPath, true); err != nil {
		return fmt.Errorf("error running ssh command: output: %s, error: %w", out, err)
	}
	cmd := "sudo chmod +x /opt/mattermost/bin/lt-agent"
	if out, err := sshc.RunCommand(cmd); err != nil {
		mlog.Error("error running ssh command", mlog.String("cmd", cmd), mlog.String("output", string(out)), mlog.Err(err))
		return err
	}

	// Upload service file.
	mlog.Info("Uploading service file", mlog.String("ip", ip))
	rdr := strings.NewReader(strings.TrimSpace(agentServiceFile))
	if out, err := sshc.Upload(rdr, "/lib/systemd/system/lt-agent.service", true); err != nil {
		mlog.Error("error uploading systemd file", mlog.String("output", string(out)), mlog.Err(err))
		return err
	}

	// Starting agent.
	mlog.Info("Starting agent", mlog.String("ip", ip))
	cmd = fmt.Sprintf("sudo service lt-agent start")
	if out, err := sshc.RunCommand(cmd); err != nil {
		mlog.Error("error running ssh command", mlog.String("cmd", cmd), mlog.String("output", string(out)), mlog.Err(err))
		return err
	}
	// TODO: copy simplecontroller.json
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
	dstPath := "/opt/mattermost/config/coordinator.json"
	if out, err := sshc.Upload(bytes.NewReader(data), dstPath, true); err != nil {
		return fmt.Errorf("error running ssh command: output: %s, error: %w", out, err)
	}

	file, err := os.Open("/Users/ibrahim/lt-coordinator")
	if err != nil {
		return err
	}
	dstPath = "/opt/mattermost/bin/lt-coordinator"
	if out, err := sshc.Upload(file, dstPath, true); err != nil {
		return fmt.Errorf("error running ssh command: output: %s, error: %w", out, err)
	}

	cmd := "sudo chmod +x /opt/mattermost/bin/lt-coordinator"
	if out, err := sshc.RunCommand(cmd); err != nil {
		mlog.Error("error running ssh command", mlog.String("cmd", cmd), mlog.String("output", string(out)), mlog.Err(err))
		return err
	}

	cmd = "cd /opt/mattermost && sudo ./bin/lt-coordinator"
	if err := sshc.StartCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: %s, error: %w", cmd, err)
	}

	return nil
}
