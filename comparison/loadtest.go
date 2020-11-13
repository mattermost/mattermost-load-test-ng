// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package comparison

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

func (c *Comparison) getLoadTestsCount() int {
	var count int
	for _, dp := range c.deployments {
		for range dp.loadTests {
			count++
		}
	}
	return count
}

func runBoundedLoadTest(t *terraform.Terraform, coordConfig *coordinator.Config, d time.Duration, cancelCh <-chan struct{}) (coordinator.Status, error) {
	var err error
	var status coordinator.Status
	mlog.Info("starting bounded load-test")
	if err := t.StartCoordinator(coordConfig); err != nil {
		return status, err
	}

	var canceled bool
	select {
	case <-cancelCh:
		mlog.Info("cancelling load-test")
		canceled = true
	case <-time.After(d):
	}

	mlog.Info("stopping bounded load-test")
	status, err = t.StopCoordinator()
	if err != nil {
		return status, err
	}

	if canceled {
		return status, errors.New("canceled")
	}

	// TODO: remove this once MM-30326 has been merged and a new release
	// published.
	status.StopTime = time.Now()

	return status, nil
}

func runUnboundedLoadTest(t *terraform.Terraform, coordConfig *coordinator.Config, cancelCh <-chan struct{}) (coordinator.Status, error) {
	mlog.Info("starting unbounded load-test")
	if err := t.StartCoordinator(coordConfig); err != nil {
		return coordinator.Status{}, err
	}

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		status, err := t.GetCoordinatorStatus()
		if err != nil {
			return status, err
		}

		if status.State == coordinator.Done {
			mlog.Info("unbounded load-test has completed")
			return status, nil
		}

		if status.State != coordinator.Running {
			return status, errors.New("coordinator should be running")
		}

		select {
		case <-cancelCh:
			mlog.Info("cancelling load-test")
			if status, err := t.StopCoordinator(); err != nil {
				return status, err
			}
			return coordinator.Status{}, errors.New("canceled")
		case <-ticker.C:
		}
	}
}

func initLoadTest(t *terraform.Terraform, config *deployment.Config, buildCfg BuildConfig, cancelCh <-chan struct{}) error {
	output, err := t.Output()
	if err != nil {
		return err
	}

	if !output.HasAppServers() {
		return errors.New("no app servers in this deployment")
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}

	agentClient, err := extAgent.NewClient(output.Agents[0].PublicIP)
	if err != nil {
		return fmt.Errorf("error in getting ssh connection %w", err)
	}
	defer agentClient.Close()

	appClients := make([]*ssh.Client, len(output.Instances))
	for i, instance := range output.Instances {
		client, err := extAgent.NewClient(instance.PublicIP)
		if err != nil {
			return fmt.Errorf("error in getting ssh connection %w", err)
		}
		defer client.Close()
		appClients[i] = client
	}

	type cmd struct {
		msg     string
		value   string
		clients []*ssh.Client
	}

	stopCmd := cmd{
		msg:     "Stopping app servers",
		value:   "sudo systemctl stop mattermost",
		clients: appClients,
	}

	buildFileName := filepath.Base(buildCfg.URL)
	installCmd := cmd{
		msg:     "Installing app",
		value:   fmt.Sprintf("cd /home/ubuntu && tar xzf %s && cp /opt/mattermost/config/config.json . && sudo rm -rf /opt/mattermost && sudo mv mattermost /opt/ && mv config.json /opt/mattermost/config/", buildFileName),
		clients: appClients,
	}

	binaryPath := "/opt/mattermost/bin/mattermost"
	resetCmd := cmd{
		msg:     "Resetting database",
		value:   fmt.Sprintf("%s reset --confirm", binaryPath),
		clients: []*ssh.Client{appClients[0]},
	}

	startCmd := cmd{
		msg:     "Restarting app server",
		value:   fmt.Sprintf("sudo systemctl start mattermost && until $(curl -sSf http://localhost:8065 --output /dev/null); do sleep 1; done;"),
		clients: appClients,
	}

	// do init process
	createAdminCmd := cmd{
		msg: "Creating sysadmin",
		value: fmt.Sprintf("%s user create --email %s --username %s --password '%s' --system_admin || true",
			binaryPath, config.AdminEmail, config.AdminUsername, config.AdminPassword),
		clients: []*ssh.Client{appClients[0]},
	}
	initDataCmd := cmd{
		msg:     "Initializing data",
		value:   fmt.Sprintf("cd mattermost-load-test-ng && ./bin/ltagent init --user-prefix '%s' > /dev/null 2>&1", output.Agents[0].Tags.Name),
		clients: []*ssh.Client{agentClient},
	}

	cmds := []cmd{stopCmd, installCmd, resetCmd, startCmd, createAdminCmd, initDataCmd}
	for _, c := range cmds {
		for _, client := range c.clients {
			select {
			case <-cancelCh:
				mlog.Info("cancelling load-test init")
				return errors.New("canceled")
			default:
			}
			if out, err := client.RunCommand(c.value); err != nil {
				return fmt.Errorf("failed to run cmd %q: %w %s", c.value, err, out)
			}
		}
	}

	return nil
}

func runLoadTest(t *terraform.Terraform, lt LoadTestConfig, cancelCh <-chan struct{}) (coordinator.Status, error) {
	var status coordinator.Status
	coordConfig, err := coordinator.ReadConfig("")
	if err != nil {
		return status, err
	}

	switch lt.Type {
	case LoadTestTypeBounded:
		coordConfig.ClusterConfig.MaxActiveUsers = lt.NumUsers
		// TODO: uncomment this line and remove the loop after a new release is
		// published.
		// coordConfig.MonitorConfig.Queries = nil
		for i := 0; i < len(coordConfig.MonitorConfig.Queries); i++ {
			coordConfig.MonitorConfig.Queries[i].Alert = false
		}
		duration, parseErr := time.ParseDuration(lt.Duration)
		if parseErr != nil {
			return status, parseErr
		}
		return runBoundedLoadTest(t, coordConfig, duration, cancelCh)
	case LoadTestTypeUnbounded:
		// TODO: cleverly set MaxActiveUsers to (numAgents * UsersConfiguration.MaxActiveUsers)
		return runUnboundedLoadTest(t, coordConfig, cancelCh)
	}

	return status, fmt.Errorf("unimplemented LoadTestType %s", lt.Type)
}
