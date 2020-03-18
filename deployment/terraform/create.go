// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/pkg/errors"
)

const cmdExecTimeoutMinutes = 5

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
		sshc, err := ssh.NewClient(ip)
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

			t.updateConfig(ip, sshc, output)

			// Upload service file
			rdr := strings.NewReader(strings.TrimSpace(serviceFile))
			if err := sshc.Upload(rdr, "/lib/systemd/system/mattermost.service", true); err != nil {
				mlog.Error("error uploading systemd file", mlog.Err(err))
				return
			}

			// Upload binary if needed.
			if uploadBinary {
				if err := sshc.UploadFile(binaryPath, "/opt/mattermost/bin/mattermost", false); err != nil {
					mlog.Error("error uploading file", mlog.String("file", binaryPath), mlog.Err(err))
					return
				}
			}

			// Starting mattermost.
			cmd := fmt.Sprintf("sudo service mattermost start")
			if err := sshc.RunCommand(cmd); err != nil {
				mlog.Error("error running ssh command", mlog.String("cmd", cmd), mlog.Err(err))
				return
			}
		}()
	}

	return nil
}

func (t *Terraform) updateConfig(ip string, sshc *ssh.Client, output *terraformOutput) {
	mlog.Info("Updating config", mlog.String("host", ip))

	var dsn string
	switch t.config.DBInstanceEngine {
	case "postgres":
		dsn = "postgres://" + t.config.DBUserName + ":" + t.config.DBPassword + "@" + output.DBEndpoint.Value + "/" + t.config.ClusterName + "db?sslmode=disable"
	case "mysql":
		dsn = t.config.DBUserName + ":" + t.config.DBPassword + "@tcp(" + output.DBEndpoint.Value + ")/" + t.config.ClusterName + "db?charset=utf8mb4,utf8\u0026readTimeout=30s\u0026writeTimeout=30s"
	}
	mlog.Info("dsn: " + dsn) // TODO: remove this later.

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
			mlog.Error("invalid config", mlog.String("key", k), mlog.Err(err))
			return
		}
		cmd := fmt.Sprintf(`jq '%s = %s' /opt/mattermost/config/config.json > /tmp/mmcfg.json && mv /tmp/mmcfg.json /opt/mattermost/config/config.json`, k, string(buf))
		if err := sshc.RunCommand(cmd); err != nil {
			mlog.Error("error running ssh command", mlog.String("cmd", cmd), mlog.Err(err))
			return
		}
	}
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

// runCommand runs terraform with the args supplied. If dst is not nil, it writes the output there.
// Otherwise, it logs the output to console.
func (t *Terraform) runCommand(dst io.Writer, args ...string) error {
	terraformBin := "terraform"
	if _, err := exec.LookPath(terraformBin); err != nil {
		return errors.Wrap(err, "terraform not installed. Please install terraform. (https://www.terraform.io/downloads.html)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), cmdExecTimeoutMinutes*time.Minute)
	defer cancel()

	mlog.Info("Running terraform command", mlog.String("args", fmt.Sprintf("%v", args)))
	cmd := exec.CommandContext(ctx, terraformBin, args...)

	// If dst is set, that means we want to capture the output.
	// We write a simple case to handle that using CombinedOutput.
	if dst != nil {
		out, err := cmd.CombinedOutput()
		if err != nil {
			return err
		}
		_, err = dst.Write(out)
		return err
	}

	// From here, we want to stream the output concurrently from stderr and stdout
	// to mlog.
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			mlog.Info(scanner.Text())
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		mlog.Info(scanner.Text())
	}
	// No need to check for scanner.Error as cmd.Wait() already does that.
	wg.Wait()

	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}
