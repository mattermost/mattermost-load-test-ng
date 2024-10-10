// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/mattermost/mattermost-load-test-ng/api"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/gencontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
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

func genData(lt *loadtest.LoadTester, numUsers int) error {
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

func RunInitCmdF(cmd *cobra.Command, args []string) error {
	mlog.Info("init started")

	configFilePath, err := cmd.Flags().GetString("config")
	if err != nil {
		return err
	}

	siteURLOverride, err := cmd.Flags().GetString("site-url")
	if err != nil {
		return err
	}

	config, err := loadtest.ReadConfig(configFilePath)
	if err != nil {
		return err
	}

	if siteURLOverride != "" {
		config.ConnectionConfiguration.ServerURL = siteURLOverride
	}

	if err := defaults.Validate(*config); err != nil {
		return fmt.Errorf("could not validate configuration: %w", err)
	}

	log := logger.New(&config.LogSettings)
	defer log.Flush()

	userPrefix, err := cmd.Flags().GetString("user-prefix")
	if err != nil {
		return err
	}

	if ok, err := isInitDone(config.ConnectionConfiguration.ServerURL, userPrefix); err != nil {
		return err
	} else if ok {
		log.Warn("init already done")
		return nil
	}

	seed := memstore.SetRandomSeed()
	log.Info(fmt.Sprintf("random seed value is: %d", seed))

	genConfig := gencontroller.Config{
		NumTeams:           config.InstanceConfiguration.NumTeams,
		NumChannelsDM:      int64(float64(config.InstanceConfiguration.NumChannels) * config.InstanceConfiguration.PercentDirectChannels),
		NumChannelsGM:      int64(float64(config.InstanceConfiguration.NumChannels) * config.InstanceConfiguration.PercentGroupChannels),
		NumChannelsPrivate: int64(float64(config.InstanceConfiguration.NumChannels) * config.InstanceConfiguration.PercentPrivateChannels),
		NumChannelsPublic:  int64(float64(config.InstanceConfiguration.NumChannels) * config.InstanceConfiguration.PercentPublicChannels),
		NumPosts:           config.InstanceConfiguration.NumPosts,
		NumReactions:       config.InstanceConfiguration.NumReactions,
		PercentReplies:     config.InstanceConfiguration.PercentReplies,
		PercentUrgentPosts: config.InstanceConfiguration.PercentUrgentPosts,
		ChannelMembersDistribution: []gencontroller.ChannelMemberDistribution{
			{
				MemberLimit:     0,
				PercentChannels: 1.0,
				Probability:     1.0,
			},
		},
	}

	numUsers := 50
	config.UserControllerConfiguration.Type = loadtest.UserControllerGenerative
	config.UsersConfiguration.InitialActiveUsers = numUsers
	config.UserControllerConfiguration.RatesDistribution = []loadtest.RatesDistribution{
		{
			Rate:       0.2,
			Percentage: 1.0,
		},
	}

	newC, err := api.NewControllerWrapper(config, &genConfig, 0, userPrefix, nil)
	if err != nil {
		return fmt.Errorf("error while creating new controller: %w", err)
	}
	lt, err := loadtest.New(config, newC, log)
	if err != nil {
		return fmt.Errorf("error while initializing loadtest: %w", err)
	}

	err = genAdmins(config, userPrefix)
	if err != nil {
		return fmt.Errorf("error while generating admin users: %w", err)
	}
	log.Info("admin generation completed")

	return genData(lt, numUsers)
}

func genAdmins(config *loadtest.Config, userPrefix string) error {
	mlog.Info(fmt.Sprintf("generating %d admins", config.InstanceConfiguration.NumAdmins))

	adminStore, err := memstore.New(nil)
	if err != nil {
		return err
	}
	adminUeSetup := userentity.Setup{
		Store: adminStore,
	}
	adminUeConfig := userentity.Config{
		ServerURL:    config.ConnectionConfiguration.ServerURL,
		WebSocketURL: config.ConnectionConfiguration.WebSocketURL,
		Username:     "",
		Email:        config.ConnectionConfiguration.AdminEmail,
		Password:     config.ConnectionConfiguration.AdminPassword,
	}
	sysadmin := userentity.New(adminUeSetup, adminUeConfig)
	if err := sysadmin.Login(); err != nil {
		return err
	}

	for i := 0; i < int(config.InstanceConfiguration.NumAdmins); i++ {
		userStore, err := memstore.New(nil)
		if err != nil {
			return err
		}
		user := &model.User{
			Password: "testPass123$",
			Email:    fmt.Sprintf("%s-%d@example.com", userPrefix, i),
			Username: fmt.Sprintf("%s-%d", userPrefix, i),
		}
		ueConfig := userentity.Config{
			ServerURL:    config.ConnectionConfiguration.ServerURL,
			WebSocketURL: config.ConnectionConfiguration.WebSocketURL,
			Username:     user.Username,
			Email:        user.Email,
			Password:     user.Password,
		}

		userId, err := sysadmin.CreateUser(user)
		if err != nil {
			return err
		}

		user.Id = userId
		err = userStore.SetUser(user)
		if err != nil {
			return err
		}
		userSetup := userentity.Setup{
			Store: userStore,
		}

		err = loadtest.PromoteToAdmin(sysadmin, userentity.New(userSetup, ueConfig))
		if err != nil {
			return err
		}
	}

	if err := sysadmin.Logout(); err != nil {
		return err
	}

	return nil
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
	cmd.PersistentFlags().StringP("site-url", "", "", "an optional override for ConnectionConfiguration.ServerURL")
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
