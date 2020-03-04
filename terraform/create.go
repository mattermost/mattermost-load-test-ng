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
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/pkg/errors"
)

const cmdExecTimeoutMinutes = 5

// Terraform manages all operations related to interacting with
// an AWS environment using Terraform.
type Terraform struct {
	config *loadtest.LoadTestConfig
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
func New(cfg *loadtest.LoadTestConfig) *Terraform {
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

	err = t.runCommand(nil, "apply",
		"-var", fmt.Sprintf("cluster_name=%s", t.config.DeploymentConfiguration.ClusterName),
		"-var", fmt.Sprintf("app_instance_count=%d", t.config.DeploymentConfiguration.AppInstanceCount),
		"-var", fmt.Sprintf("ssh_public_key=%s", t.config.DeploymentConfiguration.SSHPublicKey),
		"-var", fmt.Sprintf("db_instance_count=%d", t.config.DeploymentConfiguration.DBInstanceCount),
		"-var", fmt.Sprintf("db_instance_engine=%s", t.config.DeploymentConfiguration.DBInstanceEngine),
		"-var", fmt.Sprintf("db_instance_class=%s", t.config.DeploymentConfiguration.DBInstanceClass),
		"-var", fmt.Sprintf("db_username=%s", t.config.DeploymentConfiguration.DBUserName),
		"-var", fmt.Sprintf("db_password=%s", t.config.DeploymentConfiguration.DBPassword),
		"-var", fmt.Sprintf("mattermost_download_url=%s", t.config.DeploymentConfiguration.MattermostDownloadURL),
		"-var", fmt.Sprintf("mattermost_license_file=%s", t.config.DeploymentConfiguration.MattermostLicenseFile),
		"-auto-approve",
		"./terraform",
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
		sshc, err := sshConn(ip)
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
			var dsn string
			switch t.config.DeploymentConfiguration.DBInstanceEngine {
			case "postgres":
				dsn = "postgres://" + t.config.DeploymentConfiguration.DBUserName + ":" + t.config.DeploymentConfiguration.DBPassword + "@" + output.DBEndpoint.Value + "/" + t.config.DeploymentConfiguration.ClusterName + "db?sslmode=disable"
			case "mysql":
				dsn = t.config.DeploymentConfiguration.DBUserName + ":" + t.config.DeploymentConfiguration.DBPassword + "@tcp(" + output.DBEndpoint.Value + ")/" + t.config.DeploymentConfiguration.ClusterName + "db?charset=utf8mb4,utf8\u0026readTimeout=30s\u0026writeTimeout=30s"
			}
			mlog.Info("dsn: " + dsn) // TODO: remove this later.
			for k, v := range map[string]interface{}{
				".ServiceSettings.ListenAddress":       ":8065",
				".ServiceSettings.LicenseFileLocation": "/home/ubuntu/mattermost.mattermost-license",
				".ServiceSettings.SiteURL":             "http://" + ip + ":8065",
				".SqlSettings.DriverName":              t.config.DeploymentConfiguration.DBInstanceEngine,
				".SqlSettings.DataSource":              dsn,
				".MetricsSettings.Enable":              true,
				".PluginSettings.Enable":               true,
				".PluginSettings.EnableUploads":        true,
			} {
				// TODO: wrap this functionality in a separate package.
				session, err := sshc.NewSession()
				if err != nil {
					mlog.Error("failed to create session", mlog.Err(err))
					return
				}
				func() {
					defer func() {
						err := session.Close()
						if err != nil && err != io.EOF { // Somehow it gives an io.EOF error every time. TODO: need to debug.
							mlog.Error("error closing ssh session", mlog.Err(err))
						}
					}()

					buf, err := json.Marshal(v)
					if err != nil {
						mlog.Error("invalid config", mlog.String("key", k), mlog.Err(err))
						return
					}
					cmd := fmt.Sprintf(`jq '%s = %s' /opt/mattermost/config/config.json > /tmp/mmcfg.json && mv /tmp/mmcfg.json /opt/mattermost/config/config.json`, k, string(buf))
					if err := session.Run(cmd); err != nil {
						mlog.Error("error running ssh command", mlog.String("cmd", cmd), mlog.Err(err))
						return
					}
				}()
			}

			// Starting mattermost.
			session, err := sshc.NewSession()
			if err != nil {
				mlog.Error("failed to create session", mlog.Err(err))
				return
			}
			func() {
				defer func() {
					err := session.Close()
					if err != nil && err != io.EOF { // Somehow it gives an io.EOF error every time. TODO: need to debug.
						mlog.Error("error closing ssh session", mlog.Err(err))
					}
				}()

				cmd := fmt.Sprintf(`/opt/mattermost/bin/mattermost &`) // TODO: servicify this.
				if err := session.Run(cmd); err != nil {
					mlog.Error("error running ssh command", mlog.String("cmd", cmd), mlog.Err(err))
					return
				}
			}()
		}()
	}

	return nil
}

func (t *Terraform) preFlightCheck() error {
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		return fmt.Errorf("ssh agent not running. Please run eval \"$(ssh-agent -s)\" and then ssh-add.")
	}
	if len(t.config.DeploymentConfiguration.DBPassword) < 8 {
		return fmt.Errorf("db password needs to be at least 8 characters")
	}
	clusterName := t.config.DeploymentConfiguration.ClusterName
	if len(clusterName) == 0 || clusterName[0] != '-' || !isAlphanumeric(clusterName) {
		return fmt.Errorf("db cluster name must begin with a letter and contain only alphanumeric characters")
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
		"./terraform")
}

func (t *Terraform) validate() error {
	return t.runCommand(nil, "validate",
		"./terraform")
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

	rdr := io.MultiReader(stdout, stderr)
	if dst != nil {
		_, err = io.Copy(dst, rdr)
		if err != nil {
			return err
		}
	} else {
		scanner := bufio.NewScanner(rdr)
		for scanner.Scan() {
			mlog.Info(scanner.Text())
		}
		// No need to check for scanner.Error as cmd.Wait() already does that.
	}

	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}
