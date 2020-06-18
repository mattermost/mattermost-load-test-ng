// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/gencontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/spf13/cobra"
)

func isInitDone(serverURL, userPrefix string) (bool, error) {
	ueConfig := userentity.Config{
		ServerURL: serverURL,
		Username:  userPrefix + "-1",
		Email:     userPrefix + "-1@example.com",
		Password:  "testPass123$",
	}
	store, err := memstore.New(nil)
	if err != nil {
		return false, err
	}
	ueSetup := userentity.Setup{
		Store:     store,
		Transport: http.DefaultTransport,
	}
	return userentity.New(ueSetup, ueConfig).Login() == nil, nil
}

func genData(lt *loadtest.LoadTester, numUsers int64) error {
	if err := lt.Run(); err != nil {
		return err
	}

	defer func(start time.Time) {
		mlog.Info("loadtest done", mlog.String("elapsed", time.Since(start).String()))
	}(time.Now())

	for lt.Status().NumUsersAdded != numUsers {
		if _, err := lt.AddUsers(10); err != nil {
			return fmt.Errorf("failed to add users %w", err)
		}
		time.Sleep(5 * time.Second)
	}

	for lt.Status().NumUsersStopped != numUsers {
		time.Sleep(1 * time.Second)
	}

	return lt.Stop()
}

func RunInitCmdF(cmd *cobra.Command, args []string) error {
	mlog.Info("init started")

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

	userPrefix, err := cmd.Flags().GetString("user-prefix")
	if err != nil {
		return err
	}

	if ok, err := isInitDone(config.ConnectionConfiguration.ServerURL, userPrefix); err != nil {
		return err
	} else if ok {
		mlog.Warn("init already done")
		return nil
	}

	seed := memstore.SetRandomSeed()
	mlog.Info(fmt.Sprintf("random seed value is: %d", seed))

	genConfig := gencontroller.Config{
		NumTeams:               config.InstanceConfiguration.NumTeams,
		NumChannels:            config.InstanceConfiguration.NumChannels,
		NumPosts:               config.InstanceConfiguration.NumPosts,
		NumReactions:           config.InstanceConfiguration.NumReactions,
		PercentReplies:         config.InstanceConfiguration.PercentReplies,
		PercentPublicChannels:  config.InstanceConfiguration.PercentPublicChannels,
		PercentPrivateChannels: config.InstanceConfiguration.PercentPrivateChannels,
		PercentDirectChannels:  config.InstanceConfiguration.PercentDirectChannels,
		PercentGroupChannels:   config.InstanceConfiguration.PercentGroupChannels,
	}

	lt, err := loadtest.New(config, newControllerWrapper(config, &genConfig, 0, userPrefix, nil))
	if err != nil {
		return fmt.Errorf("error while initializing loadtest: %w", err)
	}

	return genData(lt, 50)
}

func MakeInitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "init",
		Short:        "Initialize instance",
		SilenceUsage: true,
		RunE:         RunInitCmdF,
		PreRun:       SetupLoadTest,
	}
	cmd.PersistentFlags().StringP("user-prefix", "", "testuser", "prefix used when generating usernames and emails")
	return cmd
}

func SetupLoadTest(cmd *cobra.Command, args []string) {
	configFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		mlog.Error("failed to get config flag", mlog.Err(err))
		os.Exit(1)
	}

	cfg, err := loadtest.ReadConfig(configFilePath)
	if err != nil {
		mlog.Error("failed to read config", mlog.Err(err))
		os.Exit(1)
	}

	logger.Init(&cfg.LogSettings)
}
