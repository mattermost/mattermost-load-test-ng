// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"os"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/example"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/spf13/cobra"
)

func RunExampleCmdF(cmd *cobra.Command, args []string) error {
	lt := example.New("http://localhost:8065")
	return lt.Run(4)
}

func main() {
	rootCmd := MakeLoadTestCommand()

	commands := []*cobra.Command{
		&cobra.Command{
			Use:    "example",
			Short:  "Run example implementation",
			RunE:   RunExampleCmdF,
			PreRun: initLogger,
		},
		MakeInitCommand(),
		MakeServerCommand(),
	}

	rootCmd.AddCommand(commands...)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func initLogger(cmd *cobra.Command, args []string) {
	configFilePath, _ := cmd.Flags().GetString("config")
	if err := config.ReadConfig(configFilePath); err != nil {
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
