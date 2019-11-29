// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"os"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/example"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-server/mlog"
	"github.com/spf13/cobra"
)

func RunLoadTest(cmd *cobra.Command, args []string) error {
	return loadtest.Run()
}

func RunExample(cmd *cobra.Command, args []string) error {
	return example.Run()
}

func main() {
	cobra.OnInitialize(initConfig)

	var rootCmd = &cobra.Command{Use: "loadtest", RunE: RunLoadTest}

	commands := make([]*cobra.Command, 1)

	commands[0] = &cobra.Command{
		Use:   "example",
		Short: "Run example implementation",
		RunE:  RunExample,
	}

	rootCmd.AddCommand(commands...)
	rootCmd.Execute()
}

func initConfig() {
	if err := config.ReadConfig(); err != nil {
		mlog.Error("Failed to initialize config", mlog.Err(err))
		os.Exit(1)
	}

	cfg, err := config.GetConfig()
	if err != nil {
		mlog.Error("Failed to get logging config: %s\n", mlog.Err(err))
		os.Exit(1)
	}

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
