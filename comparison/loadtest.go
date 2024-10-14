// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package comparison

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
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

func runBoundedLoadTest(t *terraform.Terraform, coordConfig *coordinator.Config, d time.Duration) (coordinator.Status, error) {
	var err error
	var status coordinator.Status
	mlog.Info("starting bounded load-test")
	if err := t.StartCoordinator(coordConfig); err != nil {
		return status, err
	}

	// Wait for the specified duration
	<-time.After(d)

	mlog.Info("stopping bounded load-test")
	status, err = t.StopCoordinator()
	if err != nil {
		return status, err
	}

	mlog.Info("bounded load-test has completed")

	return status, nil
}

func runUnboundedLoadTest(t *terraform.Terraform, coordConfig *coordinator.Config) (coordinator.Status, error) {
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
	}
}

type s3ClientWrapper struct {
	S3Client *s3.Client
}

func (s3Wrapper s3ClientWrapper) DeleteObjects(ctx context.Context, bucketName string, objectKeys []string) error {
	if len(objectKeys) == 0 {
		return nil
	}

	var objectIds []types.ObjectIdentifier
	for _, key := range objectKeys {
		objectIds = append(objectIds, types.ObjectIdentifier{Key: aws.String(key)})
	}
	_, err := s3Wrapper.S3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
		Bucket: aws.String(bucketName),
		Delete: &types.Delete{Objects: objectIds},
	})
	if err != nil {
		return fmt.Errorf("error deleting objects from bucket %q: %w", bucketName, err)
	}
	return err
}

func emptyBucket(ctx context.Context, s3Wrapper *s3ClientWrapper, bucketName string) error {
	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	}
	p := s3.NewListObjectsV2Paginator(s3Wrapper.S3Client, params)

	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("error getting the page during empty bucket: %w", err)
		}

		var objKeys []string
		for _, object := range page.Contents {
			objKeys = append(objKeys, *object.Key)
		}

		if err := s3Wrapper.DeleteObjects(ctx, bucketName, objKeys); err != nil {
			return err
		}
	}

	return nil
}

func populateBucket(ctx context.Context, bucket *s3ClientWrapper, targetBucket, sourceBucket string) error {
	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(sourceBucket),
	}
	p := s3.NewListObjectsV2Paginator(bucket.S3Client, params)

	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("error getting the page during populate bucket: %w", err)
		}

		for _, object := range page.Contents {
			_, err := bucket.S3Client.CopyObject(ctx, &s3.CopyObjectInput{
				Bucket:     aws.String(targetBucket),
				CopySource: aws.String(fmt.Sprintf("%s/%s", sourceBucket, *object.Key)),
				Key:        object.Key})

			if err != nil {
				return fmt.Errorf("failed to copy object %q: %w", *object.Key, err)
			}
		}
	}

	return nil
}

func initLoadTest(t *terraform.Terraform, buildCfg BuildConfig, dumpFilename string, s3BucketURI string) error {
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
		Msg: "Initializing data",
		Value: fmt.Sprintf("cd mattermost-load-test-ng && ./bin/ltagent init --user-prefix '%s' --server-url 'http://%s:8065' > /dev/null 2>&1",
			tfOutput.Agents[0].Tags.Name, tfOutput.Instances[0].PrivateIP),
		Clients: []*ssh.Client{agentClient},
	}

	cmds := []deployment.Cmd{stopCmd, installCmd, resetCmd}

	loadDBDumpCmd := deployment.Cmd{
		Msg:     "Loading DB dump",
		Clients: []*ssh.Client{appClients[0]},
	}

	dbInfo := deployment.DBSettings{
		UserName: dpConfig.TerraformDBSettings.UserName,
		Password: dpConfig.TerraformDBSettings.Password,
		DBName:   dbName,
		Host:     tfOutput.DBWriter(),
		Engine:   dpConfig.TerraformDBSettings.InstanceEngine,
	}

	dbCmd, err := deployment.BuildLoadDBDumpCmd(dumpFilename, dbInfo)
	if err != nil {
		return fmt.Errorf("error building command for loading DB dump: %w", err)
	}
	loadDBDumpCmd.Value = dbCmd

	clearLicensesCmdValue, err := deployment.ClearLicensesCmd(dbInfo)
	if err != nil {
		return fmt.Errorf("error building command for clearing licenses data: %w", err)
	}
	clearLicensesCmd := deployment.Cmd{
		Msg:     "Clearing old licenses data",
		Clients: []*ssh.Client{appClients[0]},
		Value:   clearLicensesCmdValue,
	}

	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithSharedConfigProfile(t.Config().AWSProfile))
	if err != nil {
		return fmt.Errorf("error loading aws config: %w", err)
	}

	s3client := s3ClientWrapper{S3Client: s3.NewFromConfig(cfg)}

	if dumpFilename == "" {
		cmds = append(cmds, startCmd, createAdminCmd, initDataCmd)
	} else {
		cmds = append(cmds, loadDBDumpCmd, clearLicensesCmd, startCmd)
	}

	// Resetting the buckets can happen concurrently with the rest of the remote commands
	resetBucketErrCh := make(chan error, 1)
	go func() {
		if s3BucketURI != "" && tfOutput.HasS3Bucket() {
			mlog.Info("Emptying S3 bucket")

			ctx := context.Background()

			err := emptyBucket(ctx, &s3client, tfOutput.S3Bucket.Id)
			if err != nil {
				resetBucketErrCh <- fmt.Errorf("failed to empty s3 bucket: %w", err)
				return
			}

			mlog.Info("Pre-populating S3 bucket")
			srcBucketName := strings.TrimPrefix(s3BucketURI, "s3://")
			err = populateBucket(ctx, &s3client, tfOutput.S3Bucket.Id, srcBucketName)
			if err != nil {
				resetBucketErrCh <- fmt.Errorf("failed to populate bucket: %w", err)
				return
			}
		}

		resetBucketErrCh <- nil
	}()

	for _, c := range cmds {
		mlog.Info(c.Msg)
		for _, client := range c.Clients {
			if out, err := client.RunCommand(c.Value); err != nil {
				return fmt.Errorf("failed to run cmd %q: %w %s", c.Value, err, out)
			}
		}
	}

	// Make sure that the S3 bucket reset routine is finished and return its error, if any
	return <-resetBucketErrCh
}

func runLoadTest(t *terraform.Terraform, lt LoadTestConfig) (coordinator.Status, error) {
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
		return runBoundedLoadTest(t, coordConfig, duration)
	case LoadTestTypeUnbounded:
		// TODO: cleverly set MaxActiveUsers to (numAgents * UsersConfiguration.MaxActiveUsers)
		return runUnboundedLoadTest(t, coordConfig)
	}

	return status, fmt.Errorf("unimplemented LoadTestType %s", lt.Type)
}
