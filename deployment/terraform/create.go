// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"encoding/json"
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

const cmdExecTimeoutMinutes = 10

const (
	latestReleaseURL           = "https://latest.mattermost.com/mattermost-enterprise-linux"
	defaultLoadTestDownloadURL = "https://github.com/mattermost/mattermost-load-test-ng/releases/download/v0.1.0-alpha/mattermost-load-test-ng-v0.1.0-alpha-linux-amd64.tar.gz"
	filePrefix                 = "file://"
)

// Terraform manages all operations related to interacting with
// an AWS environment using Terraform.
type Terraform struct {
	config *deployment.Config
	dir    string
}

// terraformOutput contains the output variables which are
// created after a deployment.
type terraformOutput struct {
	Proxy struct {
		Value struct {
			PrivateIP  string `json:"private_ip"`
			PublicIP   string `json:"public_ip"`
			PublicDNS  string `json:"public_dns"`
			PrivateDNS string `json:"private_dns"`
		} `json:"value"`
	} `json:"proxy"`
	Instances struct {
		Value []struct {
			PrivateIP  string `json:"private_ip"`
			PublicIP   string `json:"public_ip"`
			PublicDNS  string `json:"public_dns"`
			PrivateDNS string `json:"private_dns"`
		} `json:"value"`
	} `json:"instances"`
	DBCluster struct {
		Value struct {
			ClusterEndpoint string `json:"endpoint"`
			ReaderEndpoint  string `json:"reader_endpoint"`
		} `json:"value"`
	} `json:"dbCluster"`
	Agents struct {
		Value []struct {
			PrivateIP  string `json:"private_ip"`
			PublicIP   string `json:"public_ip"`
			PublicDNS  string `json:"public_dns"`
			PrivateDNS string `json:"private_dns"`
			Tags       struct {
				Name string `json:"Name"`
			} `json:"tags"`
		} `json:"value"`
	} `json:"agents"`
	MetricsServer struct {
		Value struct {
			PrivateIP  string `json:"private_ip"`
			PublicIP   string `json:"public_ip"`
			PublicDNS  string `json:"public_dns"`
			PrivateDNS string `json:"private_dns"`
		} `json:"value"`
	} `json:"metricsServer"`
}

// New returns a new Terraform instance.
func New(cfg *deployment.Config) *Terraform {
	return &Terraform{
		config: cfg,
	}
}

// Create creates a new load test environment.
func (t *Terraform) Create() error {
	err := t.preFlightCheck()

	if err != nil {
		return err
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
		"-var", fmt.Sprintf("app_instance_count=%d", t.config.AppInstanceCount),
		"-var", fmt.Sprintf("loadtest_agent_count=%d", t.config.AgentCount),
		"-var", fmt.Sprintf("ssh_public_key=%s", t.config.SSHPublicKey),
		"-var", fmt.Sprintf("db_instance_count=%d", t.config.DBInstanceCount),
		"-var", fmt.Sprintf("db_instance_engine=%s", t.config.DBInstanceEngine),
		"-var", fmt.Sprintf("db_instance_class=%s", t.config.DBInstanceClass),
		"-var", fmt.Sprintf("db_username=%s", t.config.DBUserName),
		"-var", fmt.Sprintf("db_password=%s", t.config.DBPassword),
		"-var", fmt.Sprintf("mattermost_download_url=%s", t.config.MattermostDownloadURL),
		"-var", fmt.Sprintf("mattermost_license_file=%s", t.config.MattermostLicenseFile),
		"-var", fmt.Sprintf("load_test_download_url=%s", loadTestDownloadURL),
		"-auto-approve",
		t.dir,
	)
	if err != nil {
		return err
	}

	output, err := t.getOutput()
	if err != nil {
		return err
	}

	// Setting up metrics server.
	if err := t.setupMetrics(extAgent, output); err != nil {
		return fmt.Errorf("error setting up metrics server: %w", err)
	}

	// Updating the config.json for each instance of app server
	t.setupAppServers(output, extAgent, uploadBinary, binaryPath)
	// Updating the nginx config on proxy server
	t.setupProxyServer(output, extAgent)

	time.Sleep(30 * time.Second)
	if err := t.createAdminUser(extAgent, output); err != nil {
		return fmt.Errorf("could not create admin user: %w", err)
	}

	if err := t.setupLoadtestAgents(extAgent, output); err != nil {
		return fmt.Errorf("error setting up loadtest agents: %w", err)
	}

	t.displayInfo(output)
	return nil
}

func (t *Terraform) setupAppServers(output *terraformOutput, extAgent *ssh.ExtAgent, uploadBinary bool, binaryPath string) {
	for _, val := range output.Instances.Value {
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

			// Upload service file
			mlog.Info("Uploading service file", mlog.String("host", ip))
			rdr := strings.NewReader(strings.TrimSpace(serviceFile))
			if out, err := sshc.Upload(rdr, "/lib/systemd/system/mattermost.service", true); err != nil {
				mlog.Error("error uploading systemd file", mlog.String("output", string(out)), mlog.Err(err))
				return
			}

			// Upload binary if needed.
			if uploadBinary {
				mlog.Info("Uploading binary", mlog.String("host", ip))
				if out, err := sshc.UploadFile(binaryPath, "/opt/mattermost/bin/mattermost", false); err != nil {
					mlog.Error("error uploading file", mlog.String("file", binaryPath), mlog.String("output", string(out)), mlog.Err(err))
					return
				}
			}

			// Starting mattermost.
			mlog.Info("Starting mattermost", mlog.String("host", ip))
			cmd := "sudo service mattermost start"
			if out, err := sshc.RunCommand(cmd); err != nil {
				mlog.Error("error running ssh command", mlog.String("cmd", cmd), mlog.String("output", string(out)), mlog.Err(err))
				return
			}
		}()
	}
}

func (t *Terraform) setupLoadtestAgents(extAgent *ssh.ExtAgent, output *terraformOutput) error {
	if err := t.configureAndRunAgents(extAgent, output); err != nil {
		return fmt.Errorf("error while setting up an agents: %w", err)
	}

	if err := t.initLoadtest(extAgent, output); err != nil {
		return err
	}

	return nil
}

func (t *Terraform) setupProxyServer(output *terraformOutput, extAgent *ssh.ExtAgent) {
	ip := output.Proxy.Value.PublicDNS
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
		for _, addr := range output.Instances.Value {
			backends += "server " + addr.PrivateIP + ":8065;\n"
		}

		files := []struct {
			content string
			path    string
		}{
			{content: strings.TrimSpace(fmt.Sprintf(nginxSiteConfig, backends)), path: "/etc/nginx/sites-available/mattermost"},
			{content: strings.TrimSpace(sysctlConfig), path: "/etc/sysctl.conf"},
			{content: strings.TrimSpace(nginxConfig), path: "/etc/nginx/nginx.conf"},
			{content: strings.TrimSpace(limitsConfig), path: "/etc/security/limits.conf"},
		}
		for _, fileInfo := range files {
			rdr := strings.NewReader(fileInfo.content)
			if out, err := sshc.Upload(rdr, fileInfo.path, true); err != nil {
				mlog.Error("error uploading file", mlog.String("output", string(out)), mlog.Err(err), mlog.String("file", fileInfo.path))
				return
			}
		}

		cmd := "sudo sysctl -p && sudo service nginx reload"
		if out, err := sshc.RunCommand(cmd); err != nil {
			mlog.Error("error running ssh command", mlog.String("output", string(out)), mlog.String("cmd", cmd), mlog.Err(err))
			return
		}

	}()
}

func (t *Terraform) createAdminUser(extAgent *ssh.ExtAgent, output *terraformOutput) error {
	cmd := fmt.Sprintf("/opt/mattermost/bin/mattermost user create --email %s --username %s --password %s --system_admin",
		t.config.AdminEmail,
		t.config.AdminUsername,
		t.config.AdminPassword,
	)
	mlog.Info("Creating admin user:", mlog.String("cmd", cmd))
	sshc, err := extAgent.NewClient(output.Instances.Value[0].PublicIP)
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

func (t *Terraform) updateAppConfig(ip string, sshc *ssh.Client, output *terraformOutput) error {
	var clusterDSN, driverName string
	var readerDSN []string
	switch t.config.DBInstanceEngine {
	case "aurora-postgresql":
		clusterDSN = "postgres://" + t.config.DBUserName + ":" + t.config.DBPassword + "@" + output.DBCluster.Value.ClusterEndpoint + "/" + t.config.ClusterName + "db?sslmode=disable"
		readerDSN = []string{"postgres://" + t.config.DBUserName + ":" + t.config.DBPassword + "@" + output.DBCluster.Value.ReaderEndpoint + "/" + t.config.ClusterName + "db?sslmode=disable"}
		driverName = "postgres"
	case "aurora-mysql":
		clusterDSN = t.config.DBUserName + ":" + t.config.DBPassword + "@tcp(" + output.DBCluster.Value.ClusterEndpoint + ")/" + t.config.ClusterName + "db?charset=utf8mb4,utf8\u0026readTimeout=30s\u0026writeTimeout=30s"
		readerDSN = []string{t.config.DBUserName + ":" + t.config.DBPassword + "@tcp(" + output.DBCluster.Value.ReaderEndpoint + ")/" + t.config.ClusterName + "db?charset=utf8mb4,utf8\u0026readTimeout=30s\u0026writeTimeout=30s"}
		driverName = "mysql"
	}

	cfg := &model.Config{}
	cfg.SetDefaults()
	cfg.ServiceSettings.ListenAddress = model.NewString(":8065")
	cfg.ServiceSettings.LicenseFileLocation = model.NewString("/home/ubuntu/mattermost.mattermost-license")
	cfg.ServiceSettings.SiteURL = model.NewString("http://" + ip + ":8065")

	cfg.SqlSettings.DriverName = model.NewString(driverName)
	cfg.SqlSettings.DataSource = model.NewString(clusterDSN)
	cfg.SqlSettings.DataSourceReplicas = readerDSN

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
		return fmt.Errorf("ssh agent not running. Please run eval \"$(ssh-agent -s)\" and then ssh-add")
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

	return t.runCommand(nil, "init", t.dir)
}

func (t *Terraform) validate() error {
	return t.runCommand(nil, "validate", t.dir)
}

func (t *Terraform) getOutput() (*terraformOutput, error) {
	var buf bytes.Buffer
	err := t.runCommand(&buf, "output", "-json")
	if err != nil {
		return nil, err
	}

	var output terraformOutput
	err = json.Unmarshal(buf.Bytes(), &output)
	if err != nil {
		return nil, err
	}
	return &output, nil
}

func (t *Terraform) displayInfo(output *terraformOutput) {
	mlog.Info("Deployment complete. Here is the setup information:")
	mlog.Info("Proxy server: " + output.Proxy.Value.PublicDNS)
	mlog.Info("Instances:")
	for _, instance := range output.Instances.Value {
		mlog.Info(instance.PublicIP)
	}
	mlog.Info("Agents:")
	for _, agent := range output.Agents.Value {
		mlog.Info(agent.Tags.Name + ": " + agent.PublicIP)
	}
	if len(output.Agents.Value) > 0 {
		mlog.Info("Coordinator:" + output.Agents.Value[0].PublicIP)
		runcmd := "go run ./cmd/ltctl"
		if strings.HasPrefix(os.Args[0], "ltctl") {
			runcmd = "ltctl"
		}
		mlog.Info(fmt.Sprintf("To start coordinator, you can use %q command.", runcmd+" loadtest start"))
	}
	mlog.Info("Metrics server: " + output.MetricsServer.Value.PublicIP)
	mlog.Info("DB reader endpoint: " + output.DBCluster.Value.ReaderEndpoint)
	mlog.Info("DB cluster endpoint: " + output.DBCluster.Value.ClusterEndpoint)
}
