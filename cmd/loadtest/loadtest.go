// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/spf13/cobra"
)

func RunLoadTestCmdF(cmd *cobra.Command, args []string) error {
	config, err := loadtest.GetConfig()
	if err != nil {
		return err
	}

	if err := config.IsValid(); err != nil {
		return fmt.Errorf("could not validate configuration: %w", err)
	}

	controllerType := config.UserControllerConfiguration.Type

	mlog.Info(fmt.Sprintf("will run load-test with UserController of type %s", controllerType))

	newControllerFn := func(id int, status chan<- control.UserStatus) (control.UserController, error) {
		ueConfig := userentity.Config{
			ServerURL:    config.ConnectionConfiguration.ServerURL,
			WebSocketURL: config.ConnectionConfiguration.WebSocketURL,
			Username:     fmt.Sprintf("testuser-%d", id),
			Email:        fmt.Sprintf("testuser-%d@example.com", id),
			Password:     "testPass123$",
		}
		ue := userentity.New(memstore.New(), ueConfig)
		switch controllerType {
		case loadtest.UserControllerSimple:
			path, err := cmd.Flags().GetString("simplecontroller-config")
			if err != nil {
				return nil, err
			}
			cfg, err := simplecontroller.ReadConfig(path)
			if err != nil {
				return nil, err
			}
			return simplecontroller.New(id, ue, cfg, status)
		case loadtest.UserControllerSimulative:
			return simulcontroller.New(id, ue, status)
		default:
			panic("controller type must be valid")
		}
	}

	lt, err := loadtest.New(config, newControllerFn)
	if err != nil {
		return fmt.Errorf("error while initializing loadtest: %w", err)
	}

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
		PreRun: SetupLoadTest,
	}
	cmd.PersistentFlags().StringP("simplecontroller-config", "s", "", "path to the simplecontroller configuration file to use")
	cmd.PersistentFlags().StringP("config", "c", "", "path to the configuration file to use")
	return cmd
}
