// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/api"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/gencontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/mattermost/mattermost-server/v5/shared/mlog"
	"github.com/spf13/cobra"
)

func runGenLoadtest(lt *loadtest.LoadTester, numUsers int) error {
	if err := lt.Run(); err != nil {
		return err
	}

	defer func(start time.Time) {
		mlog.Info("loadtest done", mlog.String("elapsed", time.Since(start).String()))
	}(time.Now())

	for lt.Status().NumUsersStopped != int64(numUsers) {
		time.Sleep(1 * time.Second)
	}

	return lt.Stop()
}

func RunLoadTestCmdF(cmd *cobra.Command, args []string) error {
	configFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	config, err := loadtest.ReadConfig(configFilePath)
	if err != nil {
		return err
	}

	if err := defaults.Validate(*config); err != nil {
		return fmt.Errorf("could not validate configuration: %w", err)
	}

	log := logger.New(&config.LogSettings)

	controllerType := config.UserControllerConfiguration.Type

	seed := memstore.SetRandomSeed()
	mlog.Info(fmt.Sprintf("random seed value is: %d", seed))

	mlog.Info(fmt.Sprintf("will run load-test with UserController of type %s", controllerType))

	ucConfigPath, err := cmd.Flags().GetString("controller-config")
	if err != nil {
		return err
	}
	var ucConfig interface{}
	switch controllerType {
	case loadtest.UserControllerSimple:
		ucConfig, err = simplecontroller.ReadConfig(ucConfigPath)
	case loadtest.UserControllerSimulative:
		ucConfig, err = simulcontroller.ReadConfig(ucConfigPath)
	case loadtest.UserControllerGenerative:
		ucConfig, err = gencontroller.ReadConfig(ucConfigPath)
	}
	if err != nil {
		return fmt.Errorf("failed to read controller configuration: %w", err)
	}

	userPrefix, err := cmd.Flags().GetString("user-prefix")
	if err != nil {
		return err
	}

	userOffset, err := cmd.Flags().GetInt("user-offset")
	if err != nil {
		return err
	}

	numUsers, err := cmd.Flags().GetInt("num-users")
	if err != nil {
		return err
	}
	if numUsers > 0 {
		config.UsersConfiguration.InitialActiveUsers = numUsers
	}

	rate, err := cmd.Flags().GetFloat64("rate")
	if err != nil {
		return err
	}
	if rate != 1.0 {
		config.UserControllerConfiguration.RatesDistribution = []loadtest.RatesDistribution{
			{
				Rate:       rate,
				Percentage: 1.0,
			},
		}
	}

	newC, err := api.NewControllerWrapper(config, ucConfig, userOffset, userPrefix, nil)
	if err != nil {
		return fmt.Errorf("error while creating new controller: %w", err)
	}
	lt, err := loadtest.New(config, newC, log)
	if err != nil {
		return fmt.Errorf("error while initializing loadtest: %w", err)
	}

	if controllerType == loadtest.UserControllerGenerative {
		return runGenLoadtest(lt, config.UsersConfiguration.InitialActiveUsers)
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
	cmd.PersistentFlags().StringP("config", "c", "", "path to the configuration file to use")
	cmd.Flags().StringP("controller-config", "", "", "path to the controller configuration file to use")
	cmd.Flags().IntP("duration", "d", 60, "number of seconds to pass before stopping the load-test")
	cmd.Flags().IntP("num-users", "n", 0, "number of users to run, setting this value will override the config setting")
	cmd.Flags().Float64P("rate", "r", 1.0, "rate value for the controller")
	cmd.PersistentFlags().StringP("user-prefix", "", "testuser", "prefix used when generating usernames and emails")
	cmd.PersistentFlags().IntP("user-offset", "", 0, "numerical offset applied to user ids")
	return cmd
}
