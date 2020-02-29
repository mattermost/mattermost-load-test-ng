// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bufio"
	"context"
	"fmt"
	"io"
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
	config *loadtest.Config
}

// New returns a new Terraform instance.
func New(cfg *loadtest.Config) *Terraform {
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

	err = t.runCommand("apply",
		"-var", fmt.Sprintf("cluster_name=%s", t.config.DeploymentConfiguration.ClusterName),
		"-var", fmt.Sprintf("app_instance_count=%d", t.config.DeploymentConfiguration.AppInstanceCount),
		"-var", fmt.Sprintf("ssh_public_key=%s", t.config.DeploymentConfiguration.SSHPublicKey),
		"-var", fmt.Sprintf("db_instance_count=%d", t.config.DeploymentConfiguration.DBInstanceCount),
		"-var", fmt.Sprintf("db_instance_engine=%s", t.config.DeploymentConfiguration.DBInstanceEngine),
		"-var", fmt.Sprintf("db_instance_class=%s", t.config.DeploymentConfiguration.DBInstanceClass),
		"-var", fmt.Sprintf("db_username=%s", t.config.DeploymentConfiguration.DBUserName),
		"-var", fmt.Sprintf("db_password=%s", t.config.DeploymentConfiguration.DBPassword),
		"-auto-approve",
		"./terraform",
	)
	if err != nil {
		return err
	}
	return nil
}

func (t *Terraform) preFlightCheck() error {
	if err := t.init(); err != nil {
		return err
	}

	if err := t.validate(); err != nil {
		return err
	}
	return nil
}

func (t *Terraform) init() error {
	return t.runCommand("init",
		"./terraform")
}

func (t *Terraform) validate() error {
	return t.runCommand("validate",
		"./terraform")
}

func (t *Terraform) runCommand(args ...string) error {
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
	scanner := bufio.NewScanner(rdr)
	for scanner.Scan() {
		mlog.Info(scanner.Text())
	}
	// No need to check for scanner.Error as cmd.Wait() already does that.

	if err := cmd.Wait(); err != nil {
		return err
	}
	return nil
}
