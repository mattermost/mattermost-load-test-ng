// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/opensearch"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/assets"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/mattermost/mattermost/server/v8/config"
)

const (
	latestReleaseURL = "https://latest.mattermost.com/mattermost-enterprise-linux"
	filePrefix       = "file://"
	releaseSuffix    = "tar.gz"

	cmdExecTimeoutMinutes = 120

	gossipPort = 8074
)

// requiredVersion specifies the supported versions of Terraform,
// which are those that meet the following criteria:
// 1. installedVersion.Major = requiredVersion.Major
// 2. installedVersion >= requiredVersion
var requiredVersion = semver.MustParse("1.3.3")

// A global mutex used to make t.init() safe for concurrent use.
// This is needed to prevent a data race caused by the "terraform init"
// command which can modify common files in the .terraform directory.
// Making this a global variable to avoid exporting more methods and
// having the user of this package deal with this special case.
var initMut sync.Mutex

// Terraform manages all operations related to interacting with
// an AWS environment using Terraform.
type Terraform struct {
	id          string
	config      *deployment.Config
	output      *Output
	initialized bool
}

// New returns a new Terraform instance.
func New(id string, cfg deployment.Config) (*Terraform, error) {
	if err := ensureTerraformStateDir(cfg.TerraformStateDir); err != nil {
		if errors.Is(err, os.ErrPermission) {
			errStr := fmt.Sprintf("not enough permissions to create Terraform state directory %q.\n", cfg.TerraformStateDir)
			errStr += "Here's some alternatives you can try:\n"
			errStr += "\t1. Change the TerraformStateDir setting in config/deployer.json to a directory you have permissions over (recommended).\n"
			errStr += fmt.Sprintf("\t2. Manually create the currently configured directory %q and change its owner to your current user.\n", cfg.TerraformStateDir)
			errStr += "\t3. Run this and all next commands as root (not recommended)."
			return nil, fmt.Errorf(errStr)
		}
		return nil, fmt.Errorf("unable to create Terraform state directory %q: %w", cfg.TerraformStateDir, err)
	}

	return &Terraform{
		id:     id,
		config: &cfg,
	}, nil
}

func ensureTerraformStateDir(dir string) error {
	// Make sure that the state directory exists
	_, err := os.Stat(dir)
	if err == nil {
		return nil
	}

	// Return any error different than the one showing
	// that the directory does not exist
	if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	// If it does not exist, create it
	return os.Mkdir(dir, 0700)
}

// Create creates a new load test environment.
func (t *Terraform) Create(initData bool) error {
	if err := t.preFlightCheck(); err != nil {
		return err
	}

	// Validate the license only if we deploy app nodes;
	// otherwise we don't need a license at all
	if t.config.AppInstanceCount > 0 {
		if err := validateLicense(t.config.MattermostLicenseFile); err != nil {
			return fmt.Errorf("license validation failed: %w", err)
		}
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}

	var uploadPath string
	var uploadBinary bool
	var uploadRelease bool

	if strings.HasPrefix(t.config.MattermostDownloadURL, filePrefix) &&
		strings.HasSuffix(t.config.MattermostDownloadURL, releaseSuffix) {
		uploadPath = strings.TrimPrefix(t.config.MattermostDownloadURL, filePrefix)
		info, err := os.Stat(uploadPath)
		if err != nil {
			return err
		}

		if !info.Mode().IsRegular() {
			return fmt.Errorf("release path %s has to be a regular file", uploadPath)
		}

		uploadRelease = true
	} else if strings.HasPrefix(t.config.MattermostDownloadURL, filePrefix) {
		uploadPath = strings.TrimPrefix(t.config.MattermostDownloadURL, filePrefix)
		info, err := os.Stat(uploadPath)
		if err != nil {
			return err
		}

		// We make sure the file is executable by both the owner and group.
		if info.Mode()&0110 != 0110 {
			return fmt.Errorf("file %s has to be an executable", uploadPath)
		}

		if !info.Mode().IsRegular() {
			return fmt.Errorf("binary path %s has to be a regular file", uploadPath)
		}

		t.config.MattermostDownloadURL = latestReleaseURL
		uploadBinary = true
	}

	var params []string
	params = append(params, "apply")
	params = append(params, t.getParams()...)
	params = append(params, "-auto-approve",
		"-input=false",
		"-state="+t.getStatePath())

	err = t.runCommand(nil, params...)
	if err != nil {
		return err
	}

	if err := t.loadOutput(); err != nil {
		return err
	}

	// If we are restoring from a DB backup, then we need to hook up
	// the security group to it.
	if t.config.TerraformDBSettings.ClusterIdentifier != "" {
		if len(t.output.DBSecurityGroup) == 0 {
			return errors.New("No DB security group created")
		}

		sgID := t.output.DBSecurityGroup[0].Id
		args := []string{
			"rds",
			"modify-db-cluster",
			"--db-cluster-identifier=" + t.config.TerraformDBSettings.ClusterIdentifier,
			"--vpc-security-group-ids=" + sgID,
			"--region=" + t.config.AWSRegion,
		}
		if err := t.runAWSCommand(nil, args, nil); err != nil {
			return err
		}
	}

	// Check if a policy to publish logs from OpenSearch Service to CloudWatch
	// exists, and create it otherwise.
	// Note that we are doing this outside of Terraform. This is because we
	// only need one policy in the account, not one per deployment. This
	// behaviour is also enforced by the rate limitation on this type of
	// policies: there can only be 10 such policies per region per account.
	// Check the docs for more information:
	// https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/cloudwatch_limits_cwl.html

	// BRANCH: disabling this due to being disabled in the terraform code
	// if err = t.checkCloudWatchLogsPolicy(); err != nil {
	// 	if err != ErrNotFound {
	// 		return fmt.Errorf("failed to check CloudWatchLogs policy: %w", err)
	// 	}

	// 	mlog.Info("No CloudWatchLogs policy found, creating a new one")
	// 	if err := t.createCloudWatchLogsPolicy(); err != nil {
	// 		return fmt.Errorf("failed creating CloudWatchLogs policy")
	// 	}
	// }

	if t.output.HasMetrics() {
		// Setting up metrics server.
		if err := t.setupMetrics(extAgent); err != nil {
			return fmt.Errorf("error setting up metrics server: %w", err)
		}
	}

	if t.output.HasKeycloak() {
		// Setting up keycloak.
		if err := t.setupKeycloak(extAgent); err != nil {
			return fmt.Errorf("error setting up keycloak server: %w", err)
		}
	}

	if t.output.HasAppServers() {
		var siteURL string
		switch {
		// SiteURL defined, multiple app nodes: we use SiteURL, since that points to the proxy itself
		case t.config.SiteURL != "" && t.output.HasProxy():
			siteURL = "http://" + t.config.SiteURL
		// SiteURL defined, single app node: we use SiteURL plus the port, since SiteURL points to the app node (which is listening in 8065)
		case t.config.SiteURL != "":
			siteURL = "http://" + t.config.SiteURL + ":8065"
		// SiteURL not defined, multiple app nodes: we use the proxy's public DNS
		case t.output.HasProxy():
			siteURL = "http://" + t.output.Proxies[0].PrivateDNS
		// SiteURL not defined, single app node: we use the app node's public DNS plus port
		default:
			siteURL = "http://" + t.output.Instances[0].PrivateDNS + ":8065"
		}

		// Updating the config.json for each instance of app server
		if err := t.setupAppServers(extAgent, uploadBinary, uploadRelease, uploadPath, siteURL); err != nil {
			return fmt.Errorf("error setting up app servers: %w", err)
		}

		// The URL to ping cannot be the same as the site URL, since that one could contain a
		// hostname that only instances know how to resolve
		pingURL := t.output.Instances[0].PrivateDNS + ":8065"
		if t.output.HasProxy() {
			for _, inst := range t.output.Proxies {
				// Updating the nginx config on proxy server
				t.setupProxyServer(extAgent, inst)
			}
			// We can ping the server through any of the proxies, doesn't matter here.
			pingURL = t.output.Proxies[0].PrivateDNS
		}

		if err := pingServer("http://" + pingURL); err != nil {
			return fmt.Errorf("error whiling pinging server: %w", err)
		}
	}

	// Ingesting the DB dump and restoring the Elasticsearch snapshot can
	// happen concurrently to reduce the deployment's total time
	errorsChan := make(chan error, 2)
	var wg sync.WaitGroup

	// Setup Elasticsearch
	wg.Add(1)
	go func() {
		defer wg.Done()
		if t.output.HasElasticSearch() {
			if val := os.Getenv("DISABLE_ES_SETUP"); val == "true" {
				mlog.Info("DISABLE_ES_SETUP set, skipping Elasticsearch setup")
			} else {
				mlog.Info("Setting up Elasticsearch")
				err := t.setupElasticSearchServer(extAgent, t.output.Instances[0].PrivateIP)

				if err != nil {
					errorsChan <- fmt.Errorf("unable to setup Elasticsearch server: %w", err)
					return
				}
			}
		}

		errorsChan <- nil
	}()

	// Ingest DB dump
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Note: This MUST be done after app servers have been set up.
		// Otherwise, the vacuuming command will fail because no tables would
		// have been created by then.
		if t.output.HasDB() {
			// Load the dump if specified
			if !initData && t.config.DBDumpURI != "" {
				err = t.IngestDump()
				if err != nil {
					errorsChan <- fmt.Errorf("failed to create ingest dump: %w", err)
					return
				}
			}

			if len(t.config.DBExtraSQL) > 0 {
				// Run extra SQL commands if specified
				if err := t.ExecuteCustomSQL(); err != nil {
					errorsChan <- fmt.Errorf("failed to execute custom SQL: %w", err)
					return
				}
			}

			// Clear licenses data
			if err := t.ClearLicensesData(); err != nil {
				errorsChan <- fmt.Errorf("failed to clear old licenses data: %w", err)
				return
			}

			if t.config.TerraformDBSettings.InstanceEngine == "aurora-postgresql" {
				// updatePostgresSettings does some housekeeping stuff like setting
				// default_search_config and vacuuming tables.
				if err := t.updatePostgresSettings(extAgent); err != nil {
					errorsChan <- fmt.Errorf("could not modify default_search_text_config: %w", err)
					return
				}
			}
		}

		errorsChan <- nil
	}()

	// Fail on any errors from the two goroutines above
	wg.Wait()
	close(errorsChan)
	for err := range errorsChan {
		if err != nil {
			return err
		}
	}

	if t.output.HasDB() && initData && t.config.TerraformDBSettings.ClusterIdentifier == "" {
		if err := t.createAdminUser(extAgent); err != nil {
			return fmt.Errorf("could not create admin user: %w", err)
		}
	}

	if err := t.setupLoadtestAgents(extAgent, initData); err != nil {
		return fmt.Errorf("error setting up loadtest agents: %w", err)
	}

	mlog.Info("Deployment complete.")
	displayInfo(t.output)
	runcmd := "go run ./cmd/ltctl"
	if strings.HasPrefix(os.Args[0], "ltctl") {
		runcmd = "ltctl"
	}
	fmt.Printf("To start coordinator, you can use %q command.\n", runcmd+" loadtest start")
	return nil
}

func (t *Terraform) setupAppServers(extAgent *ssh.ExtAgent, uploadBinary bool, uploadRelease bool, uploadPath string, siteURL string) error {
	for _, val := range t.output.Instances {
		err := t.setupMMServer(extAgent, val.PublicIP, siteURL, uploadBinary, uploadRelease, uploadPath, val.Tags.Name)
		if err != nil {
			return err
		}
	}

	for _, val := range t.output.JobServers {
		err := t.setupJobServer(extAgent, val.PublicIP, siteURL, uploadBinary, uploadRelease, uploadPath, val.Tags.Name)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Terraform) setupMMServer(extAgent *ssh.ExtAgent, ip, siteURL string, uploadBinary bool, uploadRelease bool, uploadPath string, instanceName string) error {
	return t.setupAppServer(extAgent, ip, siteURL, mattermostServiceFile, uploadBinary, uploadRelease, uploadPath, !t.output.HasJobServer(), instanceName)
}

func (t *Terraform) setupJobServer(extAgent *ssh.ExtAgent, ip, siteURL string, uploadBinary bool, uploadRelease bool, uploadPath string, instanceName string) error {
	return t.setupAppServer(extAgent, ip, siteURL, jobServerServiceFile, uploadBinary, uploadRelease, uploadPath, true, instanceName)
}

func (t *Terraform) setupAppServer(extAgent *ssh.ExtAgent, ip, siteURL, serviceFile string, uploadBinary bool, uploadRelease bool, uploadPath string, jobServerEnabled bool, instanceName string) error {
	sshc, err := extAgent.NewClient(ip)
	if err != nil {
		return fmt.Errorf("error in getting ssh connection to %q: %w", ip, err)
	}
	defer func() {
		err := sshc.Close()
		if err != nil {
			mlog.Error("error closing ssh connection", mlog.Err(err))
		}
	}()

	otelcolConfig, err := renderAppOtelcolConfig(instanceName, t.output.MetricsServer.PrivateIP)
	if err != nil {
		return fmt.Errorf("unable to render otelcol config template: %w", err)
	}

	// Upload files
	batch := []uploadInfo{
		{srcData: strings.TrimPrefix(serverSysctlConfig, "\n"), dstPath: "/etc/sysctl.conf"},
		{srcData: strings.TrimSpace(fmt.Sprintf(serviceFile, os.Getenv("MM_SERVICEENVIRONMENT"))), dstPath: "/lib/systemd/system/mattermost.service"},
		{srcData: strings.TrimPrefix(limitsConfig, "\n"), dstPath: "/etc/security/limits.conf"},
		{srcData: strings.TrimPrefix(prometheusNodeExporterConfig, "\n"), dstPath: "/etc/default/prometheus-node-exporter"},
		{srcData: strings.TrimSpace(fmt.Sprintf(netpeekServiceFile, gossipPort)), dstPath: "/lib/systemd/system/netpeek.service"},
		{srcData: strings.TrimSpace(otelcolConfig), dstPath: "/etc/otelcol-contrib/config.yaml"},
	}

	// If SiteURL is set, update /etc/hosts to point to the correct IP
	if t.config.SiteURL != "" {
		// Here it's fine to just use the first proxy because this is
		// only to resolve permalinks.
		appHostsFile, err := t.getAppHostsFile(0)
		if err != nil {
			return err
		}

		batch = append(batch,
			uploadInfo{srcData: appHostsFile, dstPath: "/etc/hosts"},
		)
	}

	if err := uploadBatch(sshc, batch); err != nil {
		return fmt.Errorf("batch upload failed: %w", err)
	}

	cmd := "sudo systemctl restart otelcol-contrib"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command %q, output: %q: %w", cmd, string(out), err)
	}

	cmd = "sudo systemctl daemon-reload && sudo service mattermost stop"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command %q, output: %q: %w", cmd, string(out), err)
	}

	// provision MM build
	var commands []string
	if uploadRelease {
		mlog.Info("Uploading release from tar.gz file", mlog.String("host", ip))

		filename := path.Base(uploadPath)
		if out, err := sshc.UploadFile(uploadPath, "/home/ubuntu/"+filename, false); err != nil {
			return fmt.Errorf("error uploading file %q, output: %q: %w", uploadPath, string(out), err)
		}
		commands = []string{
			"tar xzf " + filename,
			"sudo rm -rf /opt/mattermost",
			"sudo mv mattermost /opt/",
		}
	} else {
		mlog.Info("Uploading release from MattermostDownloadURL", mlog.String("host", ip))

		commands = []string{
			"wget -O mattermost-dist.tar.gz " + t.config.MattermostDownloadURL,
			"tar xzf mattermost-dist.tar.gz",
			"sudo rm -rf /opt/mattermost",
			"sudo mv mattermost /opt/",
		}
	}
	mlog.Info("Provisioning MM build", mlog.String("host", ip))
	cmd = strings.Join(commands, " && ")
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command %q, ourput: %q: %w", cmd, string(out), err)
	}

	mlog.Info("Updating config", mlog.String("host", ip))
	if err := t.updateAppConfig(siteURL, sshc, jobServerEnabled); err != nil {
		return fmt.Errorf("error updating config: %w", err)
	}

	// Upload binary if needed.
	if uploadBinary {
		mlog.Info("Uploading binary", mlog.String("host", ip))

		if out, err := sshc.UploadFile(uploadPath, "/opt/mattermost/bin/mattermost", false); err != nil {
			return fmt.Errorf("error uploading file %q, output: %q: %w", uploadPath, string(out), err)
		}
	}

	if t.config.EnableNetPeekMetrics {
		mlog.Info("Starting netpeek service", mlog.String("host", ip))
		cmd = "sudo systemctl daemon-reload && sudo chmod +x /usr/local/bin/netpeek && sudo service netpeek restart"
		if out, err := sshc.RunCommand(cmd); err != nil {
			return fmt.Errorf("error running ssh command %q, output: %q: %w", cmd, string(out), err)
		}
	}

	// Starting mattermost.
	mlog.Info("Applying kernel settings and starting mattermost", mlog.String("host", ip))
	cmd = "sudo sysctl -p && sudo systemctl daemon-reload && sudo service mattermost restart"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command %q, output: %q: %w", cmd, string(out), err)
	}

	return nil
}

func (t *Terraform) setupLoadtestAgents(extAgent *ssh.ExtAgent, initData bool) error {
	if err := t.configureAndRunAgents(extAgent); err != nil {
		return fmt.Errorf("error while setting up an agents: %w", err)
	}

	if !t.output.HasAppServers() {
		return nil
	}

	if err := t.initLoadtest(extAgent, initData); err != nil {
		return err
	}

	return nil
}

func (t *Terraform) setupElasticSearchServer(extAgent *ssh.ExtAgent, ip string) error {
	esSettings := t.config.ElasticSearchSettings

	output, err := t.Output()
	if err != nil {
		return fmt.Errorf("unable to get Terraform output to setup ElasticSearch: %w", err)
	}
	esEndpoint := output.ElasticSearchServer.Endpoint

	sshc, err := extAgent.NewClient(ip)
	if err != nil {
		return fmt.Errorf("unable to create SSH client with IP %q: %w", ip, err)
	}

	awsCreds, err := t.GetAWSCreds()
	if err != nil {
		return fmt.Errorf("failed to get AWS credentials")
	}
	os, err := opensearch.New(esEndpoint, sshc, awsCreds, t.config.AWSRegion)
	if err != nil {
		return fmt.Errorf("unable to create Elasticserach client: %w", err)
	}

	indices, err := os.ListIndices()
	if err != nil {
		return fmt.Errorf("unable to list indices: %w", err)
	}
	mlog.Debug("Indices in ElasticSearch domain", mlog.Array("indices", indices))

	repositories, err := os.ListRepositories()
	if err != nil {
		return fmt.Errorf("unable to list repositories: %w", err)
	}
	mlog.Debug("Repositories registered", mlog.Array("repositories", repositories))

	repo := esSettings.SnapshotRepository

	// Check if the registered repositories already include the one configured
	repoFound := false
	for _, r := range repositories {
		if r.Name == repo {
			repoFound = true
			break
		}
	}

	// Register the repository configured if not found
	if !repoFound {
		arn := output.ElasticSearchRoleARN
		if err := os.RegisterS3Repository(repo, arn); err != nil {
			return fmt.Errorf("unable to register repository: %w", err)
		}
		mlog.Info("Repository registered", mlog.String("repository", repo))
	}

	// List all snapshots in the configured repository
	snapshots, err := os.ListSnapshots(repo)
	if err != nil {
		return fmt.Errorf("unable to list snapshots: %w", err)
	}
	mlog.Debug("Snapshots in registered repository", mlog.Array("snapshots", snapshots))

	// Look for the configured snapshot
	var snapshot opensearch.Snapshot
	snapshotName := esSettings.SnapshotName
	for _, s := range snapshots {
		if s.Name == snapshotName {
			snapshot = s
			break
		}
	}
	if snapshot.Name == "" {
		return fmt.Errorf("snapshot %q not found in repository %q", snapshotName, repo)
	}

	// Filter out indices starting with '.', like .kibana_1
	snapshotIndices := []string{}
	for _, i := range snapshot.Indices {
		if !strings.HasPrefix(i, ".") {
			snapshotIndices = append(snapshotIndices, i)
		}

	}
	mlog.Debug("Indices in the snapshot to be restored", mlog.Array("snapshot indices", snapshotIndices))

	// We need to close all indices already present in the ElasticSearch
	// server that we want to restore from the snapshot. This is needed for
	// fresh servers as well, since at least the users index is sometimes
	// present on creation.
	indicesToClose := []string{}
	for _, index := range indices {
		for _, snapshotIndex := range snapshotIndices {
			if index == snapshotIndex {
				indicesToClose = append(indicesToClose, index)
			}
		}
	}

	if len(indicesToClose) > 0 {
		mlog.Info("Closing indices in ElasticSearch server...", mlog.Array("indices", indicesToClose))
		if err := os.CloseIndices(indicesToClose); err != nil {
			return fmt.Errorf("unable to close indices: %w", err)
		}
	}

	mlog.Info("Restoring snapshot...",
		mlog.String("repository", repo),
		mlog.String("snapshot", snapshotName),
		mlog.Array("indices", snapshotIndices))
	opts := opensearch.RestoreSnapshotOpts{
		WithIndices:      snapshotIndices,
		NumberOfReplicas: esSettings.InstanceCount - 1,
	}
	if err := os.RestoreSnapshot(repo, snapshotName, opts); err != nil {
		return fmt.Errorf("unable to restore snapshot: %w", err)
	}

	// Wait until the snapshot is completely restored, or the user-specified
	// timeout is triggered, whatever happens first
	if err := t.waitForSnapshot(os, snapshotIndices); err != nil {
		return fmt.Errorf("failed to wait for snapshot completion: %w", err)
	}

	// Double-check that the cluster's status is green: even if the index is
	// fully restored, the creation of shard replicas may not have finished,
	// so we give it an additional timeout. The time it takes for all shards
	// to get assigned and initialized is proportional to the number of nodes
	// in the Elasticsearch cluster.
	// The potentially large combined timeout (by default, 45 minutes for
	// restoring the snapshot, 45 minutes for a green cluster) will rarely
	// happen, but we need a large timeout here as well in case the snapshot is
	// already restored and we re-deploy the cluster with additional data
	// nodes: in that case, waitForSnapshot will return immediately, so we'll
	// only wait for the cluster's status to get green.
	clusterTimeout := time.Duration(esSettings.ClusterTimeoutMinutes) * time.Minute
	if err := waitForGreenCluster(clusterTimeout, os); err != nil {
		return fmt.Errorf("failed to wait for snapshot completion: %w", err)
	}

	return nil
}

// waitForSnapshot blocks until the snapshot is fully restored or the provided
// timeout is reached
func (t *Terraform) waitForSnapshot(es *opensearch.Client, snapshotIndices []string) error {
	dur := time.Duration(t.config.ElasticSearchSettings.RestoreTimeoutMinutes) * time.Minute
	esDiskSpaceBytes := t.config.StorageSizes.ElasticSearch * 1024 * 1024 * 1024
	timeout := time.After(dur)
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout after %s, snapshot was not restored", dur.String())
		case <-time.After(30 * time.Second):
			indicesRecovery, err := es.SnapshotIndicesRecovery(snapshotIndices)
			if err != nil {
				return fmt.Errorf("unable to get recovery info from indices: %w", err)
			}

			// Compute progress
			var indicesDone, indicesInProgress, totalPerc, totalIndexSize int
			for _, i := range indicesRecovery {
				if i.Stage == "DONE" {
					indicesDone++
				} else {
					indicesInProgress++
				}

				totalIndexSize += i.Size

				// Remove the trailing "%" and the dot, so 75.8% is converted to 758
				percString := strings.ReplaceAll(i.Percent[:len(i.Percent)-1], ".", "")
				perc, err := strconv.Atoi(percString)
				if err != nil {
					return fmt.Errorf("percentage %q cannot be parsed: %w", i.Percent, err)
				}
				totalPerc += perc
			}

			if totalIndexSize > esDiskSpaceBytes {
				return fmt.Errorf("total index size to be restored: %d bytes, greater than configured disk space %d bytes", totalIndexSize, esDiskSpaceBytes)
			}

			// Finish when all indices are marked as "DONE"
			if indicesDone == len(snapshotIndices) {
				mlog.Info("ElasticSearch snapshot restored")
				return nil
			}

			// Otherwise, show the progress
			avgPerc := float64(totalPerc) / 10 / float64(len(snapshotIndices))
			indicesWaiting := len(snapshotIndices) - indicesDone - indicesInProgress
			mlog.Info("Restoring ElasticSearch snapshot...",
				mlog.String("avg % completed", fmt.Sprintf("%.2f", avgPerc)),
				mlog.Int("indices waiting", indicesWaiting),
				mlog.Int("indices in progress", indicesInProgress),
				mlog.Int("indices recovered", indicesDone),
				mlog.Int("total index size", totalIndexSize),
			)
		}
	}
}

// waitForGreenCluster blocks until the cluster's status is green or the
// provided timeout is reached
func waitForGreenCluster(dur time.Duration, es *opensearch.Client) error {
	timeout := time.After(dur)
	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout after %s, cluster's status did not become green", dur.String())
		case <-time.After(30 * time.Second):
			clusterHealth, err := es.ClusterHealth()
			if err != nil {
				return fmt.Errorf("unable to get cluster status: %w", err)
			}

			// Finish when the cluster's status is green
			if clusterHealth.Status == opensearch.ClusterStatusGreen {
				return nil
			}

			mlog.Info("Waiting for cluster's status to become green",
				mlog.String("current status", clusterHealth.Status),
				mlog.Int("initializing shards", clusterHealth.InitializingShards),
				mlog.Int("unassigned shards", clusterHealth.UnassignedShards),
			)
		}
	}
}

func genNginxConfig() (string, error) {
	data := map[string]any{
		"tcpNoDelay": "off",
	}
	if val := os.Getenv(deployment.EnvVarTCPNoDelay); strings.ToLower(val) == "on" {
		data["tcpNoDelay"] = "on"
	}
	return fillConfigTemplate(nginxConfigTmpl, data)
}

func (t *Terraform) getProxyInstanceInfo() (*types.InstanceTypeInfo, error) {
	cfg, err := t.GetAWSConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading AWS config: %v", err)
	}
	descOutput, err := ec2.NewFromConfig(cfg).DescribeInstanceTypes(context.Background(), &ec2.DescribeInstanceTypesInput{
		InstanceTypes: []types.InstanceType{types.InstanceType(t.config.ProxyInstanceType)},
	})
	if err != nil {
		return nil, fmt.Errorf("error trying to describe instance type: %v", err)
	}

	if len(descOutput.InstanceTypes) == 0 {
		return nil, fmt.Errorf("no instance types returned")
	}
	return &descOutput.InstanceTypes[0], nil
}

func (t *Terraform) setupProxyServer(extAgent *ssh.ExtAgent, instance Instance) {
	ip := instance.PublicDNS

	sshc, err := extAgent.NewClient(ip)
	if err != nil {
		mlog.Error("error in getting ssh connection", mlog.String("ip", ip), mlog.Err(err))
		return
	}

	func() {
		defer func() {
			err := sshc.Close()
			if err != nil {
				mlog.Error("error closing ssh connection", mlog.Err(err))
			}
		}()

		// Upload service file
		mlog.Info("Uploading nginx config", mlog.String("host", ip))

		backends := ""
		for _, addr := range t.output.Instances {
			backends += "server " + addr.PrivateIP + ":8065 max_fails=0;\n"
		}

		cacheObjects := "10m"
		cacheSize := "3g"
		rxQueueSize := 1024 // This is the default on most EC2 instances

		info, err := t.getProxyInstanceInfo()
		if err != nil {
			mlog.Error("Error while getting proxy info", mlog.Err(err))
			return
		}

		if *info.MemoryInfo.SizeInMiB >= 32768 { // 32GiB (Usually of instance classes >=4xlarge)
			cacheObjects = "50m"
			cacheSize = "16g" // Ideally we'd like half of the total server mem. But the mem consumption rarely exceeds 10G
			// from my tests. So there's no point stretching it further.

			// MM-58179
			// We are increasing the receive ring buffer size on the network card. This proved to significantly lower packet loss
			// (and retransmissions) on particularly bursty connections (e.g. websockets).
			rxQueueSize = 8192
		}

		nginxConfig, err := genNginxConfig()
		if err != nil {
			mlog.Error("Failed to generate nginx config", mlog.Err(err))
			return
		}

		nginxSiteConfig, err := fillConfigTemplate(nginxSiteConfigTmpl, map[string]any{
			"backends":     backends,
			"cacheObjects": cacheObjects,
			"cacheSize":    cacheSize,
		})
		if err != nil {
			mlog.Error("Failed to generate nginx site config", mlog.Err(err))
			return
		}

		otelcolConfig, err := renderProxyOtelcolConfig(instance.Tags.Name, t.output.MetricsServer.PrivateIP)
		if err != nil {
			mlog.Error("unable to render otelcol config template", mlog.String("proxy", instance.Tags.Name), mlog.Err(err))
			return
		}

		batch := []uploadInfo{
			{srcData: strings.TrimLeft(nginxProxyCommonConfig, "\n"), dstPath: "/etc/nginx/snippets/proxy.conf"},
			{srcData: strings.TrimLeft(nginxCacheCommonConfig, "\n"), dstPath: "/etc/nginx/snippets/cache.conf"},
			{srcData: strings.TrimLeft(nginxSiteConfig, "\n"), dstPath: "/etc/nginx/sites-available/mattermost"},
			{srcData: strings.TrimLeft(serverSysctlConfig, "\n"), dstPath: "/etc/sysctl.conf"},
			{srcData: strings.TrimLeft(nginxConfig, "\n"), dstPath: "/etc/nginx/nginx.conf"},
			{srcData: strings.TrimLeft(limitsConfig, "\n"), dstPath: "/etc/security/limits.conf"},
			{srcData: strings.TrimPrefix(prometheusNodeExporterConfig, "\n"), dstPath: "/etc/default/prometheus-node-exporter"},
			{srcData: strings.TrimSpace(otelcolConfig), dstPath: "/etc/otelcol-contrib/config.yaml"},
		}
		if err := uploadBatch(sshc, batch); err != nil {
			mlog.Error("batch upload failed", mlog.Err(err))
			return
		}

		cmd := "sudo systemctl restart otelcol-contrib"
		if out, err := sshc.RunCommand(cmd); err != nil {
			mlog.Error("error running ssh command", mlog.String("output", string(out)), mlog.String("cmd", cmd), mlog.Err(err))
			return
		}

		incRXSizeCmd := fmt.Sprintf("sudo ethtool -G $(ip route show to default | awk '{print $5}') rx %d", rxQueueSize)
		cmd = fmt.Sprintf("%s && sudo sysctl -p && sudo service nginx restart", incRXSizeCmd)
		if out, err := sshc.RunCommand(cmd); err != nil {
			mlog.Error("error running ssh command", mlog.String("output", string(out)), mlog.String("cmd", cmd), mlog.Err(err))
			return
		}
	}()
}

func (t *Terraform) createAdminUser(extAgent *ssh.ExtAgent) error {
	cmd := fmt.Sprintf("/opt/mattermost/bin/mmctl user create --email %s --username %s --password %s --system-admin --local",
		t.config.AdminEmail,
		t.config.AdminUsername,
		t.config.AdminPassword,
	)
	mlog.Info("Creating admin user:", mlog.String("cmd", cmd))
	sshc, err := extAgent.NewClient(t.output.Instances[0].PrivateIP)
	if err != nil {
		return err
	}
	if out, err := sshc.RunCommand(cmd); err != nil {
		if strings.Contains(string(out), "account with that username already exists") {
			return nil
		}
		return fmt.Errorf("error running ssh command: %s, output: %s, error: %w", cmd, out, err)
	}

	return nil
}

func (t *Terraform) updatePostgresSettings(extAgent *ssh.ExtAgent) error {
	dns := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		t.config.TerraformDBSettings.UserName,
		t.config.TerraformDBSettings.Password,
		t.output.DBWriter(),
		t.config.DBName(),
	)

	if len(t.output.Instances) == 0 {
		return errors.New("no instances found in Terraform output")
	}

	sshc, err := extAgent.NewClient(t.output.Instances[0].PrivateIP)
	if err != nil {
		return err
	}

	const searchConfig = "pg_catalog.english"
	sqlCmd := fmt.Sprintf("ALTER DATABASE %s SET default_text_search_config TO %q",
		t.config.DBName(),
		searchConfig,
	)
	cmd := fmt.Sprintf("psql '%s' -c '%s'", dns, sqlCmd)

	mlog.Info(fmt.Sprintf("Setting default_text_search_config to %q:", searchConfig), mlog.String("cmd", cmd))
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: %s, output: %s, error: %w", cmd, out, err)
	}

	sqlCmd = "vacuum analyze channels, sidebarchannels, sidebarcategories, posts, threads, threadmemberships, channelmembers;"
	cmd = fmt.Sprintf("psql '%s' -c '%s'", dns, sqlCmd)

	mlog.Info("Vacuuming the tables", mlog.String("cmd", cmd))
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: %s, output: %s, error: %w", cmd, out, err)
	}

	return nil
}

func (t *Terraform) updateAppConfig(siteURL string, sshc *ssh.Client, jobServerEnabled bool) error {
	var clusterDSN, driverName string
	var readerDSN []string

	clusterDSN = t.config.ExternalDBSettings.DataSource
	readerDSN = t.config.ExternalDBSettings.DataSourceReplicas
	driverName = t.config.ExternalDBSettings.DriverName

	if t.output.HasDB() {
		var err error
		clusterDSN, err = t.getClusterDSN()
		if err != nil {
			return fmt.Errorf("could not update config: %w", err)
		}

		switch t.config.TerraformDBSettings.InstanceEngine {
		case "aurora-postgresql":
			for _, rd := range t.output.DBReaders() {
				readerDSN = append(readerDSN, "postgres://"+t.config.TerraformDBSettings.UserName+":"+t.config.TerraformDBSettings.Password+"@"+rd+"/"+t.config.DBName()+"?sslmode=disable")
			}
			driverName = "postgres"
		case "aurora-mysql":
			for _, rd := range t.output.DBReaders() {
				readerDSN = append(readerDSN, t.config.TerraformDBSettings.UserName+":"+t.config.TerraformDBSettings.Password+"@tcp("+rd+")/"+t.config.DBName()+"?charset=utf8mb4,utf8\u0026readTimeout=30s\u0026writeTimeout=30s")
			}
			driverName = "mysql"
		}
	}

	cfg := &model.Config{}
	cfg.SetDefaults()
	cfg.ServiceSettings.ListenAddress = model.NewPointer(":8065")
	cfg.ServiceSettings.LicenseFileLocation = model.NewPointer("/home/ubuntu/mattermost.mattermost-license")
	cfg.ServiceSettings.SiteURL = model.NewPointer(siteURL)
	cfg.ServiceSettings.ReadTimeout = model.NewPointer(60)
	cfg.ServiceSettings.WriteTimeout = model.NewPointer(60)
	cfg.ServiceSettings.IdleTimeout = model.NewPointer(90)
	cfg.ServiceSettings.EnableLocalMode = model.NewPointer(true)
	cfg.ServiceSettings.ThreadAutoFollow = model.NewPointer(true)
	cfg.ServiceSettings.CollapsedThreads = model.NewPointer(model.CollapsedThreadsDefaultOn)
	cfg.ServiceSettings.EnableLinkPreviews = model.NewPointer(true)
	cfg.ServiceSettings.EnablePermalinkPreviews = model.NewPointer(true)
	cfg.ServiceSettings.PostPriority = model.NewPointer(true)
	cfg.ServiceSettings.AllowSyncedDrafts = model.NewPointer(true)
	// Setting to * is more of a quick fix. A proper fix would be to get the DNS name of the first
	// node or the proxy and set that.
	cfg.ServiceSettings.AllowCorsFrom = model.NewPointer("*")
	cfg.ServiceSettings.EnableOpenTracing = model.NewPointer(false)    // Large overhead, better to disable
	cfg.ServiceSettings.EnableTutorial = model.NewPointer(false)       // Makes manual testing easier
	cfg.ServiceSettings.EnableOnboardingFlow = model.NewPointer(false) // Makes manual testing easier

	cfg.EmailSettings.SMTPServer = model.NewPointer(t.output.MetricsServer.PrivateIP)
	cfg.EmailSettings.SMTPPort = model.NewPointer("2500")

	if t.output.HasProxy() && t.output.HasS3Key() && t.output.HasS3Bucket() {
		cfg.FileSettings.DriverName = model.NewPointer("amazons3")
		cfg.FileSettings.AmazonS3AccessKeyId = model.NewPointer(t.output.S3Key.Id)
		cfg.FileSettings.AmazonS3SecretAccessKey = model.NewPointer(t.output.S3Key.Secret)
		cfg.FileSettings.AmazonS3Bucket = model.NewPointer(t.output.S3Bucket.Id)
		cfg.FileSettings.AmazonS3Region = model.NewPointer(t.output.S3Bucket.Region)
	} else if t.config.ExternalBucketSettings.AmazonS3Bucket != "" {
		cfg.FileSettings.DriverName = model.NewPointer("amazons3")
		cfg.FileSettings.AmazonS3AccessKeyId = model.NewPointer(t.config.ExternalBucketSettings.AmazonS3AccessKeyId)
		cfg.FileSettings.AmazonS3SecretAccessKey = model.NewPointer(t.config.ExternalBucketSettings.AmazonS3SecretAccessKey)
		cfg.FileSettings.AmazonS3Bucket = model.NewPointer(t.config.ExternalBucketSettings.AmazonS3Bucket)
		cfg.FileSettings.AmazonS3PathPrefix = model.NewPointer(t.config.ExternalBucketSettings.AmazonS3PathPrefix)
		cfg.FileSettings.AmazonS3Region = model.NewPointer(t.config.ExternalBucketSettings.AmazonS3Region)
		cfg.FileSettings.AmazonS3Endpoint = model.NewPointer(t.config.ExternalBucketSettings.AmazonS3Endpoint)
		cfg.FileSettings.AmazonS3SSL = model.NewPointer(t.config.ExternalBucketSettings.AmazonS3SSL)
		cfg.FileSettings.AmazonS3SignV2 = model.NewPointer(t.config.ExternalBucketSettings.AmazonS3SignV2)
		cfg.FileSettings.AmazonS3SSE = model.NewPointer(t.config.ExternalBucketSettings.AmazonS3SSE)
	}

	cfg.LogSettings.EnableConsole = model.NewPointer(true)
	cfg.LogSettings.ConsoleLevel = model.NewPointer("ERROR")
	cfg.LogSettings.EnableFile = model.NewPointer(true)
	cfg.LogSettings.FileLevel = model.NewPointer("WARN")
	cfg.LogSettings.EnableSentry = model.NewPointer(false)

	cfg.NotificationLogSettings.EnableConsole = model.NewPointer(true)
	cfg.NotificationLogSettings.ConsoleLevel = model.NewPointer("ERROR")
	cfg.NotificationLogSettings.EnableFile = model.NewPointer(true)
	cfg.NotificationLogSettings.FileLevel = model.NewPointer("WARN")

	cfg.SqlSettings.DriverName = model.NewPointer(driverName)
	cfg.SqlSettings.DataSource = model.NewPointer(clusterDSN)
	cfg.SqlSettings.DataSourceReplicas = readerDSN
	cfg.SqlSettings.MaxIdleConns = model.NewPointer(100)
	cfg.SqlSettings.MaxOpenConns = model.NewPointer(100)
	cfg.SqlSettings.Trace = model.NewPointer(false) // Can be enabled for specific tests, but defaulting to false to declutter logs

	cfg.TeamSettings.MaxUsersPerTeam = model.NewPointer(200000)           // We don't want to be capped by this limit
	cfg.TeamSettings.MaxChannelsPerTeam = model.NewPointer(int64(200000)) // We don't want to be capped by this limit
	cfg.TeamSettings.EnableOpenServer = model.NewPointer(true)
	cfg.TeamSettings.MaxNotificationsPerChannel = model.NewPointer(int64(1000))

	cfg.ClusterSettings.GossipPort = model.NewPointer(gossipPort)
	cfg.ClusterSettings.Enable = model.NewPointer(true)
	cfg.ClusterSettings.ClusterName = model.NewPointer(t.config.ClusterName)
	cfg.ClusterSettings.ReadOnlyConfig = model.NewPointer(false)
	cfg.ClusterSettings.EnableGossipCompression = model.NewPointer(false)
	cfg.ClusterSettings.EnableExperimentalGossipEncryption = model.NewPointer(true)

	cfg.MetricsSettings.Enable = model.NewPointer(true)
	cfg.MetricsSettings.BlockProfileRate = model.NewPointer(t.config.PyroscopeSettings.BlockProfileRate)

	cfg.PluginSettings.Enable = model.NewPointer(true)
	cfg.PluginSettings.EnableUploads = model.NewPointer(true)

	cfg.JobSettings.RunJobs = model.NewPointer(jobServerEnabled)

	if t.output.HasElasticSearch() {
		cfg.ElasticsearchSettings.ConnectionURL = model.NewPointer("https://" + t.output.ElasticSearchServer.Endpoint)
		cfg.ElasticsearchSettings.Username = model.NewPointer("")
		cfg.ElasticsearchSettings.Password = model.NewPointer("")
		cfg.ElasticsearchSettings.Sniff = model.NewPointer(false)
		cfg.ElasticsearchSettings.EnableIndexing = model.NewPointer(true)
		cfg.ElasticsearchSettings.EnableAutocomplete = model.NewPointer(true)
		cfg.ElasticsearchSettings.EnableSearching = model.NewPointer(true)
		cfg.ElasticsearchSettings.Backend = model.NewPointer(model.ElasticsearchSettingsOSBackend)

		// Make all indices have a shard replica in every data node
		numReplicas := t.config.ElasticSearchSettings.InstanceCount - 1
		cfg.ElasticsearchSettings.ChannelIndexReplicas = model.NewPointer(numReplicas)
		cfg.ElasticsearchSettings.PostIndexReplicas = model.NewPointer(numReplicas)
		cfg.ElasticsearchSettings.UserIndexReplicas = model.NewPointer(numReplicas)
	}

	if t.output.HasRedis() {
		cfg.CacheSettings.CacheType = model.NewString(model.CacheTypeRedis)
		redisEndpoint := net.JoinHostPort(t.output.RedisServer.Address, strconv.Itoa(t.output.RedisServer.Port))
		cfg.CacheSettings.RedisAddress = model.NewString(redisEndpoint)
		cfg.CacheSettings.RedisDB = model.NewInt(0)
	}

	if t.config.MattermostConfigPatchFile != "" {
		data, err := os.ReadFile(t.config.MattermostConfigPatchFile)
		if err != nil {
			return fmt.Errorf("error reading MattermostConfigPatchFile: %w", err)
		}

		var patch model.Config
		if err := json.Unmarshal(data, &patch); err != nil {
			return fmt.Errorf("error parsing patch config: %w", err)
		}

		cfg, err = config.Merge(cfg, &patch, nil)
		if err != nil {
			return fmt.Errorf("error patching config: %w", err)
		}
	}

	if t.output.HasKeycloak() {
		if err := t.setupKeycloakAppConfig(sshc, cfg); err != nil {
			return fmt.Errorf("error setting up Keycloak config: %w", err)
		}
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("error in marshalling config: %w", err)
	}

	if out, err := sshc.Upload(bytes.NewReader(b), "/opt/mattermost/config/config.json", false); err != nil {
		return fmt.Errorf("error uploading config.json: output: %s,  error: %w", out, err)
	}

	return nil
}

func (t *Terraform) preFlightCheck() error {
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		return errors.New("ssh agent not running. Please run eval \"$(ssh-agent -s)\" and then ssh-add")
	}

	if err := checkTerraformVersion(); err != nil {
		return fmt.Errorf("failed when checking terraform version: %w", err)
	}

	if err := t.checkAWSCLI(); err != nil {
		return fmt.Errorf("failed when checking AWS CLI: %w", err)
	}

	if !t.initialized {
		if err := t.init(); err != nil {
			return err
		}
		if err := t.validate(); err != nil {
			return err
		}
	}

	t.initialized = true

	return nil
}

func (t *Terraform) init() error {
	assets.RestoreAssets(t.config.TerraformStateDir, "outputs.tf")
	assets.RestoreAssets(t.config.TerraformStateDir, "variables.tf")
	assets.RestoreAssets(t.config.TerraformStateDir, "cluster.tf")
	assets.RestoreAssets(t.config.TerraformStateDir, "elasticsearch.tf")
	assets.RestoreAssets(t.config.TerraformStateDir, "datasource.yaml")
	assets.RestoreAssets(t.config.TerraformStateDir, "dashboard.yaml")
	assets.RestoreAssets(t.config.TerraformStateDir, "default_dashboard_tmpl.json")
	assets.RestoreAssets(t.config.TerraformStateDir, "coordinator_dashboard_tmpl.json")
	assets.RestoreAssets(t.config.TerraformStateDir, "es_dashboard_data.json")
	assets.RestoreAssets(t.config.TerraformStateDir, "redis_dashboard_data.json")
	assets.RestoreAssets(t.config.TerraformStateDir, "keycloak.service")
	assets.RestoreAssets(t.config.TerraformStateDir, "saml-idp.crt")
	assets.RestoreAssets(t.config.TerraformStateDir, "provisioners")

	// We lock to make this call safe for concurrent use
	// since "terraform init" command can write to common files under
	// the .terraform directory.
	initMut.Lock()
	defer initMut.Unlock()
	return t.runCommand(nil, "init")
}

func (t *Terraform) validate() error {
	return t.runCommand(nil, "validate")
}

func pingServer(addr string) error {
	mlog.Info("Checking server status:", mlog.String("host", addr))
	client := model.NewAPIv4Client(addr)
	client.HTTPClient.Timeout = 10 * time.Second
	dur := 240 * time.Second
	timeout := time.After(dur)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout after %s, server is not responding", dur)
		case <-time.After(3 * time.Second):
			status, _, err := client.GetPingWithServerStatus(context.Background())
			if err != nil {
				mlog.Debug("got error", mlog.Err(err), mlog.String("status", status))
				mlog.Info("Waiting for the server...")
				continue
			}
			mlog.Info("Server status is OK")
			return nil
		}
	}
}
