package terraform

import (
	"encoding/json"
	"fmt"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/agent"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-server/v5/mlog"
)

func (t *Terraform) generateLoadtestAgentConfig(output *terraformOutput) loadtest.Config {
	return loadtest.Config{
		ConnectionConfiguration: loadtest.ConnectionConfiguration{
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

	srcPath := "$HOME/mattermost-load-test-ng-" + t.config.SourceCodeRef

	// Populate the DB.
	mlog.Info("Populating DB")
	cmd := fmt.Sprintf("cd %s && go run ./cmd/loadtest init", srcPath)
	if err := sshc.RunCommand(cmd); err != nil {
		return err
	}

	// Starting coordinator.
	mlog.Info("Starting coordinator", mlog.String("ip", ip))
	cmd = fmt.Sprintf("cd %s && go run ./cmd/loadtest coordinator &", srcPath)
	if err := sshc.RunCommand(cmd); err != nil {
		return err
	}

	return nil
}

func (t *Terraform) runAgent(extAgent *ssh.ExtAgent, ip string) error {
	sshc, err := extAgent.NewClient(ip)
	if err != nil {
		return err
	}

	srcPath := "$HOME/mattermost-load-test-ng-" + t.config.SourceCodeRef

	// Starting agent.
	mlog.Info("Starting agent", mlog.String("ip", ip))
	cmd := fmt.Sprintf("cd %s && go run ./cmd/loadtest server &", srcPath)
	if err := sshc.RunCommand(cmd); err != nil {
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

	srcPath := "$HOME/mattermost-load-test-ng-" + t.config.SourceCodeRef
	data, err := json.Marshal(loadAgentConfigs)
	if err != nil {
		return err
	}
	for k, v := range map[string]interface{}{
		"ClusterConfig.Agents": string(data),
		".PrometheusURL":       output.MetricsServer.Value.PrivateIP,
	} {
		buf, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("invalid config: key: %s, err: %v", k, err)
		}
		cmd := fmt.Sprintf(`jq '%s = %s' %s/config/config.json > /tmp/agent.json && mv /tmp/agent.json %s/config/config.json`, k, string(buf), srcPath, srcPath)
		fmt.Println(cmd)
		if err := sshc.RunCommand(cmd); err != nil {
			return fmt.Errorf("error running ssh command: cmd: %s, err: %v", cmd, err)
		}
	}
	return nil
}
