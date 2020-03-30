package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"

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

func (t *Terraform) initLoadtest(extAgent *ssh.ExtAgent, ip string) error {
	sshc, err := extAgent.NewClient(ip)
	if err != nil {
		return err
	}

	mlog.Info("Populating DB", mlog.String("ip", ip))
	cmd := "cd mattermost-load-test-ng && export PATH=$PATH:/usr/local/go/bin && go run ./cmd/loadtest init"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: %s, out: %s, error: %w", cmd, out, err)
	}
	return nil
}

func (t *Terraform) configureAndRunAgent(extAgent *ssh.ExtAgent, ip string, output *terraformOutput) error {
	sshc, err := extAgent.NewClient(ip)
	if err != nil {
		return err
	}

	loadtestConfig := t.generateLoadtestAgentConfig(output)
	data, err := json.Marshal(loadtestConfig)
	if err != nil {
		return err
	}
	dstPath := "/home/ubuntu/mattermost-load-test-ng/config/config.json"
	if out, err := sshc.Upload(bytes.NewReader(data), dstPath, false); err != nil {
		return fmt.Errorf("error running ssh command: output: %s, error: %w", out, err)
	}

	// Starting agent.
	mlog.Info("Starting agent", mlog.String("ip", ip))
	cmd := "cd mattermost-load-test-ng && export PATH=$PATH:/usr/local/go/bin && go run ./cmd/loadtest server"
	if err := sshc.StartCommand(cmd); err != nil {
		mlog.Error("error running ssh command: " + err.Error())
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
	dstPath := "/home/ubuntu/mattermost-load-test-ng/config/coordinator.json"
	if out, err := sshc.Upload(bytes.NewReader(data), dstPath, false); err != nil {
		return fmt.Errorf("error running ssh command: output: %s, error: %w", out, err)
	}

	// TODO: This is a hack to overcome an issue with go run command.
	mlog.Info("Compiling coordinator", mlog.String("ip", ip))
	cmd := "cd mattermost-load-test-ng && export PATH=$PATH:/usr/local/go/bin && go build -o lt-coordinator ./cmd/coordinator"
	if out, err := sshc.RunCommand(cmd); err != nil {
		mlog.Error("error running ssh command: ", mlog.String("output", string(out)), mlog.Err(err))
	}
	mlog.Info("Starting coordinator", mlog.String("ip", ip))
	if err := sshc.StartCommand("cd mattermost-load-test-ng && ./lt-coordinator"); err != nil {
		mlog.Error("error starting command: " + err.Error())
	}

	return nil
}
