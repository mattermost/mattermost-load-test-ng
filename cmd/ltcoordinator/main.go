// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/spf13/cobra"
)

func RunCoordinatorCmdF(cmd *cobra.Command, args []string) error {
	configFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}
	cfg, err := coordinator.ReadConfig(configFilePath)
	if err != nil {
		return err
	}

	ltConfigFilePath, err := cmd.Flags().GetString("ltagent-config")
	if err != nil {
		return err
	}
	ltConfig, err := loadtest.ReadConfig(ltConfigFilePath)
	if err != nil {
		return err
	}

	for i := 0; i < len(cfg.ClusterConfig.Agents); i++ {
		cfg.ClusterConfig.Agents[i].LoadTestConfig = *ltConfig
	}

	c, err := coordinator.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create coordinator: %w", err)
	}

	return c.Run()
}

func main() {
	rootCmd := &cobra.Command{
		Use:          "ltcoordinator",
		SilenceUsage: true,
		RunE:         RunCoordinatorCmdF,
		PreRunE:      initConfig,
	}
	rootCmd.PersistentFlags().StringP("config", "c", "", "path to the configuration file to use")
	rootCmd.PersistentFlags().StringP("ltagent-config", "l", "", "path to the load-test agent configuration file to use")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func initConfig(cmd *cobra.Command, args []string) error {
	configFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}
	cfg, err := coordinator.ReadConfig(configFilePath)
	if err != nil {
		return err
	}

	initLogger(cfg)

	return nil
}

func initLogger(cfg *coordinator.Config) {
	// Initalize logging
	log := mlog.NewLogger(&mlog.LoggerConfiguration{
		EnableConsole: cfg.LogSettings.EnableConsole,
		ConsoleJson:   cfg.LogSettings.ConsoleJson,
		ConsoleLevel:  strings.ToLower(cfg.LogSettings.ConsoleLevel),
		EnableFile:    cfg.LogSettings.EnableFile,
		FileJson:      cfg.LogSettings.FileJson,
		FileLevel:     strings.ToLower(cfg.LogSettings.FileLevel),
		FileLocation:  cfg.LogSettings.FileLocation,
	})

	// Redirect default golang logger to this logger
	mlog.RedirectStdLog(log)

	// Use this app logger as the global logger
	mlog.InitGlobalLogger(log)
}
