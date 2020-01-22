package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
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
		team, err := admin.Store().RandomTeam()
		if err != nil {
			return err
		}

		id, err := admin.CreateChannel(&model.Channel{
			Name:   model.NewId(),
			TeamId: team.Id,
			Type:   channelTypes[rand.Intn(len(channelTypes))],
		})
		if err != nil {
			return err
		}
		mlog.Info("channel created", mlog.String("channel_id", id))
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

	if err = createChannels(admin, 10); err != nil {
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
		Use:    "init",
		Short:  "Initialize instance",
		RunE:   RunInitCmdF,
		PreRun: SetupLoadTest,
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
