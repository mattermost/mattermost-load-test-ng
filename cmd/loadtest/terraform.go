// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/terraform"
	"github.com/mattermost/mattermost-server/v5/mlog"

	"github.com/spf13/cobra"
)

func RunDeployCmdF(cmd *cobra.Command, args []string) error {
	config, err := loadtest.GetConfig()
	if err != nil {
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

func MakeDeployCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "deploy",
		Short:  "Create and deploy a load test environment",
		RunE:   RunTerraformCmdF,
		PreRun: SetupLoadTest,
	}

	return cmd
}
