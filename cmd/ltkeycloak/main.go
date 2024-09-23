// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"os"
	"time"

	"github.com/spf13/cobra"
)

const (
	// How much time to wait for a single operation to complete (all requests used during the
	// migration of an user)
	operationTimeout = 30 * time.Second
)

func MakeSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "sync",
		Short:             "Sync users between from Mattermost to Keycloak",
		PersistentPostRun: func(_ *cobra.Command, _ []string) { os.Unsetenv("MM_SERVICEENVIRONMENT") },
	}

	cmd.PersistentFlags().StringP("keycloak-host", "", "http://localhost:8484", "keycloak host")
	cmd.PersistentFlags().String("mattermost-host", "", "The Mattermost host to migrate users from")
	cmd.PersistentFlags().BoolP("dry-run", "", false, "perform a dry run without making any changes")
	cmd.PersistentFlags().BoolP("force-migrate", "", false, "Migrate all users ignoring their current auth method")

	cmd.AddCommand(MakeSyncFromMattermostCommand())
	return cmd
}

func MakeKeylcoakCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "ltkeycloak",
		SilenceUsage: true,
		Short:        "Utilities to interact with Mattermost and Keycloak deployments",
	}
	rootCmd.PersistentFlags().StringP("config", "c", "", "path to the deployer configuration file to use")

	return rootCmd
}

func main() {
	rootCmd := MakeKeylcoakCommand()
	commands := []*cobra.Command{
		MakeSyncCommand(),
	}
	rootCmd.AddCommand(commands...)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
