// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"os"

	"github.com/mattermost/mattermost-load-test-ng/cmd/metricswatcher/config"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:          "metricswatcher",
		RunE:         runMetricsWatcher,
		SilenceUsage: true,
		PreRun:       config.SetupMetricsCheck,
	}

	persistentFlags := rootCmd.PersistentFlags()
	persistentFlags.StringP("config", "c", "", "path to the configuration file to use")

	if err := rootCmd.Execute(); err != nil {
		mlog.Error(err.Error())
		os.Exit(1)
	}
}

func runMetricsWatcher(cmd *cobra.Command, args []string) error {
	configFilePath, _ := cmd.Flags().GetString("config")
	configuration, err := config.ReadConfig(configFilePath)
	if err != nil {
		return err
	}

	errChan := make(chan error, 1)
	defer close(errChan)

	go healthcheck(errChan, configuration)
	go checkMetrics(errChan, configuration)

	err = <-errChan

	return err
}
