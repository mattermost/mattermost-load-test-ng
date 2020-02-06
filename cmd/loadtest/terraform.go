// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/terraform"
	"github.com/mattermost/mattermost-server/v5/mlog"

	"github.com/spf13/cobra"
)

func RunCreateCmdF(cmd *cobra.Command, args []string) error {
	config, err := loadtest.GetConfig()
	if err != nil {
		mlog.Error(err.Error())
		return nil
	}

	if ok, err := config.IsValid(); !ok {
		mlog.Error(err.Error())
		return nil
	}

	t := terraform.New(config)
	err = t.Create()
	if err != nil {
		mlog.Error(err.Error())
	}
	return nil
}

func RunDestroyCmdF(cmd *cobra.Command, args []string) error {
	config, err := loadtest.GetConfig()
	if err != nil {
		mlog.Error(err.Error())
		return nil
	}

	if ok, err := config.IsValid(); !ok {
		mlog.Error(err.Error())
		return nil
	}

	t := terraform.New(config)
	err = t.Destroy()
	if err != nil {
		mlog.Error(err.Error())
	}
	return nil
}

func MakeEnvCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Create and deploy a load test environment",
	}

	commands := []*cobra.Command{
		{
			Use:    "create",
			Short:  "Deploy a load test environment",
			RunE:   RunCreateCmdF,
			PreRun: SetupLoadTest,
		},
		{
			Use:    "destroy",
			Short:  "Destroy a load test environment",
			RunE:   RunDestroyCmdF,
			PreRun: SetupLoadTest,
		},
	}

	cmd.AddCommand(commands...)
	return cmd
}
