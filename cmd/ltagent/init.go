// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/spf13/cobra"
)

func createTeams(admin *userentity.UserEntity, numTeams int) error {
	team := &model.Team{
		AllowOpenInvite: true,
		Type:            "O",
	}
	for i := 0; i < numTeams; i++ {
		team.Name = fmt.Sprintf("team%d", i)
		team.DisplayName = team.Name
		id, err := admin.CreateTeam(team)
		if err != nil {
			return err
		}
		mlog.Info("team created", mlog.String("team_id", id))
		err = admin.GetTeam(id)
		if err != nil {
			return err
		}
	}
	return nil
}

func createChannels(admin *userentity.UserEntity, numChannels int) error {
	channelTypes := []string{"O", "P"}

	for i := 0; i < numChannels; i++ {
		team, err := admin.Store().RandomTeam(store.SelectAny)
		if err != nil {
			return err
		}

		id, err := admin.CreateChannel(&model.Channel{
			Name:        model.NewId(),
			DisplayName: fmt.Sprintf("ch-%d", i),
			TeamId:      team.Id,
			Type:        channelTypes[rand.Intn(len(channelTypes))],
		})
		if err != nil {
			return err
		}
		mlog.Info("channel created", mlog.String("channel_id", id))
	}
	return nil
}

func createTeamAdmins(admin *userentity.UserEntity, numUsers int, config *loadtest.Config) error {
	for i := 0; i < numUsers; i++ {
		index := i * config.InstanceConfiguration.TeamAdminInterval
		ueConfig := userentity.Config{
			ServerURL:    config.ConnectionConfiguration.ServerURL,
			WebSocketURL: config.ConnectionConfiguration.WebSocketURL,
			Username:     fmt.Sprintf("testuser-%d", index),
			Email:        fmt.Sprintf("testuser-%d@example.com", index),
			Password:     "testPass123$",
		}
		store := memstore.New()
		u := userentity.New(store, ueConfig)

		if err := u.SignUp(ueConfig.Email, ueConfig.Username, ueConfig.Password); err != nil {
			mlog.Warn("error while signing up", mlog.Err(err)) // Possibly, user already exists.
			continue
		}
		id := u.Store().Id()

		if err = admin.UpdateUserRoles(id, model.SYSTEM_USER_ROLE_ID+" "+model.TEAM_ADMIN_ROLE_ID); err != nil {
			return err
		}
		mlog.Info("user created", mlog.String("user_id", id))
	}
	return nil
}

func RunInitCmdF(cmd *cobra.Command, args []string) error {
	mlog.Info("init started")

	config, err := loadtest.GetConfig()
	if err != nil {
		return err
	}

	numTeams := config.InstanceConfiguration.NumTeams
	numChannels := config.InstanceConfiguration.NumChannels
	numTeamAdmins := config.InstanceConfiguration.NumTeamAdmins

	ueConfig := userentity.Config{
		ServerURL:    config.ConnectionConfiguration.ServerURL,
		WebSocketURL: config.ConnectionConfiguration.WebSocketURL,
	}
	store := memstore.New()
	err = store.SetUser(&model.User{
		Email:    config.ConnectionConfiguration.AdminEmail,
		Password: config.ConnectionConfiguration.AdminPassword,
	})
	if err != nil {
		return err
	}

	admin := userentity.New(store, ueConfig)

	start := time.Now()

	if err = admin.Login(); err != nil {
		return err
	}

	if err = createTeams(admin, numTeams); err != nil {
		return err
	}

	if err = createChannels(admin, numChannels); err != nil {
		return err
	}

	if err = createTeamAdmins(admin, numTeamAdmins, config); err != nil {
		return err
	}

	if _, err = admin.Logout(); err != nil {
		return err
	}

	mlog.Info("done", mlog.String("elapsed", time.Since(start).String()))

	return nil
}

func MakeInitCommand() *cobra.Command {
	return &cobra.Command{
		Use:          "init",
		Short:        "Initialize instance",
		SilenceUsage: true,
		RunE:         RunInitCmdF,
		PreRun:       SetupLoadTest,
	}
}

func SetupLoadTest(cmd *cobra.Command, args []string) {
	configFilePath, _ := cmd.Flags().GetString("config")

	if err := loadtest.ReadConfig(configFilePath); err != nil {
		mlog.Error("Failed to initialize config", mlog.Err(err))
		os.Exit(1)
	}

	cfg, err := loadtest.GetConfig()

	if err != nil {
		mlog.Error("Failed to get logging config:", mlog.Err(err))
		os.Exit(1)
	}

	logger.Init(&cfg.LogSettings)
}
