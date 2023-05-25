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
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/server/v8/platform/shared/mlog"
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

type cmd struct {
	msg     string
	value   string
	clients []*ssh.Client
}

type localCmd struct {
	msg   string
	value []string
}

type dbSettings struct {
	UserName string
	Password string
	DBName   string
	Host     string
	Engine   string
}

// buildLoadDBDumpCmds returns a slice of commands that, when piped, feed the
// provided DB dump file into the database, replacing first the old IPs found
// in the posts that contain a permalink with the new IP. Something like:
//
//	zcat dbdump.sql
//	sed -r -e 's/old_ip_1/new_ip' -e 's/old_ip_2/new_ip'
//	mysql/psql connection_details
func buildLoadDBDumpCmds(dumpFilename string, newIP string, permalinkIPsToReplace []string, dbInfo dbSettings) ([]string, error) {
	zcatCmd := fmt.Sprintf("zcat %s", dumpFilename)

	var replacements []string
	for _, oldIP := range permalinkIPsToReplace {
		// Let's build the match and replace parts of a sed command: 's/match/replace/g'
		// First, the match. We want to match anything of the form
		//    54.126.54.26:8065/debitis-1/pl/
		// where the IP is exactly the old one, the port is optional and arbitrary and the
		// team name is the pattern defined by the server's function model.IsValidTeamname
		validTeamName := `[a-z0-9]+([a-z0-9-]+|(__)?)[a-z0-9]+`
		escapedOldIP := strings.ReplaceAll(oldIP, ".", "\\.")
		match := escapedOldIP + `(:[0-9]+)?\/(` + validTeamName + `)\/pl\/`
		// Now, the replace. We need to replace this with the same thing, only changing the
		// IP with the new one and hard-coding the port to 8065, but maintaining the team
		// name (hence the second group match, \2)
		replace := newIP + `:8065\/\2\/pl\/`
		// We can build the whole command now and add it to the list of replacements
		sedRegex := fmt.Sprintf(`'s/%s/%s/g'`, match, replace)
		replacements = append(replacements, sedRegex)
	}
	var sedCmd string
	if len(replacements) > 0 {
		sedCmd = strings.Join(append([]string{"sed -r"}, replacements...), " -e ")
	}

	var dbCmd string
	switch dbInfo.Engine {
	case "aurora-postgresql":
		dbCmd = fmt.Sprintf("psql 'postgres://%[1]s:%[2]s@%[3]s/%[4]s?sslmode=disable'", dbInfo.UserName, dbInfo.Password, dbInfo.Host, dbInfo.DBName)
	case "aurora-mysql":
		dbCmd = fmt.Sprintf("mysql -h %[1]s -u %[2]s -p%[3]s %[4]s", dbInfo.Host, dbInfo.UserName, dbInfo.Password, dbInfo.DBName)
	default:
		return []string{}, fmt.Errorf("invalid db engine %s", dbInfo.Engine)
	}

	if sedCmd != "" {
		return []string{zcatCmd, sedCmd, dbCmd}, nil
	}

	return []string{zcatCmd, dbCmd}, nil
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

	dbCmds, err := buildLoadDBDumpCmds(dumpFilename, tfOutput.Instances[0].PublicIP, permalinkIPsToReplace, dbSettings{
		UserName: dpConfig.TerraformDBSettings.UserName,
		Password: dpConfig.TerraformDBSettings.Password,
		DBName:   dbName,
		Host:     tfOutput.DBWriter(),
		Engine:   dpConfig.TerraformDBSettings.InstanceEngine,
	})
	if err != nil {
		return fmt.Errorf("error building commands for loading DB dump: %w", err)
	}

	loadDBDumpCmd.value = strings.Join(dbCmds, " | ")

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
