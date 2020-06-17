// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/clustercontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/gencontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/noopcontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/spf13/cobra"
)

func runGenLoadtest(lt *loadtest.LoadTester, numUsers int) error {
	start := time.Now()
	if err := lt.Run(); err != nil {
		return err
	}
	mlog.Info("loadtest started")

	var err error
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			if status := lt.Status(); status.NumUsersStopped == int64(numUsers) {
				err = lt.Stop()
				return
			}
			time.Sleep(1 * time.Second)
		}
	}()
	wg.Wait()

	mlog.Info("loadtest done", mlog.String("elapsed", time.Since(start).String()))

	return err
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

	lt, err := loadtest.New(config, controllerInitializer(config, userOffset, userPrefix, ucConfig))
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
	cmd.PersistentFlags().StringP("controller-config", "", "", "path to the controller configuration file to use")
	cmd.PersistentFlags().StringP("config", "c", "", "path to the configuration file to use")
	cmd.PersistentFlags().IntP("duration", "d", 60, "number of seconds to pass before stopping the load-test")
	cmd.PersistentFlags().IntP("num-users", "n", 0, "number of users to run, setting this value will override the config setting")
	cmd.PersistentFlags().Float64P("rate", "r", 1.0, "rate value for the controller")
	cmd.PersistentFlags().StringP("user-prefix", "", "testuser", "prefix used when generating usernames and emails")
	cmd.PersistentFlags().IntP("user-offset", "", 0, "numerical offset applied to user ids")
	return cmd
}

func controllerInitializer(config *loadtest.Config, userOffset int, namePrefix string, controllerConfig interface{}) loadtest.NewController {
	return func(id int, status chan<- control.UserStatus) (control.UserController, error) {
		id += userOffset

		ueConfig := userentity.Config{
			ServerURL:    config.ConnectionConfiguration.ServerURL,
			WebSocketURL: config.ConnectionConfiguration.WebSocketURL,
			Username:     fmt.Sprintf("%s-%d", namePrefix, id),
			Email:        fmt.Sprintf("%s-%d@example.com", namePrefix, id),
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
		// http.Transport to be shared amongst all clients.
		transport := &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   1 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxConnsPerHost:       500,
			MaxIdleConns:          500,
			MaxIdleConnsPerHost:   500,
			ResponseHeaderTimeout: 5 * time.Second,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   1 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		}
		ueSetup := userentity.Setup{
			Store:     store,
			Transport: transport,
		}

		ue := userentity.New(ueSetup, ueConfig)

		switch config.UserControllerConfiguration.Type {
		case loadtest.UserControllerSimple:
			return simplecontroller.New(id, ue, controllerConfig.(*simplecontroller.Config), status)
		case loadtest.UserControllerSimulative:
			return simulcontroller.New(id, ue, controllerConfig.(*simulcontroller.Config), status)
		case loadtest.UserControllerGenerative:
			return gencontroller.New(id, ue, controllerConfig.(*gencontroller.Config), status)
		case loadtest.UserControllerNoop:
			return noopcontroller.New(id, ue, status)
		case loadtest.UserControllerCluster:
			// For cluster controller, we only use the sysadmin
			// because we are just testing system console APIs.
			ueConfig.Username = ""
			ueConfig.Email = config.ConnectionConfiguration.AdminEmail
			ueConfig.Password = config.ConnectionConfiguration.AdminPassword

			admin := userentity.New(ueSetup, ueConfig)
			return clustercontroller.New(id, admin, status)
		default:
			panic("controller type must be valid")
		}
	}
}
