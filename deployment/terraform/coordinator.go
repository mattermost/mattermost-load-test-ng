package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"
	"github.com/mattermost/mattermost-server/v5/mlog"
)

// StartCoordinator starts the coordinator in the current load-test deployment.
func (t *Terraform) StartCoordinator() error {
	if err := t.preFlightCheck(); err != nil {
		return err
	}

	output, err := t.Output()
	if err != nil {
		return err
	}

	if len(output.Agents.Value) == 0 {
		return fmt.Errorf("there are no agent instances to run the coordinator")
	}
	ip := output.Agents.Value[0].PublicIP

	var loadAgentConfigs []cluster.LoadAgentConfig
	for _, val := range output.Agents.Value {
		loadAgentConfigs = append(loadAgentConfigs, cluster.LoadAgentConfig{
			Id:     val.Tags.Name,
			ApiURL: "http://" + val.PrivateIP + ":4000",
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

	coordinatorConfig, err := coordinator.ReadConfig("")
	if err != nil {
		return err
	}
	coordinatorConfig.ClusterConfig.Agents = loadAgentConfigs
	coordinatorConfig.MonitorConfig.PrometheusURL = "http://" + output.MetricsServer.Value.PrivateIP + ":9090"

	data, err := json.MarshalIndent(coordinatorConfig, "", "  ")
	if err != nil {
		return err
	}
	mlog.Info("Uploading updated coordinator config file")
	dstPath := "/home/ubuntu/mattermost-load-test-ng/config/coordinator.json"
	if out, err := sshc.Upload(bytes.NewReader(data), dstPath, false); err != nil {
		return fmt.Errorf("error running ssh command: output: %s, error: %w", out, err)
	}

	mlog.Info("Uploading other load-test config files")

	agentConfig, err := t.generateLoadtestAgentConfig(output)
	if err != nil {
		return err
	}

	simulConfig, err := simulcontroller.ReadConfig("")
	if err != nil {
		return err
	}

	simpleConfig, err := simplecontroller.ReadConfig("")
	if err != nil {
		return err
	}

	batch := []struct {
		input   interface{}
		dstPath string
	}{
		{
			input:   agentConfig,
			dstPath: "/home/ubuntu/mattermost-load-test-ng/config/config.json",
		},
		{
			input:   simulConfig,
			dstPath: "/home/ubuntu/mattermost-load-test-ng/config/simulcontroller.json",
		},
		{
			input:   simpleConfig,
			dstPath: "/home/ubuntu/mattermost-load-test-ng/config/simplecontroller.json",
		},
	}

	for _, info := range batch {
		data, err := json.MarshalIndent(info.input, "", "  ")
		if err != nil {
			return err
		}

		mlog.Info(info.dstPath)
		if out, err := sshc.Upload(bytes.NewReader(data), info.dstPath, false); err != nil {
			return fmt.Errorf("error uploading file, dstPath: %s, output: %q: %w", info.dstPath, out, err)
		}
	}

	mlog.Info("Starting the coordinator")
	cmd := "sudo service ltcoordinator start"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: output: %q, error: %w", out, err)
	}

	mlog.Info("Done")
	return nil
}

// StopCoordinator stops the coordinator in the current load-test deployment.
func (t *Terraform) StopCoordinator() error {
	if err := t.preFlightCheck(); err != nil {
		return err
	}

	output, err := t.Output()
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
