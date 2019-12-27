// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package main

import (
	"os"
	"strings"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/example"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/spf13/cobra"
)

func newSimpleController(id int, status chan<- control.UserStatus) control.UserController {
	config, err := config.GetConfig()
	if err != nil {
		return nil
	}
	ueConfig := userentity.Config{
		ServerURL:    config.ConnectionConfiguration.ServerURL,
		WebSocketURL: config.ConnectionConfiguration.WebSocketURL,
	}
	ue := userentity.New(memstore.New(), ueConfig)
	return simplecontroller.New(id, ue, status)
}

func RunLoadTestCmdF(cmd *cobra.Command, args []string) error {
	config, err := config.GetConfig()
	if err != nil {
		return err
	}
	lt := loadtest.New(config, newSimpleController)
	start := time.Now()
	err = lt.Run()
	if err != nil {
		return err
	}
	mlog.Info("loadtest started")
	time.Sleep(60 * time.Second)
	err = lt.Stop()
	mlog.Info("loadtest done", mlog.String("elapsed", time.Since(start).String()))
	return err
}

func RunExampleCmdF(cmd *cobra.Command, args []string) error {
	lt := example.New("http://localhost:8065")
	return lt.Run(4)
}

func main() {
	rootCmd := &cobra.Command{
		Use:    "loadtest",
		RunE:   RunLoadTestCmdF,
		PreRun: initializeRootCmdF,
	}
	rootCmd.PersistentFlags().StringP("config", "c", "", "path to the configuration file to use")

	commands := make([]*cobra.Command, 1)
	commands[0] = &cobra.Command{
		Use:   "example",
		Short: "Run example implementation",
		RunE:  RunExampleCmdF,
	}

	rootCmd.AddCommand(commands...)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func initializeRootCmdF(cmd *cobra.Command, args []string) {
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
