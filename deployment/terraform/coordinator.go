package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/agent"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost-server/v5/mlog"
)

// StartCoordinator starts the coordinator in the current load-test deployment.
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

	var loadAgentConfigs []agent.LoadAgentConfig
	for _, val := range output.Agents.Value {
		loadAgentConfigs = append(loadAgentConfigs, agent.LoadAgentConfig{
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

	data, err := json.MarshalIndent(coordinatorConfig, "", "  ")
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

// StopCoordinator stops the coordinator in the current load-test deployment.
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
