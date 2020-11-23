// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/assets"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/model"
)

const cmdExecTimeoutMinutes = 30

const (
	latestReleaseURL           = "https://latest.mattermost.com/mattermost-enterprise-linux"
	defaultLoadTestDownloadURL = "https://github.com/mattermost/mattermost-load-test-ng/releases/download/v1.1.0/mattermost-load-test-ng-v1.1.0-linux-amd64.tar.gz"
	filePrefix                 = "file://"
	minSupportedVersion        = 0.12
	maxSupportedVersion        = 0.13
)

// Terraform manages all operations related to interacting with
// an AWS environment using Terraform.
type Terraform struct {
	id     string
	config *deployment.Config
	dir    string
}

// New returns a new Terraform instance.
func New(id string, cfg *deployment.Config) *Terraform {
	return &Terraform{
		id:     id,
		config: cfg,
	}
}

// Create creates a new load test environment.
func (t *Terraform) Create(initData bool) error {
	if err := t.preFlightCheck(); err != nil {
		return err
	}

	if err := validateLicense(t.config.MattermostLicenseFile); err != nil {
		return fmt.Errorf("license validation failed: %w", err)
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}

	var uploadBinary bool
	var binaryPath string
	if strings.HasPrefix(t.config.MattermostDownloadURL, filePrefix) {
		binaryPath = strings.TrimPrefix(t.config.MattermostDownloadURL, filePrefix)
		info, err := os.Stat(binaryPath)
		if err != nil {
			return err
		}

		// We make sure the file is executable by both the owner and group.
		if info.Mode()&0110 != 0110 {
			return fmt.Errorf("file %s has to be an executable", binaryPath)
		}

		if !info.Mode().IsRegular() {
			return fmt.Errorf("binary path %s has to be a regular file", binaryPath)
		}

		t.config.MattermostDownloadURL = latestReleaseURL
		uploadBinary = true
	}

	loadTestDownloadURL := t.config.LoadTestDownloadURL
	if strings.HasPrefix(t.config.LoadTestDownloadURL, filePrefix) {
		loadTestDownloadURL = defaultLoadTestDownloadURL
	}

	err = t.runCommand(nil, "apply",
		"-var", fmt.Sprintf("cluster_name=%s", t.config.ClusterName),
		"-var", fmt.Sprintf("cluster_vpc_id=%s", t.config.ClusterVpcID),
		"-var", fmt.Sprintf("cluster_subnet_id=%s", t.config.ClusterSubnetID),
		"-var", fmt.Sprintf("app_instance_count=%d", t.config.AppInstanceCount),
		"-var", fmt.Sprintf("app_instance_type=%s", t.config.AppInstanceType),
		"-var", fmt.Sprintf("agent_instance_count=%d", t.config.AgentInstanceCount),
		"-var", fmt.Sprintf("agent_instance_type=%s", t.config.AgentInstanceType),
		"-var", fmt.Sprintf("es_instance_count=%d", t.config.ElasticSearchSettings.InstanceCount),
		"-var", fmt.Sprintf("es_instance_type=%s", t.config.ElasticSearchSettings.InstanceType),
		"-var", fmt.Sprintf("es_version=%.1f", t.config.ElasticSearchSettings.Version),
		"-var", fmt.Sprintf("es_vpc=%s", t.config.ElasticSearchSettings.VpcID),
		"-var", fmt.Sprintf("es_create_role=%t", t.config.ElasticSearchSettings.CreateRole),
		"-var", fmt.Sprintf("proxy_instance_type=%s", t.config.ProxyInstanceType),
		"-var", fmt.Sprintf("ssh_public_key=%s", t.config.SSHPublicKey),
		"-var", fmt.Sprintf("db_instance_count=%d", t.config.DBInstanceCount),
		"-var", fmt.Sprintf("db_instance_engine=%s", t.config.DBInstanceEngine),
		"-var", fmt.Sprintf("db_instance_class=%s", t.config.DBInstanceType),
		"-var", fmt.Sprintf("db_username=%s", t.config.DBUserName),
		"-var", fmt.Sprintf("db_password=%s", t.config.DBPassword),
		"-var", fmt.Sprintf("mattermost_download_url=%s", t.config.MattermostDownloadURL),
		"-var", fmt.Sprintf("mattermost_license_file=%s", t.config.MattermostLicenseFile),
		"-var", fmt.Sprintf("load_test_download_url=%s", loadTestDownloadURL),
		"-auto-approve",
		"-input=false",
		"-state="+t.getStatePath(),
		t.dir,
	)
	if err != nil {
		return err
	}

	output, err := t.Output()
	if err != nil {
		return err
	}

	if output.HasMetrics() {
		// Setting up metrics server.
		if err := t.setupMetrics(extAgent, output); err != nil {
			return fmt.Errorf("error setting up metrics server: %w", err)
		}
	}

	if output.HasAppServers() {
		url := output.Instances[0].PublicDNS + ":8065"

		// Updating the config.json for each instance of app server
		t.setupAppServers(output, extAgent, uploadBinary, binaryPath)
		if output.HasProxy() {
			// Updating the nginx config on proxy server
			t.setupProxyServer(output, extAgent)
			url = output.Proxy.PublicDNS
		}

		if err := pingServer("http://" + url); err != nil {
			return fmt.Errorf("error whiling pinging server: %w", err)
		}

		if initData {
			if err := t.createAdminUser(extAgent, output); err != nil {
				return fmt.Errorf("could not create admin user: %w", err)
			}
		}
	}

	if err := t.setupLoadtestAgents(extAgent, output, initData); err != nil {
		return fmt.Errorf("error setting up loadtest agents: %w", err)
	}

	mlog.Info("Deployment complete.")
	t.displayInfo(output)
	runcmd := "go run ./cmd/ltctl"
	if strings.HasPrefix(os.Args[0], "ltctl") {
		runcmd = "ltctl"
	}
	fmt.Printf("To start coordinator, you can use %q command.\n", runcmd+" loadtest start")
	return nil
}

func (t *Terraform) setupAppServers(output *Output, extAgent *ssh.ExtAgent, uploadBinary bool, binaryPath string) {
	for _, val := range output.Instances {
		ip := val.PublicIP
		sshc, err := extAgent.NewClient(ip)
		if err != nil {
			mlog.Error("error in getting ssh connection", mlog.String("ip", ip), mlog.Err(err))
			continue
		}
		func() {
			defer func() {
				err := sshc.Close()
				if err != nil {
					mlog.Error("error closing ssh connection", mlog.Err(err))
				}
			}()

			mlog.Info("Updating config", mlog.String("host", ip))
			if err := t.updateAppConfig(ip, sshc, output); err != nil {
				mlog.Error("error updating config", mlog.Err(err))
				return
			}

			// Upload files
			batch := []uploadInfo{
				{srcData: strings.TrimSpace(serverSysctlConfig), dstPath: "/etc/sysctl.conf"},
				{srcData: strings.TrimSpace(serviceFile), dstPath: "/lib/systemd/system/mattermost.service"},
				{srcData: strings.TrimPrefix(limitsConfig, "\n"), dstPath: "/etc/security/limits.conf"},
			}
			if err := uploadBatch(sshc, batch); err != nil {
				mlog.Error("batch upload failed", mlog.Err(err))
				return
			}

			// Upload binary if needed.
			if uploadBinary {
				mlog.Info("Uploading binary", mlog.String("host", ip))
				cmd := "sudo systemctl daemon-reload && sudo service mattermost stop"
				if out, err := sshc.RunCommand(cmd); err != nil {
					mlog.Error("error running ssh command", mlog.String("cmd", cmd), mlog.String("output", string(out)), mlog.Err(err))
					return
				}

				if out, err := sshc.UploadFile(binaryPath, "/opt/mattermost/bin/mattermost", false); err != nil {
					mlog.Error("error uploading file", mlog.String("file", binaryPath), mlog.String("output", string(out)), mlog.Err(err))
					return
				}
			}

			// Starting mattermost.
			mlog.Info("Applying kernel settings and starting mattermost", mlog.String("host", ip))
			cmd := "sudo sysctl -p && sudo systemctl daemon-reload && sudo service mattermost restart"
			if out, err := sshc.RunCommand(cmd); err != nil {
				mlog.Error("error running ssh command", mlog.String("cmd", cmd), mlog.String("output", string(out)), mlog.Err(err))
				return
			}
		}()
	}
}

func (t *Terraform) setupLoadtestAgents(extAgent *ssh.ExtAgent, output *Output, initData bool) error {
	if err := t.configureAndRunAgents(extAgent, output); err != nil {
		return fmt.Errorf("error while setting up an agents: %w", err)
	}

	if !output.HasAppServers() {
		return nil
	}

	if err := t.initLoadtest(extAgent, output, initData); err != nil {
		return err
	}

	return nil
}

func (t *Terraform) setupProxyServer(output *Output, extAgent *ssh.ExtAgent) {
	ip := output.Proxy.PublicDNS

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
		for _, addr := range output.Instances {
			backends += "server " + addr.PrivateIP + ":8065 max_fails=3;\n"
		}

		batch := []uploadInfo{
			{srcData: strings.TrimSpace(fmt.Sprintf(nginxSiteConfig, backends)), dstPath: "/etc/nginx/sites-available/mattermost"},
			{srcData: strings.TrimSpace(serverSysctlConfig), dstPath: "/etc/sysctl.conf"},
			{srcData: strings.TrimSpace(nginxConfig), dstPath: "/etc/nginx/nginx.conf"},
			{srcData: strings.TrimSpace(limitsConfig), dstPath: "/etc/security/limits.conf"},
		}
		if err := uploadBatch(sshc, batch); err != nil {
			mlog.Error("batch upload failed", mlog.Err(err))
			return
		}

		cmd := "sudo sysctl -p && sudo service nginx reload"
		if out, err := sshc.RunCommand(cmd); err != nil {
			mlog.Error("error running ssh command", mlog.String("output", string(out)), mlog.String("cmd", cmd), mlog.Err(err))
			return
		}

	}()
}

func (t *Terraform) createAdminUser(extAgent *ssh.ExtAgent, output *Output) error {
	cmd := fmt.Sprintf("/opt/mattermost/bin/mattermost user create --email %s --username %s --password %s --system_admin",
		t.config.AdminEmail,
		t.config.AdminUsername,
		t.config.AdminPassword,
	)
	mlog.Info("Creating admin user:", mlog.String("cmd", cmd))
	sshc, err := extAgent.NewClient(output.Instances[0].PublicIP)
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

func (t *Terraform) updateAppConfig(ip string, sshc *ssh.Client, output *Output) error {
	var clusterDSN, driverName string
	var readerDSN []string
	switch t.config.DBInstanceEngine {
	case "aurora-postgresql":
		clusterDSN = "postgres://" + t.config.DBUserName + ":" + t.config.DBPassword + "@" + output.DBCluster.ClusterEndpoint + "/" + t.config.ClusterName + "db?sslmode=disable"
		readerDSN = []string{"postgres://" + t.config.DBUserName + ":" + t.config.DBPassword + "@" + output.DBCluster.ReaderEndpoint + "/" + t.config.ClusterName + "db?sslmode=disable"}
		driverName = "postgres"
	case "aurora-mysql":
		clusterDSN = t.config.DBUserName + ":" + t.config.DBPassword + "@tcp(" + output.DBCluster.ClusterEndpoint + ")/" + t.config.ClusterName + "db?charset=utf8mb4,utf8\u0026readTimeout=30s\u0026writeTimeout=30s"
		readerDSN = []string{t.config.DBUserName + ":" + t.config.DBPassword + "@tcp(" + output.DBCluster.ReaderEndpoint + ")/" + t.config.ClusterName + "db?charset=utf8mb4,utf8\u0026readTimeout=30s\u0026writeTimeout=30s"}
		driverName = "mysql"
	}

	cfg := &model.Config{}
	cfg.SetDefaults()
	cfg.ServiceSettings.ListenAddress = model.NewString(":8065")
	cfg.ServiceSettings.LicenseFileLocation = model.NewString("/home/ubuntu/mattermost.mattermost-license")
	cfg.ServiceSettings.SiteURL = model.NewString("http://" + ip + ":8065")
	cfg.ServiceSettings.ReadTimeout = model.NewInt(60)
	cfg.ServiceSettings.WriteTimeout = model.NewInt(60)
	cfg.ServiceSettings.IdleTimeout = model.NewInt(90)

	cfg.EmailSettings.SMTPServer = model.NewString(output.MetricsServer.PrivateIP)
	cfg.EmailSettings.SMTPPort = model.NewString("2500")

	if output.HasProxy() && output.HasS3Key() && output.HasS3Bucket() {
		cfg.FileSettings.DriverName = model.NewString("amazons3")
		cfg.FileSettings.AmazonS3AccessKeyId = model.NewString(output.S3Key.Id)
		cfg.FileSettings.AmazonS3SecretAccessKey = model.NewString(output.S3Key.Secret)
		cfg.FileSettings.AmazonS3Bucket = model.NewString(output.S3Bucket.Id)
		cfg.FileSettings.AmazonS3Region = model.NewString(output.S3Bucket.Region)
	}

	cfg.LogSettings.EnableConsole = model.NewBool(true)
	cfg.LogSettings.ConsoleLevel = model.NewString("ERROR")
	cfg.LogSettings.EnableFile = model.NewBool(true)
	cfg.LogSettings.FileLevel = model.NewString("WARN")

	cfg.SqlSettings.DriverName = model.NewString(driverName)
	cfg.SqlSettings.DataSource = model.NewString(clusterDSN)
	cfg.SqlSettings.DataSourceReplicas = readerDSN
	cfg.SqlSettings.MaxIdleConns = model.NewInt(100)
	cfg.SqlSettings.MaxOpenConns = model.NewInt(512)

	cfg.TeamSettings.MaxUsersPerTeam = model.NewInt(50000)
	cfg.TeamSettings.EnableOpenServer = model.NewBool(true)

	cfg.ClusterSettings.GossipPort = model.NewInt(8074)
	cfg.ClusterSettings.StreamingPort = model.NewInt(8075)
	cfg.ClusterSettings.Enable = model.NewBool(true)
	cfg.ClusterSettings.ClusterName = model.NewString(t.config.ClusterName)
	cfg.ClusterSettings.ReadOnlyConfig = model.NewBool(false)

	cfg.MetricsSettings.Enable = model.NewBool(true)

	cfg.PluginSettings.Enable = model.NewBool(true)
	cfg.PluginSettings.EnableUploads = model.NewBool(true)

	if output.HasElasticSearch() {
		cfg.ElasticsearchSettings.ConnectionUrl = model.NewString("https://" + output.ElasticSearchServer.Endpoint)
		cfg.ElasticsearchSettings.Username = model.NewString("")
		cfg.ElasticsearchSettings.Password = model.NewString("")
		cfg.ElasticsearchSettings.Sniff = model.NewBool(false)
		cfg.ElasticsearchSettings.EnableIndexing = model.NewBool(true)
		cfg.ElasticsearchSettings.EnableAutocomplete = model.NewBool(true)
		cfg.ElasticsearchSettings.EnableSearching = model.NewBool(true)
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

	if err := t.init(); err != nil {
		return err
	}

	if err := t.validate(); err != nil {
		return err
	}
	return nil
}

func (t *Terraform) init() error {
	dir, err := ioutil.TempDir("", "terraform")
	if err != nil {
		return err
	}
	t.dir = dir
	assets.RestoreAssets(dir, "outputs.tf")
	assets.RestoreAssets(dir, "variables.tf")
	assets.RestoreAssets(dir, "cluster.tf")
	assets.RestoreAssets(dir, "datasource.yaml")
	assets.RestoreAssets(dir, "dashboard.yaml")
	assets.RestoreAssets(dir, "dashboard_data.json")
	assets.RestoreAssets(dir, "es_dashboard_data.json")

	return t.runCommand(nil, "init", t.dir)
}

func (t *Terraform) validate() error {
	return t.runCommand(nil, "validate", t.dir)
}

func pingServer(addr string) error {
	mlog.Info("Checking server status:", mlog.String("host", addr))
	client := model.NewAPIv4Client(addr)
	client.HttpClient.Timeout = 10 * time.Second
	timeout := time.After(30 * time.Second)

	for {
		select {
		case <-timeout:
			return errors.New("timeout after 30 seconds, server is not responding")
		case <-time.After(3 * time.Second):
			_, resp := client.GetPingWithServerStatus()
			if resp.Error != nil {
				mlog.Debug("got error", mlog.Err(resp.Error))
				mlog.Info("Waiting for the server...")
				continue
			}
			mlog.Info("Server status is OK")
			return nil
		}
	}
}
