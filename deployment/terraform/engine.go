// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-load-test-ng/deployment"

	"github.com/mattermost/mattermost-server/v6/shared/mlog"
)

const (
	// All terraform commands are based on the assetsDir through the -chdir global option
	assetsDir = "./deployment/terraform/assets"
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

	args = append([]string{"-chdir=" + assetsDir}, args...)
	mlog.Debug("Running terraform command", mlog.String("args", fmt.Sprintf("%v", args)))
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
