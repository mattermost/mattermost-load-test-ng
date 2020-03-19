// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

const cmdExecTimeoutMinutes = 10

// TODO: fetch this dynamically. See IS-327.
const latestReleaseURL = "https://releases.mattermost.com/5.20.1/mattermost-5.20.1-linux-amd64.tar.gz"

const filePrefix = "file://"

// Terraform manages all operations related to interacting with
// an AWS environment using Terraform.
type Terraform struct {
	config *deployment.Config
}

// terraformOutput contains the output variables which are
// created after a deployment.
type terraformOutput struct {
	InstanceIps struct {
		Value []string `json:"value"`
	} `json:"instanceIPs"`
	DBEndpoint struct {
		Value string `json:"value"`
	} `json:"dbEndpoint"`
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

	err = t.runCommand(nil, "apply",
		"-var", fmt.Sprintf("cluster_name=%s", t.config.ClusterName),
		"-var", fmt.Sprintf("app_instance_count=%d", t.config.AppInstanceCount),
		"-var", fmt.Sprintf("ssh_public_key=%s", t.config.SSHPublicKey),
		"-var", fmt.Sprintf("db_instance_count=%d", t.config.DBInstanceCount),
		"-var", fmt.Sprintf("db_instance_engine=%s", t.config.DBInstanceEngine),
		"-var", fmt.Sprintf("db_instance_class=%s", t.config.DBInstanceClass),
		"-var", fmt.Sprintf("db_username=%s", t.config.DBUserName),
		"-var", fmt.Sprintf("db_password=%s", t.config.DBPassword),
		"-var", fmt.Sprintf("mattermost_download_url=%s", t.config.MattermostDownloadURL),
		"-var", fmt.Sprintf("mattermost_license_file=%s", t.config.MattermostLicenseFile),
		"-auto-approve",
		"./deployment/terraform",
	)
	if err != nil {
		return err
	}

	output, err := t.getOutput()
	if err != nil {
		return err
	}

	// Updating the config.json for each instance.
	for _, ip := range output.InstanceIps.Value {
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
			if err := t.updateConfig(ip, sshc, output); err != nil {
				mlog.Error("error updating config", mlog.Err(err))
				return
			}

			// Upload service file
			mlog.Info("Uploading service file", mlog.String("host", ip))
			rdr := strings.NewReader(strings.TrimSpace(serviceFile))
			if err := sshc.Upload(rdr, "/lib/systemd/system/mattermost.service", true); err != nil {
				mlog.Error("error uploading systemd file", mlog.Err(err))
				return
			}

			// Upload binary if needed.
			if uploadBinary {
				mlog.Info("Uploading binary", mlog.String("host", ip))
				if err := sshc.UploadFile(binaryPath, "/opt/mattermost/bin/mattermost", false); err != nil {
					mlog.Error("error uploading file", mlog.String("file", binaryPath), mlog.Err(err))
					return
				}
			}

			// Starting mattermost.
			mlog.Info("Starting mattermost", mlog.String("host", ip))
			cmd := fmt.Sprintf("sudo service mattermost start")
			if err := sshc.RunCommand(cmd); err != nil {
				mlog.Error("error running ssh command", mlog.String("cmd", cmd), mlog.Err(err))
				return
			}
		}()
	}

	// TODO: display the entire cluster info from terraformOutput later
	// when we have cluster support.
	mlog.Info("Deployment complete.")

	return nil
}

func (t *Terraform) updateConfig(ip string, sshc *ssh.Client, output *terraformOutput) error {
	var dsn string
	switch t.config.DBInstanceEngine {
	case "postgres":
		dsn = "postgres://" + t.config.DBUserName + ":" + t.config.DBPassword + "@" + output.DBEndpoint.Value + "/" + t.config.ClusterName + "db?sslmode=disable"
	case "mysql":
		dsn = t.config.DBUserName + ":" + t.config.DBPassword + "@tcp(" + output.DBEndpoint.Value + ")/" + t.config.ClusterName + "db?charset=utf8mb4,utf8\u0026readTimeout=30s\u0026writeTimeout=30s"
	}

	for k, v := range map[string]interface{}{
		".ServiceSettings.ListenAddress":       ":8065",
		".ServiceSettings.LicenseFileLocation": "/home/ubuntu/mattermost.mattermost-license",
		".ServiceSettings.SiteURL":             "http://" + ip + ":8065",
		".SqlSettings.DriverName":              t.config.DBInstanceEngine,
		".SqlSettings.DataSource":              dsn,
		".MetricsSettings.Enable":              true,
		".PluginSettings.Enable":               true,
		".PluginSettings.EnableUploads":        true,
	} {
		buf, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("invalid config: key: %s, err: %v", k, err)
		}
		cmd := fmt.Sprintf(`jq '%s = %s' /opt/mattermost/config/config.json > /tmp/mmcfg.json && mv /tmp/mmcfg.json /opt/mattermost/config/config.json`, k, string(buf))
		if err := sshc.RunCommand(cmd); err != nil {
			return fmt.Errorf("error running ssh command: cmd: %s, err: %v", cmd, err)
		}
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
	return t.runCommand(nil, "init",
		"./deployment/terraform")
}

func (t *Terraform) validate() error {
	return t.runCommand(nil, "validate",
		"./deployment/terraform")
}

func (t *Terraform) getOutput() (*terraformOutput, error) {
	var buf bytes.Buffer
	err := t.runCommand(&buf, "output",
		"-json")
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
