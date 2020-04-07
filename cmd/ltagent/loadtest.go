// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/noopcontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/spf13/cobra"
)

func RunLoadTestCmdF(cmd *cobra.Command, args []string) error {
	configFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	config, err := loadtest.ReadConfig(configFilePath)
	if err != nil {
		return err
	}

	if err := config.IsValid(); err != nil {
		return fmt.Errorf("could not validate configuration: %w", err)
	}

	controllerType := config.UserControllerConfiguration.Type

	seed := memstore.SetRandomSeed()
	mlog.Info(fmt.Sprintf("random seed value is: %d", seed))

	mlog.Info(fmt.Sprintf("will run load-test with UserController of type %s", controllerType))

	ucConfigPath, err := cmd.Flags().GetString("controller-config")
	if err != nil {
		return err
	}
	var ucConfig control.Config
	switch controllerType {
	case loadtest.UserControllerSimple:
		ucConfig, err = simplecontroller.ReadConfig(ucConfigPath)
	case loadtest.UserControllerSimulative:
		ucConfig, err = simulcontroller.ReadConfig(ucConfigPath)
	}
	if err != nil {
		return fmt.Errorf("failed to read controller configuration: %w", err)
	}

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
			return simplecontroller.New(id, ue, ucConfig.(*simplecontroller.Config), status)
		case loadtest.UserControllerSimulative:
			return simulcontroller.New(id, ue, ucConfig.(*simulcontroller.Config), status)
		case loadtest.UserControllerNoop:
			return noopcontroller.New(id, ue, status)
		default:
			panic("controller type must be valid")
		}
	}

	numUsers, err := cmd.Flags().GetInt("num-users")
	if err != nil {
		return err
	}
	if numUsers > 0 {
		config.UsersConfiguration.InitialActiveUsers = numUsers
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

	durationSec, err := cmd.Flags().GetInt("duration")
	if err != nil {
		return err
	}
	time.Sleep(time.Duration(durationSec) * time.Second)

	err = lt.Stop()
	mlog.Info("loadtest done", mlog.String("elapsed", time.Since(start).String()))

	return err
}

func MakeLoadTestCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "ltagent",
		RunE:         RunLoadTestCmdF,
		SilenceUsage: true,
		PreRun:       SetupLoadTest,
	}
	cmd.PersistentFlags().StringP("controller-config", "", "", "path to the controller configuration file to use")
	cmd.PersistentFlags().StringP("config", "c", "", "path to the configuration file to use")
	cmd.PersistentFlags().IntP("duration", "d", 60, "number of seconds to pass before stopping the load-test")
	cmd.PersistentFlags().IntP("num-users", "n", 0, "number of users to run")
	return cmd
}
