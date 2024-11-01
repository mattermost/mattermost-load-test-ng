// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-load-test-ng/deployment"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// Config returns the deployment config associated with the Terraform instance.
func (t *Terraform) Config() *deployment.Config {
	return t.config
}

// runCommand runs terraform with the args supplied.
// If dst is not nil, it writes the output there. Otherwise, it logs the output to console.
func (t *Terraform) runCommand(dst io.Writer, args ...string) error {
	terraformBin := "terraform"
	if _, err := exec.LookPath(terraformBin); err != nil {
		return fmt.Errorf("terraform not installed. Please install terraform. (https://www.terraform.io/downloads.html): %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), cmdExecTimeoutMinutes*time.Minute)
	defer cancel()

	args = append([]string{"-chdir=" + t.config.TerraformStateDir}, args...)
	mlog.Debug("Running terraform command", mlog.String("args", fmt.Sprintf("%v", args)))
	cmd := exec.CommandContext(ctx, terraformBin, args...)

	return _runCommand(cmd, dst)
}

func (t *Terraform) runAWSCommand(ctx context.Context, args []string, dst io.Writer) error {
	awsBin := "aws"
	if _, err := exec.LookPath(awsBin); err != nil {
		return fmt.Errorf("aws not installed. Please install aws. (https://aws.amazon.com/cli): %w", err)
	}

	var cancel context.CancelFunc
	if ctx == nil {
		ctx, cancel = context.WithTimeout(context.Background(), cmdExecTimeoutMinutes*time.Minute)
		defer cancel()
	}

	if t.config.AWSProfile != "" {
		args = append(args, "--profile="+t.config.AWSProfile)
	}

	mlog.Debug("Running aws command", mlog.String("args", fmt.Sprintf("%v", args)))
	cmd := exec.CommandContext(ctx, awsBin, args...)

	return _runCommand(cmd, dst)
}

type cmdLogger struct {
}

func (*cmdLogger) Write(in []byte) (int, error) {
	mlog.Info(strings.TrimSpace(string(in)))
	return len(in), nil
}

func _runCommand(cmd *exec.Cmd, dst io.Writer) error {
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

	cmd.Stdout = &cmdLogger{}
	cmd.Stderr = cmd.Stdout

	return cmd.Run()
}

func checkTerraformVersion() error {
	versionInfoJSON, err := exec.Command("terraform", "version", "-json").Output()
	if err != nil {
		return fmt.Errorf("could not run %q command: %w", "terraform version", err)
	}

	var versionInfo struct {
		Version string `json:"terraform_version"`
	}

	if err := json.Unmarshal(versionInfoJSON, &versionInfo); err != nil {
		return fmt.Errorf("could not parse terraform command output: %w", err)
	}

	installedVersion, err := semver.Parse(versionInfo.Version)
	if err != nil {
		return fmt.Errorf("could not parse installed version: %w", err)
	}

	if installedVersion.Major > requiredVersion.Major {
		return fmt.Errorf("installed major version %q is greater than supported major version %q", installedVersion.Major, requiredVersion.Major)
	}

	if installedVersion.LT(requiredVersion) {
		return fmt.Errorf("installed version %q is lower than supported version %q", installedVersion.String(), requiredVersion.String())
	}

	return nil
}

// checkAWSCLI checks that the aws command is available in the system, and that
// the profile that will be used is correctly configured
func (t *Terraform) checkAWSCLI() error {
	args := []string{"configure", "list"}
	if t.config.AWSProfile != "" {
		args = append(args, "--profile", t.Config().AWSProfile)
	}

	cmd := exec.Command("aws", args...)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("the AWS CLI is either not installed or not properly configured; error: %w", err)
	}
	return nil
}
