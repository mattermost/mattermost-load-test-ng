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
	"github.com/mattermost/mattermost-server/v5/mlog"
)

// Start starts the coordinator that is deployed.
func (t *Terraform) StartCoordinator() error {
	if err := t.preFlightCheck(); err != nil {
		return err
	}

	output, err := t.getOutput()
	if err != nil {
		return err
	}

	if len(output.Agents.Value) == 0 {
		return fmt.Errorf("there are no agent instances to run the coordinator")
	}
	ip := output.Agents.Value[0].PublicIP

	loadtestConfig := t.generateLoadtestAgentConfig(output)

	var loadAgentConfigs []agent.LoadAgentConfig
	for _, val := range output.Agents.Value {
		loadAgentConfigs = append(loadAgentConfigs, agent.LoadAgentConfig{
			Id:             val.Tags.Name,
			ApiURL:         "http://" + val.PrivateIP + ":4000",
			LoadTestConfig: loadtestConfig,
		})
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}
	sshc, err := extAgent.NewClient(ip)
	if err != nil {
		return err
	}

	mlog.Info("Setting up coordinator", mlog.String("ip", ip))
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

	data, err := json.MarshalIndent(clusterConfig, "", "  ")
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
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: output: %q, error: %w", out, err)
	}

	mlog.Info("Done")
	return nil
}

func (t *Terraform) StopCoordinator() error {
	if err := t.preFlightCheck(); err != nil {
		return err
	}

	output, err := t.getOutput()
	if err != nil {
		return err
	}

	if len(output.Agents.Value) == 0 {
		return fmt.Errorf("there are no agents to initialize load-test")
	}
	ip := output.Agents.Value[0].PublicIP

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}
	sshc, err := extAgent.NewClient(ip)
	if err != nil {
		return err
	}

	mlog.Info("Stopping the coordinator", mlog.String("ip", ip))
	cmd := "sudo service ltcoordinator stop"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: output: %q, error: %w", out, err)
	}

	mlog.Info("Done")
	return nil
}
