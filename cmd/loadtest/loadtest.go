// Copyright (c) 2017 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package main

import (
	// "io/ioutil"
	"os"
	"strings"

	// "github.com/pkg/errors"
	"github.com/spf13/cobra"
	// "github.com/spf13/viper"

	"github.com/mattermost/mattermost-load-test/loadtest"
	"github.com/mattermost/mattermost-server/mlog"
)

func RunLoadTest(cmd *cobra.Command, args []string) error {
	return loadtest.Run()
}

func main() {
	cobra.OnInitialize(initConfig)

	var rootCmd = &cobra.Command{Use: "loadtest", RunE: RunLoadTest}

	commands := make([]*cobra.Command, 0)
	rootCmd.AddCommand(commands...)
	rootCmd.Execute()
}

func initConfig() {
	if err := loadtest.ReadConfig(); err != nil {
		mlog.Error("Failed to initialize config", mlog.Err(err))
		os.Exit(1)
	}

	cfg, err := loadtest.GetConfig()
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
