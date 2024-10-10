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

	t, err := terraform.New("", config)
	if err != nil {
		return fmt.Errorf("failed to create terraform engine: %w", err)
	}

	output, err := t.Output()
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

	appClients := make([]*ssh.Client, len(output.Instances))
	for i, instance := range output.Instances {
		client, err := extAgent.NewClient(instance.PublicIP)
		if err != nil {
			return fmt.Errorf("error in getting ssh connection %w", err)
		}
		defer client.Close()
		appClients[i] = client
	}

	agentClient, err := extAgent.NewClient(output.Agents[0].PublicIP)
	if err != nil {
		return fmt.Errorf("error in getting ssh connection %w", err)
	}
	defer agentClient.Close()

	confirmFlag, _ := cmd.Flags().GetBool("confirm")
	if !confirmFlag {
		fmt.Print("Are you sure you want to delete everything? All data will be permanently deleted? [y/N] ")
		var confirm string
		fmt.Scanln(&confirm)
		if !regexp.MustCompile(`(?i)^(y|yes){1}?$`).MatchString(confirm) {
			return nil
		}
	}

	binaryPath := "/opt/mattermost/bin/mattermost"
	mmctlPath := "/opt/mattermost/bin/mmctl"

	cmds := []struct {
		msg     string
		value   string
		clients []*ssh.Client
	}{
		{
			msg:     "Shutting down MM server on primary...",
			value:   "sudo systemctl stop mattermost",
			clients: []*ssh.Client{appClients[0]},
		},
		{
			msg:     "Resetting database...",
			value:   fmt.Sprintf("%s db reset --confirm", binaryPath),
			clients: []*ssh.Client{appClients[0]},
		},
		{
			msg:     "Restarting app servers...",
			value:   "sudo systemctl restart mattermost && until $(curl -sSf http://localhost:8065 --output /dev/null); do sleep 1; done;",
			clients: appClients,
		},
		{
			msg: "Creating sysadmin...",
			value: fmt.Sprintf("%s user create --email %s --username %s --password '%s' --system-admin --local",
				mmctlPath, config.AdminEmail, config.AdminUsername, config.AdminPassword),
			clients: []*ssh.Client{appClients[0]},
		},
		{
			msg: "Initializing data...",
			value: fmt.Sprintf("cd mattermost-load-test-ng && ./bin/ltagent init --user-prefix '%s' --site-url 'http://%s:8065'",
				output.Agents[0].Tags.Name, output.Instances[0].PrivateIP),
			clients: []*ssh.Client{agentClient},
		},
	}

	for _, c := range cmds {
		fmt.Printf(c.msg)
		for _, client := range c.clients {
			if out, err := client.RunCommand(c.value); err != nil {
				return fmt.Errorf("failed to run cmd %q: %w %s", c.value, err, out)
			}
		}
		fmt.Println(" done")
	}

	return nil
}
