// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/gencontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/spf13/cobra"
)

func initDone(serverURL, userPrefix string) (bool, error) {
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
		lt.AddUsers(10)
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

	if err := config.IsValid(); err != nil {
		return fmt.Errorf("could not validate configuration: %w", err)
	}

	userPrefix, err := cmd.Flags().GetString("user-prefix")
	if err != nil {
		return err
	}

	if ok, err := initDone(config.ConnectionConfiguration.ServerURL, userPrefix); err != nil {
		return err
	} else if ok {
		mlog.Warn("init already done")
		return nil
	}

	seed := memstore.SetRandomSeed()
	mlog.Info(fmt.Sprintf("random seed value is: %d", seed))

	// http.Transport to be shared amongst all clients.
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   1 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxConnsPerHost:       100,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   100,
		ResponseHeaderTimeout: 10 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   1 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	genConfig := config.InstanceConfiguration

	newControllerFn := func(id int, status chan<- control.UserStatus) (control.UserController, error) {
		ueConfig := userentity.Config{
			ServerURL:    config.ConnectionConfiguration.ServerURL,
			WebSocketURL: config.ConnectionConfiguration.WebSocketURL,
			Username:     fmt.Sprintf("%s-%d", userPrefix, id),
			Email:        fmt.Sprintf("%s-%d@example.com", userPrefix, id),
			Password:     "testPass123$",
		}
		store, err := memstore.New(&memstore.Config{
			MaxStoredPosts:          500,
			MaxStoredUsers:          1000,
			MaxStoredChannelMembers: 1000,
			MaxStoredStatuses:       1000,
		})
		if err != nil {
			return nil, err
		}
		ueSetup := userentity.Setup{
			Store:     store,
			Transport: transport,
		}
		ue := userentity.New(ueSetup, ueConfig)
		return gencontroller.New(id, ue, &genConfig, status)
	}

	config.UsersConfiguration.InitialActiveUsers = 0
	config.UserControllerConfiguration.RatesDistribution = []struct {
		Rate       float64
		Percentage float64
	}{
		{
			Rate:       0.2,
			Percentage: 1.0,
		},
	}

	lt, err := loadtest.New(config, newControllerFn)
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
