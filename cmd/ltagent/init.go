// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/mattermost/mattermost-load-test-ng/api"
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

	log := logger.New(&config.LogSettings)

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

	config.UserControllerConfiguration.Type = loadtest.UserControllerGenerative
	config.UsersConfiguration.InitialActiveUsers = 0
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
	mlog.Info("admin generation completed")

	return genData(lt, 50)
}

func genAdmins(config *loadtest.Config, userPrefix string) error {
	mlog.Info(fmt.Sprintf("generating %d admins", config.InstanceConfiguration.NumAdmins))

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxConnsPerHost:       500,
		MaxIdleConns:          500,
		MaxIdleConnsPerHost:   500,
		ResponseHeaderTimeout: 5 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   1 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	adminStore, err := memstore.New(&memstore.Config{
		MaxStoredPosts:          1,
		MaxStoredUsers:          1,
		MaxStoredChannelMembers: 1,
		MaxStoredStatuses:       1,
	})
	if err != nil {
		return err
	}
	adminUeSetup := userentity.Setup{
		Store:     adminStore,
		Transport: transport,
	}
	adminUeConfig := userentity.Config{
		ServerURL:    config.ConnectionConfiguration.ServerURL,
		WebSocketURL: config.ConnectionConfiguration.WebSocketURL,
		Username:     "",
		Email:        config.ConnectionConfiguration.AdminEmail,
		Password:     config.ConnectionConfiguration.AdminPassword,
	}
	sysadmin := userentity.New(adminUeSetup, adminUeConfig)

	for i := 0; i < int(config.InstanceConfiguration.NumAdmins); i++ {
		userStore, err := memstore.New(&memstore.Config{
			MaxStoredPosts:          1,
			MaxStoredUsers:          1,
			MaxStoredChannelMembers: 1,
			MaxStoredStatuses:       1,
		})
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
			Store:     userStore,
			Transport: transport,
		}
		err = sysadmin.PromoteToAdmin(userentity.New(userSetup, ueConfig))
		if err != nil {
			return err
		}
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
