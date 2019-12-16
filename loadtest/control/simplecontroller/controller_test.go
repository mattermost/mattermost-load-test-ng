package simplecontroller

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost-server/v5/model"
	"github.com/stretchr/testify/require"
)

func TestSetRate(t *testing.T) {
	c := SimpleController{}

	c.Init(&userentity.UserEntity{})
	require.Equal(t, 1.0, c.rate)

	err := c.SetRate(-1.0)
	require.NotNil(t, err)
	require.Equal(t, 1.0, c.rate)

	err = c.SetRate(0.0)
	require.Nil(t, err)
	require.Equal(t, 0.0, c.rate)

	err = c.SetRate(1.5)
	require.Nil(t, err)
	require.Equal(t, 1.5, c.rate)
}

func TestReload(t *testing.T) {
	c := SimpleController{}
	c.Init(userentity.New(memstore.New(), 0, userentity.Config{
		ServerURL:    "http://localhost:8065",
		WebSocketURL: "ws://localhost:8065",
	}))

	status := c.signUp()
	require.Nil(t, status.Err)

	status = c.login()
	require.Nil(t, status.Err)
	userId := c.user.Store().Id()

	teamId, err := c.user.CreateTeam(&model.Team{
		Name:        "myteam",
		DisplayName: "myteam",
		Type:        model.TEAM_OPEN,
	})
	require.Nil(t, err)

	err = c.user.AddTeamMember(teamId, userId)
	require.Nil(t, err)

	channelId, err := c.user.CreateChannel(&model.Channel{
		Name:        "mychannel",
		DisplayName: "mychannel",
		TeamId:      teamId,
		Type:        model.CHANNEL_OPEN,
	})
	require.Nil(t, err)
	err = c.user.AddChannelMember(channelId, userId)
	require.Nil(t, err)

	channels, err := c.user.GetChannelsForTeamForUser(teamId, userId)
	require.Nil(t, err)
	for _, ch := range channels {
		t.Log(ch.DisplayName)
	}
}
