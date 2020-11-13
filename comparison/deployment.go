// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package comparison

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

func (c *Comparison) deploymentAction(action func(t *terraform.Terraform) error) error {
	var wg sync.WaitGroup
	wg.Add(len(c.deployments))
	errsCh := make(chan error, len(c.deployments))
	for id, dp := range c.deployments {
		go func(id string, dp *deploymentInfo) {
			defer wg.Done()
			t := terraform.New(id, &dp.config)
			defer t.Cleanup()
			if err := action(t); err != nil {
				errsCh <- fmt.Errorf("deployment action failed: %w", err)
			}
		}(id, dp)
	}
	wg.Wait()
	close(errsCh)
	var err error
	for err = range errsCh {
		mlog.Error(err.Error())
	}
	return err
}

// provisionBuilds loads the provided build files into the app servers to be
// used later on during initialization.
// If the build URL is an HTTP URL then the file is directly downloaded into the
// servers. If the build URL is prefixed by `file://` then the build is uploaded
// from the local filesystem.
func provisionBuilds(t *terraform.Terraform, baseBuildURL, newBuildURL string) error {
	output, err := t.Output()
	if err != nil {
		return err
	}
	if !output.HasAppServers() {
		return errors.New("no app servers in this deployment")
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}
	clients := make([]*ssh.Client, len(output.Instances))
	for i, instance := range output.Instances {
		client, err := extAgent.NewClient(instance.PublicIP)
		if err != nil {
			return fmt.Errorf("error in getting ssh connection %w", err)
		}
		defer client.Close()
		clients[i] = client
	}

	for _, client := range clients {
		for _, url := range []string{baseBuildURL, newBuildURL} {
			filePrefix := "file://"
			buildFileName := filepath.Base(url)
			if strings.HasPrefix(url, filePrefix) {
				// upload build file from local filesystem
				buildPath := strings.TrimPrefix(url, filePrefix)
				info, err := os.Stat(buildPath)
				if err != nil {
					return err
				}
				if !info.Mode().IsRegular() {
					return fmt.Errorf("build file %s has to be a regular file", buildPath)
				}
				if out, err := client.UploadFile(buildPath, "/home/ubuntu/"+buildFileName, false); err != nil {
					return fmt.Errorf("error uploading build: %w %s", err, out)
				}
			} else {
				// download build file from URL
				cmd := fmt.Sprintf("wget -O %s %s", buildFileName, url)
				if out, err := client.RunCommand(cmd); err != nil {
					return fmt.Errorf("failed to run cmd %q: %w %s", cmd, err, out)
				}
			}
		}
	}

	return nil
}
