package memstore

import (
	"testing"

	"github.com/mattermost/mattermost-server/model"

	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("NewMemStore", func(t *testing.T) {
		s := New()
		require.NotNil(t, s)
	})
}

func TestUser(t *testing.T) {
	s := New()

	t.Run("NilUser", func(t *testing.T) {
		u, err := s.User()
		require.NoError(t, err)
		require.Nil(t, u)
	})

	t.Run("SetUser", func(t *testing.T) {
		err := s.SetUser(nil)
		require.Error(t, err)
		u := &model.User{}
		err = s.SetUser(u)
		require.NoError(t, err)
		uu, err := s.User()
		require.NoError(t, err)
		require.Equal(t, u, uu)
	})

	t.Run("SetPreferences", func(t *testing.T) {
		p := model.Preferences{
			{"user-id-1", "category-1", "name-1", "value-1"},
			{"user-id-2", "category-2", "name-2", "value-2"},
		}
		err := s.SetPreferences(&p)
		require.NoError(t, err)
		pp, err := s.Preferences()
		require.NoError(t, err)
		require.Equal(t, &p, pp)
	})

	t.Run("SetPost", func(t *testing.T) {
		err := s.SetPost(nil)
		require.Error(t, err)
		p := &model.Post{Id: model.NewId()}
		err = s.SetPost(p)
		require.NoError(t, err)
		uu, err := s.Post(p.Id)
		require.NoError(t, err)
		require.Equal(t, p, uu)
	})
}

func TestChannel(t *testing.T) {
	t.Run("Create channel", func(t *testing.T) {
		s := New()
		err := s.SetChannel(nil)
		require.Error(t, err)
		channel := &model.Channel{Id: model.NewId()}
		err = s.SetChannel(channel)
		require.NoError(t, err)
		c, err := s.Channel(channel.Id)
		require.NoError(t, err)
		require.Equal(t, channel, c)
	})
}

func TestId(t *testing.T) {
	s := New()

	t.Run("EmptyId", func(t *testing.T) {
		id := s.Id()
		require.Empty(t, id)
	})

	t.Run("ExpectedId", func(t *testing.T) {
		expected := model.NewId()
		require.NoError(t, s.SetUser(&model.User{
			Id: expected,
		}))
		id := s.Id()
		require.Equal(t, expected, id)
	})
}

func TestChannelMembers(t *testing.T) {
	s := New()

	t.Run("SetChannelMembers", func(t *testing.T) {
		channelId := model.NewId()
		err := s.SetChannelMembers(channelId, nil)
		require.Error(t, err)
		userId := model.NewId()
		expected := model.ChannelMembers{
			model.ChannelMember{
				ChannelId: channelId,
				UserId:    userId,
			},
		}
		err = s.SetChannelMembers(channelId, &expected)
		require.NoError(t, err)
		members, err := s.ChannelMembers(channelId)
		require.NoError(t, err)
		require.Equal(t, &expected, members)
	})

	t.Run("SetChannelMember", func(t *testing.T) {
		channelId := model.NewId()
		err := s.SetChannelMember(channelId, nil)
		require.Error(t, err)
		userId := model.NewId()
		member, err := s.ChannelMember(channelId, userId)
		require.NoError(t, err)
		require.Nil(t, member)
		expected := model.ChannelMember{
			ChannelId: channelId,
			UserId:    userId,
		}
		err = s.SetChannelMember(channelId, &expected)
		require.NoError(t, err)
		member, err = s.ChannelMember(channelId, userId)
		require.NoError(t, err)
		require.Equal(t, &expected, member)
	})

	t.Run("Remove channel members", func(t *testing.T) {
		s := New()
		channel := &model.Channel{Id: model.NewId()}
		err := s.SetChannel(channel)
		require.NoError(t, err)
		channelMember1 := model.ChannelMember{
			ChannelId: channel.Id,
			UserId:    model.NewId(),
		}
		channelMember2 := model.ChannelMember{
			ChannelId: channel.Id,
			UserId:    model.NewId(),
		}
		channelMembers := append(model.ChannelMembers{}, channelMember1, channelMember2)
		err = s.SetChannelMembers(channel.Id, &channelMembers)
		require.NoError(t, err)
		require.Equal(t, 2, len(s.channelMembers[channel.Id]))
		err = s.RemoveChannelMember(channel.Id, channelMember1.UserId)
		members, err := s.ChannelMembers(channel.Id)
		require.NoError(t, err)
		require.Equal(t, 1, len(s.channelMembers[channel.Id]))
		require.Equal(t, channelMember2, (*members)[0])
	})
}

func TestTeamMembers(t *testing.T) {
	s := New()

	t.Run("SetTeamMember", func(t *testing.T) {
		teamId := model.NewId()
		userId := model.NewId()
		member, err := s.TeamMember(teamId, userId)
		require.NoError(t, err)
		require.Nil(t, member)
		expected := model.TeamMember{
			TeamId: teamId,
			UserId: userId,
		}
		err = s.SetTeamMember(teamId, nil)
		require.Error(t, err)
		err = s.SetTeamMember(teamId, &expected)
		require.NoError(t, err)
		member, err = s.TeamMember(teamId, userId)
		require.NoError(t, err)
		require.Equal(t, &expected, member)
	})
}
