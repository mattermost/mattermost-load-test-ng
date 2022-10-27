// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/v6/config"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/shared/mlog"
)

const cmdExecTimeoutMinutes = 30

const (
	latestReleaseURL = "https://latest.mattermost.com/mattermost-enterprise-linux"
	filePrefix       = "file://"
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
	stateDir    string
	initialized bool
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

	if t.output.HasMetrics() {
		// Setting up metrics server.
		if err := t.setupMetrics(extAgent); err != nil {
			return fmt.Errorf("error setting up metrics server: %w", err)
		}
	}

	if t.output.HasAppServers() {
		url := t.output.Instances[0].PublicDNS + ":8065"

		// Updating the config.json for each instance of app server
		if err := t.setupAppServers(extAgent, uploadBinary, binaryPath); err != nil {
			return fmt.Errorf("error setting up app servers: %w", err)
		}
		if t.output.HasProxy() {
			// Updating the nginx config on proxy server
			t.setupProxyServer(extAgent)
			url = t.output.Proxy.PublicDNS
		}

		if err := pingServer("http://" + url); err != nil {
			return fmt.Errorf("error whiling pinging server: %w", err)
		}

		// Aurora sets the default_text_search_config parameter to 'simple';
		// a more sane default for a generic deployment is 'english'
		if t.config.TerraformDBSettings.InstanceEngine == "aurora-postgresql" {
			if err := t.setDefaultTextSearchConfig(extAgent, "pg_catalog.english"); err != nil {
				return fmt.Errorf("could not modify default_search_text_config: %w", err)
			}
		}

		if initData {
			if err := t.createAdminUser(extAgent); err != nil {
				return fmt.Errorf("could not create admin user: %w", err)
			}
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

func (t *Terraform) setupAppServers(extAgent *ssh.ExtAgent, uploadBinary bool, binaryPath string) error {
	for _, val := range t.output.Instances {
		err := t.setupMMServer(extAgent, val.PublicIP, uploadBinary, binaryPath)
		if err != nil {
			return err
		}
	}

	for _, val := range t.output.JobServers {
		err := t.setupJobServer(extAgent, val.PublicIP, uploadBinary, binaryPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Terraform) setupMMServer(extAgent *ssh.ExtAgent, ip string, uploadBinary bool, binaryPath string) error {
	return t.setupAppServer(extAgent, ip, mattermostServiceFile, uploadBinary, binaryPath, !t.output.HasJobServer())
}

func (t *Terraform) setupJobServer(extAgent *ssh.ExtAgent, ip string, uploadBinary bool, binaryPath string) error {
	return t.setupAppServer(extAgent, ip, jobServerServiceFile, uploadBinary, binaryPath, true)
}

func (t *Terraform) setupAppServer(extAgent *ssh.ExtAgent, ip, serviceFile string, uploadBinary bool, binaryPath string, jobServerEnabled bool) error {
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

	// Upload files
	batch := []uploadInfo{
		{srcData: strings.TrimPrefix(serverSysctlConfig, "\n"), dstPath: "/etc/sysctl.conf"},
		{srcData: strings.TrimSpace(serviceFile), dstPath: "/lib/systemd/system/mattermost.service"},
		{srcData: strings.TrimPrefix(limitsConfig, "\n"), dstPath: "/etc/security/limits.conf"},
	}
	if err := uploadBatch(sshc, batch); err != nil {
		return fmt.Errorf("batch upload failed: %w", err)
	}

	cmd := "sudo systemctl daemon-reload && sudo service mattermost stop"
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command %q, ourput: %q: %w", cmd, string(out), err)
	}

	// provision MM build
	commands := []string{
		"wget -O mattermost-dist.tar.gz " + t.config.MattermostDownloadURL,
		"tar xzf mattermost-dist.tar.gz",
		"sudo rm -rf /opt/mattermost",
		"sudo mv mattermost /opt/",
	}
	mlog.Info("Provisioning MM build", mlog.String("host", ip))
	cmd = strings.Join(commands, " && ")
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command %q, ourput: %q: %w", cmd, string(out), err)
	}

	mlog.Info("Updating config", mlog.String("host", ip))
	if err := t.updateAppConfig(ip, sshc, jobServerEnabled); err != nil {
		return fmt.Errorf("error updating config: %w", err)
	}

	// Upload binary if needed.
	if uploadBinary {
		mlog.Info("Uploading binary", mlog.String("host", ip))

		if out, err := sshc.UploadFile(binaryPath, "/opt/mattermost/bin/mattermost", false); err != nil {
			return fmt.Errorf("error uploading file %q, output: %q: %w", binaryPath, string(out), err)
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

func genNginxConfig() (string, error) {
	data := map[string]string{
		"tcpNoDelay": "off",
	}
	if val := os.Getenv(deployment.EnvVarTCPNoDelay); strings.ToLower(val) == "on" {
		data["tcpNoDelay"] = "on"
	}
	return fillConfigTemplate(nginxConfigTmpl, data)
}

func (t *Terraform) setupProxyServer(extAgent *ssh.ExtAgent) {
	ip := t.output.Proxy.PublicDNS

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
			backends += "server " + addr.PrivateIP + ":8065 max_fails=3;\n"
		}

		nginxConfig, err := genNginxConfig()
		if err != nil {
			mlog.Error("Failed to generate nginx config", mlog.Err(err))
			return
		}

		batch := []uploadInfo{
			{srcData: strings.TrimLeft(nginxProxyCommonConfig, "\n"), dstPath: "/etc/nginx/snippets/proxy.conf"},
			{srcData: strings.TrimLeft(nginxCacheCommonConfig, "\n"), dstPath: "/etc/nginx/snippets/cache.conf"},
			{srcData: strings.TrimLeft(fmt.Sprintf(nginxSiteConfig, backends), "\n"), dstPath: "/etc/nginx/sites-available/mattermost"},
			{srcData: strings.TrimLeft(serverSysctlConfig, "\n"), dstPath: "/etc/sysctl.conf"},
			{srcData: strings.TrimLeft(nginxConfig, "\n"), dstPath: "/etc/nginx/nginx.conf"},
			{srcData: strings.TrimLeft(limitsConfig, "\n"), dstPath: "/etc/security/limits.conf"},
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

func (t *Terraform) createAdminUser(extAgent *ssh.ExtAgent) error {
	cmd := fmt.Sprintf("/opt/mattermost/bin/mmctl user create --email %s --username %s --password %s --system-admin --local",
		t.config.AdminEmail,
		t.config.AdminUsername,
		t.config.AdminPassword,
	)
	mlog.Info("Creating admin user:", mlog.String("cmd", cmd))
	sshc, err := extAgent.NewClient(t.output.Instances[0].PublicIP)
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

func (t *Terraform) setDefaultTextSearchConfig(extAgent *ssh.ExtAgent, config string) error {
	if t.config.TerraformDBSettings.InstanceEngine != "aurora-postgresql" {
		return fmt.Errorf("default_text_search_config can only be set in postgres")
	}

	dns := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
		t.config.TerraformDBSettings.UserName,
		t.config.TerraformDBSettings.Password,
		t.output.DBWriter(),
		t.config.DBName(),
	)

	sqlCmd := fmt.Sprintf("ALTER DATABASE %s SET default_text_search_config TO \"%s\"",
		t.config.DBName(),
		config,
	)

	cmd := fmt.Sprintf("psql '%s' -c '%s'", dns, sqlCmd)

	mlog.Info(fmt.Sprintf("Setting default_text_search_config to %q:", config), mlog.String("cmd", cmd))
	sshc, err := extAgent.NewClient(t.output.Instances[0].PublicIP)
	if err != nil {
		return err
	}
	if out, err := sshc.RunCommand(cmd); err != nil {
		return fmt.Errorf("error running ssh command: %s, output: %s, error: %w", cmd, out, err)
	}

	return nil
}

func (t *Terraform) updateAppConfig(ip string, sshc *ssh.Client, jobServerEnabled bool) error {
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
	cfg.ServiceSettings.ListenAddress = model.NewString(":8065")
	cfg.ServiceSettings.LicenseFileLocation = model.NewString("/home/ubuntu/mattermost.mattermost-license")
	cfg.ServiceSettings.SiteURL = model.NewString("http://" + ip + ":8065")
	cfg.ServiceSettings.ReadTimeout = model.NewInt(60)
	cfg.ServiceSettings.WriteTimeout = model.NewInt(60)
	cfg.ServiceSettings.IdleTimeout = model.NewInt(90)
	cfg.ServiceSettings.EnableLocalMode = model.NewBool(true)
	cfg.ServiceSettings.ThreadAutoFollow = model.NewBool(true)
	cfg.ServiceSettings.CollapsedThreads = model.NewString(model.CollapsedThreadsDefaultOn)
	cfg.ServiceSettings.EnableLinkPreviews = model.NewBool(true)
	cfg.ServiceSettings.EnablePermalinkPreviews = model.NewBool(true)
	cfg.EmailSettings.SMTPServer = model.NewString(t.output.MetricsServer.PrivateIP)
	cfg.EmailSettings.SMTPPort = model.NewString("2500")

	if t.output.HasProxy() && t.output.HasS3Key() && t.output.HasS3Bucket() {
		cfg.FileSettings.DriverName = model.NewString("amazons3")
		cfg.FileSettings.AmazonS3AccessKeyId = model.NewString(t.output.S3Key.Id)
		cfg.FileSettings.AmazonS3SecretAccessKey = model.NewString(t.output.S3Key.Secret)
		cfg.FileSettings.AmazonS3Bucket = model.NewString(t.output.S3Bucket.Id)
		cfg.FileSettings.AmazonS3Region = model.NewString(t.output.S3Bucket.Region)
	}

	cfg.LogSettings.EnableConsole = model.NewBool(true)
	cfg.LogSettings.ConsoleLevel = model.NewString("ERROR")
	cfg.LogSettings.EnableFile = model.NewBool(true)
	cfg.LogSettings.FileLevel = model.NewString("WARN")

	cfg.NotificationLogSettings.EnableConsole = model.NewBool(true)
	cfg.NotificationLogSettings.ConsoleLevel = model.NewString("ERROR")
	cfg.NotificationLogSettings.EnableFile = model.NewBool(true)
	cfg.NotificationLogSettings.FileLevel = model.NewString("WARN")

	cfg.SqlSettings.DriverName = model.NewString(driverName)
	cfg.SqlSettings.DataSource = model.NewString(clusterDSN)
	cfg.SqlSettings.DataSourceReplicas = readerDSN
	cfg.SqlSettings.MaxIdleConns = model.NewInt(100)
	cfg.SqlSettings.MaxOpenConns = model.NewInt(100)

	cfg.TeamSettings.MaxUsersPerTeam = model.NewInt(50000)
	cfg.TeamSettings.EnableOpenServer = model.NewBool(true)

	cfg.ClusterSettings.GossipPort = model.NewInt(8074)
	cfg.ClusterSettings.StreamingPort = model.NewInt(8075)
	cfg.ClusterSettings.Enable = model.NewBool(true)
	cfg.ClusterSettings.ClusterName = model.NewString(t.config.ClusterName)
	cfg.ClusterSettings.ReadOnlyConfig = model.NewBool(false)
	cfg.ClusterSettings.EnableGossipCompression = model.NewBool(false)
	cfg.ClusterSettings.EnableExperimentalGossipEncryption = model.NewBool(true)

	cfg.MetricsSettings.Enable = model.NewBool(true)

	cfg.PluginSettings.Enable = model.NewBool(true)
	cfg.PluginSettings.EnableUploads = model.NewBool(true)

	cfg.JobSettings.RunJobs = model.NewBool(jobServerEnabled)

	if t.output.HasElasticSearch() {
		cfg.ElasticsearchSettings.ConnectionURL = model.NewString("https://" + t.output.ElasticSearchServer.Endpoint)
		cfg.ElasticsearchSettings.Username = model.NewString("")
		cfg.ElasticsearchSettings.Password = model.NewString("")
		cfg.ElasticsearchSettings.Sniff = model.NewBool(false)
		cfg.ElasticsearchSettings.EnableIndexing = model.NewBool(true)
		cfg.ElasticsearchSettings.EnableAutocomplete = model.NewBool(true)
		cfg.ElasticsearchSettings.EnableSearching = model.NewBool(true)
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
	stateDir, err := os.Getwd()
	if err != nil {
		return err
	}
	t.stateDir = stateDir

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
	timeout := time.After(30 * time.Second)

	for {
		select {
		case <-timeout:
			return errors.New("timeout after 30 seconds, server is not responding")
		case <-time.After(3 * time.Second):
			status, _, err := client.GetPingWithServerStatus()
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
