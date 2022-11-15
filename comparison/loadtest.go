// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package comparison

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/v6/shared/mlog"
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

func initLoadTest(t *terraform.Terraform, buildCfg BuildConfig, dumpFilename string, s3BucketURI string, cancelCh <-chan struct{}) error {
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

	type cmd struct {
		msg     string
		value   string
		clients []*ssh.Client
	}

	type localCmd struct {
		msg   string
		value []string
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

	dbName := dpConfig.DBName()
	resetCmd := cmd{
		msg:     "Resetting database",
		clients: []*ssh.Client{appClients[0]},
	}
	switch dpConfig.TerraformDBSettings.InstanceEngine {
	case "aurora-postgresql":
		sqlConnParams := fmt.Sprintf("-U %s -h %s %s", dpConfig.TerraformDBSettings.UserName, tfOutput.DBWriter(), dbName)
		resetCmd.value = strings.Join([]string{
			fmt.Sprintf("export PGPASSWORD='%s'", dpConfig.TerraformDBSettings.Password),
			fmt.Sprintf("dropdb %s", sqlConnParams),
			fmt.Sprintf("createdb %s", sqlConnParams),
			fmt.Sprintf("psql %s -c 'ALTER DATABASE %s SET default_text_search_config TO \"pg_catalog.english\"'", sqlConnParams, dbName),
		}, " && ")
	case "aurora-mysql":
		subCmd := fmt.Sprintf("mysqladmin -h %s -u %s -p%s -f", tfOutput.DBWriter(), dpConfig.TerraformDBSettings.UserName, dpConfig.TerraformDBSettings.Password)
		resetCmd.value = fmt.Sprintf("%s drop %s && %s create %s", subCmd, dbName, subCmd, dbName)
	default:
		return fmt.Errorf("invalid db engine %s", dpConfig.TerraformDBSettings.InstanceEngine)
	}

	startCmd := cmd{
		msg:     "Restarting app server",
		value:   "sudo systemctl start mattermost && until $(curl -sSf http://localhost:8065 --output /dev/null); do sleep 1; done;",
		clients: appClients,
	}

	// do init process
	mmctlPath := "/opt/mattermost/bin/mmctl"
	createAdminCmd := cmd{
		msg: "Creating sysadmin",
		value: fmt.Sprintf("%s user create --email %s --username %s --password '%s' --system-admin --local || true",
			mmctlPath, dpConfig.AdminEmail, dpConfig.AdminUsername, dpConfig.AdminPassword),
		clients: []*ssh.Client{appClients[0]},
	}
	initDataCmd := cmd{
		msg:     "Initializing data",
		value:   fmt.Sprintf("cd mattermost-load-test-ng && ./bin/ltagent init --user-prefix '%s' > /dev/null 2>&1", tfOutput.Agents[0].Tags.Name),
		clients: []*ssh.Client{agentClient},
	}

	cmds := []cmd{stopCmd, installCmd, resetCmd}

	loadDBDumpCmd := cmd{
		msg:     "Loading DB dump",
		clients: []*ssh.Client{appClients[0]},
	}
	switch dpConfig.TerraformDBSettings.InstanceEngine {
	case "aurora-postgresql":
		loadDBDumpCmd.value = fmt.Sprintf("zcat %s | psql 'postgres://%s:%s@%s/%s?sslmode=disable'", dumpFilename,
			dpConfig.TerraformDBSettings.UserName, dpConfig.TerraformDBSettings.Password, tfOutput.DBWriter(), dbName)
	case "aurora-mysql":
		loadDBDumpCmd.value = fmt.Sprintf("zcat %s | mysql -h %s -u %s -p%s %s", dumpFilename,
			tfOutput.DBWriter(), dpConfig.TerraformDBSettings.UserName, dpConfig.TerraformDBSettings.Password, dbName)
	default:
		return fmt.Errorf("invalid db engine %s", dpConfig.TerraformDBSettings.InstanceEngine)
	}

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

	// Resetting the buckets can happen concurrently with the rest of the remote commands
	resetBucketErrCh := make(chan error)
	go func() {
		for _, c := range resetBucketCmds {
			mlog.Info(c.msg)
			if err := exec.Command("aws", c.value...).Run(); err != nil {
				resetBucketErrCh <- fmt.Errorf("failed to run local cmd %q: %w", c.value, err)
				return
			}
		}
		resetBucketErrCh <- nil
	}()

	for _, c := range cmds {
		mlog.Info(c.msg)
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
