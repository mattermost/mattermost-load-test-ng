package terraform

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/agent"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-server/v5/mlog"
)

func (t *Terraform) generateLoadtestAgentConfig(output *terraformOutput) loadtest.Config {
	return loadtest.Config{
		ConnectionConfiguration: loadtest.ConnectionConfiguration{
			// TODO: replace with reverse nginx ip
			ServerURL:                   "http://" + output.Instances.Value[0].PrivateIP + ":8065",
			WebSocketURL:                "ws://" + output.Instances.Value[0].PrivateIP + ":8065",
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
	}
}

func (t *Terraform) startCoordinator(extAgent *ssh.ExtAgent, ip string) error {
	sshc, err := extAgent.NewClient(ip)
	if err != nil {
		return err
	}

	// Populate the DB.
	mlog.Info("Populating DB")
	cmd := fmt.Sprintf("agent init")
	if err := sshc.RunCommand(cmd); err != nil {
		mlog.Error(cmd)
		return err
	}

	// Starting coordinator.
	mlog.Info("Starting coordinator", mlog.String("ip", ip))
	if err := sshc.RunCommand("agent coordinator &"); err != nil {
		mlog.Error(cmd)
		return err
	}

	return nil
}

func (t *Terraform) runAgent(extAgent *ssh.ExtAgent, ip string) error {
	sshc, err := extAgent.NewClient(ip)
	if err != nil {
		return err
	}

	// Starting agent.
	mlog.Info("Starting agent", mlog.String("ip", ip))
	if err := sshc.RunCommand("agent server &"); err != nil {
		return err
	}
	return nil
}

func (t *Terraform) updateCoordinatorConfig(extAgent *ssh.ExtAgent, output *terraformOutput) error {
	loadtestConfig := t.generateLoadtestAgentConfig(output)

	var loadAgentConfigs []agent.LoadAgentConfig
	for i := 1; i < len(output.Agents.Value); i++ {
		val := output.Agents.Value[i]
		loadAgentConfigs = append(loadAgentConfigs, agent.LoadAgentConfig{
			Id:             val.Tags.Name,
			ApiURL:         val.PrivateIP + ":4000",
			LoadTestConfig: loadtestConfig,
		})
	}

	sshc, err := extAgent.NewClient(output.Agents.Value[0].PublicIP)
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

	data, err := json.Marshal(loadtestConfig)
	if err != nil {
		return err
	}
	dstPath := "/home/ubuntu/config.json"
	if err := sshc.Upload(strings.NewReader(string(data)), dstPath, false); err != nil {
		return fmt.Errorf("error running ssh command: %w", err)
	}

	data, err = json.Marshal(clusterConfig)
	if err != nil {
		return err
	}
	dstPath = "/home/ubuntu/coordinator.json"
	if err := sshc.Upload(strings.NewReader(string(data)), dstPath, false); err != nil {
		return fmt.Errorf("error running ssh command: %w", err)
	}

	return nil
}
