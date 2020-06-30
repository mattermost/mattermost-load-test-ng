// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/deployment/terraform"

	"github.com/spf13/cobra"
)

func RunLoadTestStartCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t := terraform.New(config)
	defer t.Cleanup()
	return t.StartCoordinator()
}

func RunLoadTestStopCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t := terraform.New(config)
	defer t.Cleanup()
	return t.StopCoordinator()
}

func printCoordinatorStatus(status *coordinator.Status) {
	fmt.Println("==================================================")
	fmt.Println("load-test status:")
	fmt.Println("")
	fmt.Println("State:", status.State)
	fmt.Println("Start time:", status.StartTime.Format(time.UnixDate))
	if status.State == coordinator.Done {
		fmt.Println("Stop time:", status.StopTime.Format(time.UnixDate))
		fmt.Println("Duration:", status.StopTime.Sub(status.StartTime).Round(time.Second))
	}
	fmt.Println("Active users:", status.ActiveUsers)
	fmt.Println("Number of errors:", status.NumErrors)
	if status.State == coordinator.Done {
		fmt.Println("Supported users:", status.SupportedUsers)
	}
	fmt.Println("==================================================")
}

func RunLoadTestStatusCmdF(cmd *cobra.Command, args []string) error {
	config, err := getConfig(cmd)
	if err != nil {
		return err
	}

	t := terraform.New(config)
	defer t.Cleanup()

	status, err := t.GetCoordinatorStatus()
	if err != nil {
		return err
	}

	printCoordinatorStatus(status)

	return nil
}
