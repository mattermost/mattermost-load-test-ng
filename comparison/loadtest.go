// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package comparison

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
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

func runBoundedLoadTest(t *terraform.Terraform, coordConfig *coordinator.Config,
	d time.Duration, cancelCh <-chan struct{}) (coordinator.Status, error) {
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

	mlog.Info("bounded load-test has completed")

	// TODO: remove this once MM-30326 has been merged and a new release
	// published.
	status.StopTime = time.Now()

	return status, nil
}

func runUnboundedLoadTest(t *terraform.Terraform, coordConfig *coordinator.Config,
	cancelCh <-chan struct{}) (coordinator.Status, error) {
	var err error
	var status coordinator.Status

	mlog.Info("starting unbounded load-test")
	if err := t.StartCoordinator(coordConfig); err != nil {
		return status, err
	}

	defer func() {
		if _, err := t.StopCoordinator(); err != nil {
			mlog.Error("stopping coordinator failed", mlog.Err(err))
		}
	}()

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		status, err = t.GetCoordinatorStatus()
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
			return status, errors.New("canceled")
		case <-ticker.C:
		}
	}
}

type localCmd struct {
	msg   string
	value []string
}

func initLoadTest(t *terraform.Terraform, buildCfg BuildConfig, dumpFilename string, s3BucketURI string, permalinkIPsToReplace []string, cancelCh <-chan struct{}) error {
	tfOutput, err := t.Output()
	if err != nil {
		return fmt.Errorf("failed to get terraform output: %w", err)
	}

	dpConfig := t.Config()

	if !tfOutput.HasAppServers() {
		return errors.New("no app servers in this deployment")
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}

	agentClient, err := extAgent.NewClient(tfOutput.Agents[0].PublicIP)
	if err != nil {
		return fmt.Errorf("error in getting ssh connection %w", err)
	}
	defer agentClient.Close()

	appClients := make([]*ssh.Client, len(tfOutput.Instances))
	for i, instance := range tfOutput.Instances {
		client, err := extAgent.NewClient(instance.PublicIP)
		if err != nil {
			return fmt.Errorf("error in getting ssh connection %w", err)
		}
		defer client.Close()
		appClients[i] = client
	}

	stopCmd := deployment.Cmd{
		Msg:     "Stopping app servers",
		Value:   "sudo systemctl stop mattermost",
		Clients: appClients,
	}

	buildFileName := filepath.Base(buildCfg.URL)
	installCmd := deployment.Cmd{
		Msg:     "Installing app",
		Value:   fmt.Sprintf("cd /home/ubuntu && tar xzf %s && cp /opt/mattermost/config/config.json . && sudo rm -rf /opt/mattermost && sudo mv mattermost /opt/ && mv config.json /opt/mattermost/config/", buildFileName),
		Clients: appClients,
	}

	dbName := dpConfig.DBName()
	resetCmd := deployment.Cmd{
		Msg:     "Resetting database",
		Clients: []*ssh.Client{appClients[0]},
	}
	switch dpConfig.TerraformDBSettings.InstanceEngine {
	case "aurora-postgresql":
		sqlConnParams := fmt.Sprintf("-U %s -h %s %s", dpConfig.TerraformDBSettings.UserName, tfOutput.DBWriter(), dbName)
		resetCmd.Value = strings.Join([]string{
			fmt.Sprintf("export PGPASSWORD='%s'", dpConfig.TerraformDBSettings.Password),
			fmt.Sprintf("dropdb %s", sqlConnParams),
			fmt.Sprintf("createdb %s", sqlConnParams),
			fmt.Sprintf("psql %s -c 'ALTER DATABASE %s SET default_text_search_config TO \"pg_catalog.english\"'", sqlConnParams, dbName),
		}, " && ")
	case "aurora-mysql":
		subCmd := fmt.Sprintf("mysqladmin -h %s -u %s -p%s -f", tfOutput.DBWriter(), dpConfig.TerraformDBSettings.UserName, dpConfig.TerraformDBSettings.Password)
		resetCmd.Value = fmt.Sprintf("%s drop %s && %s create %s", subCmd, dbName, subCmd, dbName)
	default:
		return fmt.Errorf("invalid db engine %s", dpConfig.TerraformDBSettings.InstanceEngine)
	}

	startCmd := deployment.Cmd{
		Msg:     "Restarting app server",
		Value:   "sudo systemctl start mattermost && until $(curl -sSf http://localhost:8065 --output /dev/null); do sleep 1; done;",
		Clients: appClients,
	}

	// do init process
	mmctlPath := "/opt/mattermost/bin/mmctl"
	createAdminCmd := deployment.Cmd{
		Msg: "Creating sysadmin",
		Value: fmt.Sprintf("%s user create --email %s --username %s --password '%s' --system-admin --local || true",
			mmctlPath, dpConfig.AdminEmail, dpConfig.AdminUsername, dpConfig.AdminPassword),
		Clients: []*ssh.Client{appClients[0]},
	}
	initDataCmd := deployment.Cmd{
		Msg:     "Initializing data",
		Value:   fmt.Sprintf("cd mattermost-load-test-ng && ./bin/ltagent init --user-prefix '%s' > /dev/null 2>&1", tfOutput.Agents[0].Tags.Name),
		Clients: []*ssh.Client{agentClient},
	}

	cmds := []deployment.Cmd{stopCmd, installCmd, resetCmd}

	loadDBDumpCmd := deployment.Cmd{
		Msg:     "Loading DB dump",
		Clients: []*ssh.Client{appClients[0]},
	}

	dbCmds, err := deployment.BuildLoadDBDumpCmds(dumpFilename, tfOutput.PermalinksIPsSubstCommand(permalinkIPsToReplace), deployment.DBSettings{
		UserName: dpConfig.TerraformDBSettings.UserName,
		Password: dpConfig.TerraformDBSettings.Password,
		DBName:   dbName,
		Host:     tfOutput.DBWriter(),
		Engine:   dpConfig.TerraformDBSettings.InstanceEngine,
	})
	if err != nil {
		return fmt.Errorf("error building commands for loading DB dump: %w", err)
	}

	loadDBDumpCmd.Value = strings.Join(dbCmds, " | ")

	resetBucketCmds := []localCmd{}
	if s3BucketURI != "" && tfOutput.HasS3Bucket() {
		deleteBucketCmd := localCmd{
			msg:   "Emptying S3 bucket",
			value: []string{"--profile", t.Config().AWSProfile, "s3", "rm", "s3://" + tfOutput.S3Bucket.Id, "--recursive"},
		}

		prepopulateBucketCmd := localCmd{
			msg:   "Pre-populating S3 bucket",
			value: []string{"--profile", t.Config().AWSProfile, "s3", "cp", s3BucketURI, "s3://" + tfOutput.S3Bucket.Id, "--recursive"},
		}

		resetBucketCmds = []localCmd{deleteBucketCmd, prepopulateBucketCmd}
	}

	if dumpFilename == "" {
		cmds = append(cmds, startCmd, createAdminCmd, initDataCmd)
	} else {
		cmds = append(cmds, loadDBDumpCmd, startCmd)
	}

	// Resetting the buckets can happen concurrently with the rest of the remote commands,
	// but we need to cancel them on return in case we return early (as when we receive from cancelCh)
	resetBucketCtx, resetBucketCancel := context.WithCancel(context.Background())
	defer resetBucketCancel()
	resetBucketErrCh := make(chan error, 1)
	go func() {
		for _, c := range resetBucketCmds {
			mlog.Info(c.msg)
			if err := exec.CommandContext(resetBucketCtx, "aws", c.value...).Run(); err != nil {
				resetBucketErrCh <- fmt.Errorf("failed to run local cmd %q: %w", c.value, err)
				return
			}
		}
		resetBucketErrCh <- nil
	}()

	for _, c := range cmds {
		mlog.Info(c.Msg)
		for _, client := range c.Clients {
			select {
			case <-cancelCh:
				mlog.Info("cancelling load-test init")
				return errors.New("canceled")
			default:
			}
			if out, err := client.RunCommand(c.Value); err != nil {
				return fmt.Errorf("failed to run cmd %q: %w %s", c.Value, err, out)
			}
		}
	}

	// Make sure that the S3 bucket reset routine is finished and return its error, if any
	return <-resetBucketErrCh
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
