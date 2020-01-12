package main

import (
	"time"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/spf13/cobra"
)

func RunLoadTestCmdF(cmd *cobra.Command, args []string) error {
	config, err := config.GetConfig()
	if err != nil {
		return err
	}

	newSimpleController := func(id int, status chan<- control.UserStatus) control.UserController {
		ueConfig := userentity.Config{
			ServerURL:    config.ConnectionConfiguration.ServerURL,
			WebSocketURL: config.ConnectionConfiguration.WebSocketURL,
		}
		ue := userentity.New(memstore.New(), ueConfig)
		return simplecontroller.New(id, ue, status)
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

func MakeLoadTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "loadtest",
		RunE:   RunLoadTestCmdF,
		PreRun: config.Setup,
	}
	cmd.PersistentFlags().StringP("config", "c", "", "path to the configuration file to use")
	return cmd
}
