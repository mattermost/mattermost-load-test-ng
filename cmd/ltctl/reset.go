// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"errors"
	"fmt"
	"os"
	"regexp"

	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform/ssh"

	"github.com/spf13/cobra"
)

func RunResetCmdF(cmd *cobra.Command, args []string) error {
	if os.Getenv("SSH_AUTH_SOCK") == "" {
		return errors.New("ssh agent not running. Please run eval \"$(ssh-agent -s)\" and then ssh-add")
	}

	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	output, err := terraform.New(config).Output()
	if err != nil {
		return fmt.Errorf("could not parse output: %w", err)
	}

	if !output.HasAppServers() {
		return errors.New("no active deployment found")
	}

	extAgent, err := ssh.NewAgent()
	if err != nil {
		return err
	}

	appClient, err := extAgent.NewClient(output.Instances[0].PublicIP)
	if err != nil {
		return fmt.Errorf("error in getting ssh connection %w", err)
	}

	agentClient, err := extAgent.NewClient(output.Agents[0].PublicIP)
	if err != nil {
		return fmt.Errorf("error in getting ssh connection %w", err)
	}

	fmt.Println("Are you sure you want to delete everything? All data will be permanently deleted? [y/N]")
	var confirm string
	fmt.Scanln(&confirm)
	if !regexp.MustCompile(`(?i)^(y|yes){1}?$`).MatchString(confirm) {
		return nil
	}

	binaryPath := "/opt/mattermost/bin/mattermost"

	cmds := []struct {
		msg    string
		value  string
		client *ssh.Client
	}{
		{
			msg:    "Resetting database",
			value:  fmt.Sprintf("%s reset --confirm", binaryPath),
			client: appClient,
		},
		{
			msg: "Creating sysadmin",
			value: fmt.Sprintf("%s user create --email %s --username %s --password '%s' --system_admin",
				binaryPath, config.AdminEmail, config.AdminUsername, config.AdminPassword),
			client: appClient,
		},
		{
			msg:    "Initializing data",
			value:  fmt.Sprintf("cd mattermost-load-test-ng && ./bin/ltagent init --user-prefix '%s'", output.Agents[0].Tags.Name),
			client: agentClient,
		},
	}

	for _, c := range cmds {
		fmt.Printf(c.msg)
		if _, err := c.client.RunCommand(c.value); err != nil {
			return fmt.Errorf("failed to run cmd %q: %w", c.value, err)
		}
		fmt.Println(" done")
	}

	return nil
}
