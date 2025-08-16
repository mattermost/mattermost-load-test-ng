package terraform

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	client "github.com/mattermost/mattermost-load-test-ng/api/client/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

const ltAPIPort = "4000"

// StartCoordinator starts the coordinator in the current load-test deployment.
func (t *Terraform) StartCoordinator(config *coordinator.Config) error {
	if err := t.preFlightCheck(); err != nil {
		return err
	}

	if err := t.setOutput(); err != nil {
		return err
	}

	if len(t.output.Agents) == 0 {
		return errors.New("there are no agent instances to run the coordinator")
	}

	// Coordinator resides in the first agent instance
	coordinatorIP := t.output.Agents[0].GetConnectionIP()

	var loadAgentConfigs []cluster.LoadAgentConfig
	for _, val := range t.output.Agents {
		loadAgentConfigs = append(loadAgentConfigs, cluster.LoadAgentConfig{
			Id:     val.Tags.Name,
			ApiURL: "http://" + val.GetConnectionIP() + ":" + ltAPIPort,
		})
	}

	// Notice we are not passing the port of LTBrowser API server here
	// but its the LT API port since LTBrowser API will be called from LT API server and not directly from coordinator
	var browserAgentConfigs []cluster.LoadAgentConfig
	for _, val := range t.output.BrowserAgents {
		browserAgentConfigs = append(browserAgentConfigs, cluster.LoadAgentConfig{
			Id:     val.Tags.Name,
			ApiURL: "http://" + val.GetConnectionIP() + ":" + ltAPIPort,
		})
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}

	sshClientCoordinator, err := extAgent.NewClient(t.Config().AWSAMIUser, coordinatorIP)
	if err != nil {
		return err
	}
	defer sshClientCoordinator.Close()

	mlog.Info("Setting up coordinator", mlog.String("ip", coordinatorIP))

	if config == nil {
		config, err = coordinator.ReadConfig("")
		if err != nil {
			return err
		}
	}
	config.ClusterConfig.Agents = loadAgentConfigs
	config.ClusterConfig.BrowserAgents = browserAgentConfigs
	config.MonitorConfig.PrometheusURL = "http://" + t.output.MetricsServer.GetConnectionIP() + ":9090"

	// TODO: consider removing this. Config is passed dynamically when creating
	// a coordinator resource through the API.
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	mlog.Info("Uploading updated coordinator config file")
	dstPath := fmt.Sprintf("/home/%s/mattermost-load-test-ng/config/coordinator.json", t.config.AWSAMIUser)
	if out, err := sshClientCoordinator.Upload(bytes.NewReader(data), dstPath, false); err != nil {
		return fmt.Errorf("error running ssh command: output: %s, error: %w", out, err)
	}

	mlog.Info("Uploading other load-test config files")

	var agentConfig *loadtest.Config
	if len(t.output.Instances) > 0 {
		agentConfig, err = t.generateLoadtestAgentConfig()
	} else {
		agentConfig, err = loadtest.ReadConfig("")
	}
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	simulConfig, err := simulcontroller.ReadConfig("")
	if err != nil {
		return err
	}

	simpleConfig, err := simplecontroller.ReadConfig("")
	if err != nil {
		return err
	}

	batchForControllerConfigs := []struct {
		input   interface{}
		dstPath string
	}{
		{
			input:   agentConfig,
			dstPath: fmt.Sprintf("/home/%s/mattermost-load-test-ng/config/config.json", t.Config().AWSAMIUser),
		},
		{
			input:   simulConfig,
			dstPath: fmt.Sprintf("/home/%s/mattermost-load-test-ng/config/simulcontroller.json", t.Config().AWSAMIUser),
		},
		{
			input:   simpleConfig,
			dstPath: fmt.Sprintf("/home/%s/mattermost-load-test-ng/config/simplecontroller.json", t.Config().AWSAMIUser),
		},
	}

	// Upload config files to the coordinator instance
	for _, info := range batchForControllerConfigs {
		data, err := json.MarshalIndent(info.input, "", "  ")
		if err != nil {
			return err
		}

		mlog.Info("Uploading config file to instance", mlog.String("file", info.dstPath))
		if out, err := sshClientCoordinator.Upload(bytes.NewReader(data), info.dstPath, false); err != nil {
			return fmt.Errorf("error uploading file to instance, dstPath: %s, output: %q: %w", info.dstPath, out, err)
		}
	}

	mlog.Info("Starting the coordinator")

	id := t.config.ClusterName + "-coordinator-0"
	coord, err := client.New(id, "http://"+coordinatorIP+":4000", nil)
	if err != nil {
		return fmt.Errorf("failed to create coordinator client: %w", err)
	}
	if st, err := coord.Status(); err == nil && st.State == coordinator.Done {
		mlog.Info("coordinator exists and its state is done, destroying", mlog.String("status", fmt.Sprintf("%+v", st)))
		if _, err := coord.Destroy(); err != nil {
			return fmt.Errorf("failed to destroy coordinator: %w", err)
		}
	}
	if _, err := coord.Create(config, agentConfig); err != nil {
		return fmt.Errorf("failed to create coordinator: %w", err)
	}
	if _, err := coord.Run(); err != nil {
		return fmt.Errorf("failed to start coordinator: %w", err)
	}

	mlog.Info("Done")
	return nil
}

// StopCoordinator stops the coordinator in the current load-test deployment.
func (t *Terraform) StopCoordinator() (coordinator.Status, error) {
	var status coordinator.Status

	if err := t.setOutput(); err != nil {
		return status, err
	}

	if len(t.output.Agents) == 0 {
		return status, errors.New("there are no agents to initialize load-test")
	}
	ip := t.output.Agents[0].GetConnectionIP()

	mlog.Info("Stopping the coordinator", mlog.String("ip", ip))

	id := fmt.Sprintf("%s-coordinator-%d", t.config.ClusterName, 0)
	coord, err := client.New(id, "http://"+ip+":4000", nil)
	if err != nil {
		return status, fmt.Errorf("failed to create coordinator client: %w", err)
	}

	status, err = coord.Destroy()
	if err != nil {
		return status, fmt.Errorf("failed to stop coordinator: %w", err)
	}

	mlog.Info("Done")
	return status, nil
}

// GetCoordinatorStatus returns information about the status of the
// coordinator in the current load-test deployment.
func (t *Terraform) GetCoordinatorStatus() (coordinator.Status, error) {
	var status coordinator.Status

	if err := t.setOutput(); err != nil {
		return status, err
	}

	if len(t.output.Agents) == 0 {
		return status, errors.New("there are no agents to initialize load-test")
	}
	ip := t.output.Agents[0].GetConnectionIP()

	id := t.config.ClusterName + "-coordinator-0"
	coord, err := client.New(id, "http://"+ip+":4000", nil)
	if err != nil {
		return status, fmt.Errorf("failed to create coordinator client: %w", err)
	}
	status, err = coord.Status()
	if err != nil {
		return status, fmt.Errorf("failed to get coordinator status: %w", err)
	}

	return status, nil
}

// InjectAction injects a named action for all agents that is run once.
func (t *Terraform) InjectAction(actionID string) (coordinator.Status, error) {
	var status coordinator.Status

	if err := t.setOutput(); err != nil {
		return status, err
	}

	if len(t.output.Agents) == 0 {
		return status, errors.New("there are no agents to inject the action")
	}
	ip := t.output.Agents[0].GetConnectionIP()

	id := t.config.ClusterName + "-coordinator-0"
	coord, err := client.New(id, "http://"+ip+":4000", nil)
	if err != nil {
		return status, fmt.Errorf("failed to create coordinator client: %w", err)
	}

	status, err = coord.InjectAction(actionID)
	if err != nil {
		return status, fmt.Errorf("failed to inject action %q: %w", actionID, err)
	}

	return status, nil
}
