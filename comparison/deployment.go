// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package comparison

import (
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func (c *Comparison) deploymentAction(action func(t *terraform.Terraform, dpConfig *deploymentConfig) error) error {
	var wg sync.WaitGroup
	wg.Add(len(c.deployments))
	errsCh := make(chan error, len(c.deployments))
	for id, dp := range c.deployments {
		go func(id string, dp *deploymentConfig) {
			defer wg.Done()
			t, err := terraform.New(id, dp.config)
			if err != nil {
				errsCh <- fmt.Errorf("failed to create terraform engine: %w", err)
				return
			}
			if err := action(t, dp); err != nil {
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

// provisionFiles loads the provided build and (optionally) db dump files
// into the app servers to be used later on during initialization.
// If the URL is an HTTP URL then the file is directly downloaded into the
// servers. If the URL is prefixed by `file://` then the file is uploaded
// from the local filesystem.
func provisionFiles(t *terraform.Terraform, dpConfig *deploymentConfig, baseBuildURL, newBuildURL string) error {
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
		for id, ltConfig := range dpConfig.loadTests {
			if ltConfig.DBDumpURL != "" {
				if err := deployment.ProvisionURL(client, ltConfig.DBDumpURL, ltConfig.getDumpFilename(id)); err != nil {
					return err
				}
			}
		}
		for _, url := range []string{baseBuildURL, newBuildURL} {
			if err := deployment.ProvisionURL(client, url, filepath.Base(url)); err != nil {
				return err
			}
		}
	}

	return nil
}
