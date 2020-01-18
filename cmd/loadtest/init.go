package main

import (
	"fmt"
	"github.com/mattermost/mattermost-load-test-ng/cmd/loadtest/config"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"

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
	}
	return nil
}

func RunInitCmdF(cmd *cobra.Command, args []string) error {
	mlog.Info("init started")

	config, err := config.GetConfig()
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
		PreRun: config.SetupLoadTest,
	}
}
