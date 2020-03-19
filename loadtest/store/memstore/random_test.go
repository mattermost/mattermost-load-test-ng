// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"errors"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomUsers(t *testing.T) {
	s := New()
	id1 := model.NewId()
	id2 := model.NewId()
	id3 := model.NewId()
	id4 := model.NewId()
	err := s.SetUsers([]*model.User{
		{Id: id1},
		{Id: id2},
		{Id: id3},
		{Id: id4},
	})
	require.NoError(t, err)
	u, err := s.RandomUsers(2)
	require.NoError(t, err)
	require.Len(t, u, 2)

	_, err = s.RandomUsers(5)
	require.Equal(t, err, ErrLenMismatch)
}

func TestRandomUser(t *testing.T) {
	s := New()
	id1 := model.NewId()
	id2 := model.NewId()
	err := s.SetUsers([]*model.User{
		{Id: id1},
		{Id: id2},
	})
	require.NoError(t, err)
	u, err := s.RandomUser()
	require.NoError(t, err)
	assert.Condition(t, func() bool {
		switch u.Id {
		case id1, id2:
			return true
		default:
			return false
		}
	})
}

func TestRandomPost(t *testing.T) {
	s := New()
	id1 := model.NewId()
	id2 := model.NewId()
	err := s.SetPosts([]*model.Post{
		{Id: id1},
		{Id: id2},
	})
	require.NoError(t, err)
	p, err := s.RandomPost()
	require.NoError(t, err)
	assert.Condition(t, func() bool {
		switch p.Id {
		case id1, id2:
			return true
		default:
			return false
		}
	})
}

func TestRandomPostForChannel(t *testing.T) {
	s := New()
	post, err := s.RandomPostForChannel("someId")
	require.Empty(t, post)
	require.Equal(t, ErrPostNotFound, err)

	channelId := "ch-" + model.NewId()

	id1 := model.NewId()
	id2 := model.NewId()
	err = s.SetPosts([]*model.Post{
		{
			Id:        id1,
			ChannelId: channelId,
		},
		{Id: id2},
	})
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		p, err := s.RandomPostForChannel(channelId)
		require.NoError(t, err)
		require.Equal(t, id1, p.Id)
	}

	id3 := model.NewId()
	err = s.SetPost(&model.Post{
		Id:        id3,
		ChannelId: channelId,
	})
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		p, err := s.RandomPostForChannel(channelId)
		require.NoError(t, err)
		assert.Condition(t, func() bool {
			switch p.Id {
			case id1, id3:
				return true
			default:
				return false
			}
		})
	}
}

func TestRandomEmoji(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		s := New()
		id1 := model.NewId()
		id2 := model.NewId()
		err := s.SetEmojis([]*model.Emoji{
			{Id: id1},
			{Id: id2},
		})
		require.NoError(t, err)
		e, err := s.RandomEmoji()
		require.NoError(t, err)
		assert.Condition(t, func() bool {
			switch e.Id {
			case id1, id2:
				return true
			default:
				return false
			}
		})
	})
	t.Run("emptyslice", func(t *testing.T) {
		s := New()
		_, err := s.RandomEmoji()
		require.Equal(t, ErrEmptySlice, err)
	})
}

func TestRandomChannelMember(t *testing.T) {
	s := New()
	channelId := model.NewId()
	userId := model.NewId()
	userId2 := model.NewId()
	cms := &model.ChannelMembers{
		{
			ChannelId: channelId,
			UserId:    userId,
		},
		{
			ChannelId: channelId,
			UserId:    userId2,
		},
	}
	err := s.SetChannelMembers(cms)
	require.NoError(t, err)

	member, err := s.RandomChannelMember(channelId)
	require.NoError(t, err)
	assert.Condition(t, func() bool {
		switch member.UserId {
		case userId, userId2:
			return true
		default:
			return false
		}
	})
}

func TestRandomTeamMember(t *testing.T) {
	s := New()
	teamId := model.NewId()
	userId := model.NewId()
	userId2 := model.NewId()
	err := s.SetTeamMembers(teamId,
		[]*model.TeamMember{
			{
				TeamId: teamId,
				UserId: userId,
			},
			{
				TeamId: teamId,
				UserId: userId2,
			},
		})
	require.NoError(t, err)

	member, err := s.RandomTeamMember(teamId)
	require.NoError(t, err)
	assert.Condition(t, func() bool {
		switch member.UserId {
		case userId, userId2:
			return true
		default:
			return false
		}
	})
}

func TestPickRandomKeyFromMap(t *testing.T) {
	t.Run("Basic", func(t *testing.T) {
		m := make(map[string]int)
		m["a"] = 1
		m["b"] = 2
		key, err := pickRandomKeyFromMap(m)
		require.NoError(t, err)
		assert.Condition(t, func() bool {
			switch key.(string) {
			case "a", "b":
				return true
			default:
				return false
			}
		})
	})

	t.Run("NotMap", func(t *testing.T) {
		_, err := pickRandomKeyFromMap(1)
		require.Equal(t, err.Error(), "memstore: not a map")
	})

	t.Run("EmptyMap", func(t *testing.T) {
		_, err := pickRandomKeyFromMap(map[string]int{})
		require.Equal(t, ErrEmptyMap, err)
	})
}

var errG error

func BenchmarkRandomTeam(b *testing.B) {
	s := New()
	s.SetUser(&model.User{})
	id1 := model.NewId()
	id2 := model.NewId()
	err := s.SetTeams([]*model.Team{
		{Id: id1},
		{Id: id2},
	})
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		_, errG = s.RandomTeam(store.SelectAny)
		require.NoError(b, errG)
	}
}

func TestRandomTeam(t *testing.T) {
	t.Run("user not set", func(t *testing.T) {
		s := New()
		team, err := s.RandomTeam(store.SelectMemberOf)
		require.Error(t, err)
		require.Empty(t, team)
		require.Equal(t, ErrUserNotSet, err)
	})

	t.Run("team not found", func(t *testing.T) {
		s := New()
		user := &model.User{
			Id: model.NewId(),
		}
		err := s.SetUser(user)
		require.NoError(t, err)
		team, err := s.RandomTeam(store.SelectMemberOf)
		require.Error(t, err)
		require.Empty(t, team)
		require.Equal(t, ErrTeamStoreEmpty, err)
	})

	t.Run("select rom any", func(t *testing.T) {
		s := New()
		s.SetUser(&model.User{})
		id1 := model.NewId()
		id2 := model.NewId()
		err := s.SetTeams([]*model.Team{
			{Id: id1},
			{Id: id2},
		})
		require.NoError(t, err)
		team, err := s.RandomTeam(store.SelectAny)
		require.NoError(t, err)
		assert.Condition(t, func() bool {
			switch team.Id {
			case id1, id2:
				return true
			default:
				return false
			}
		})
	})

	t.Run("team found which user is a member of", func(t *testing.T) {
		s := New()
		user := &model.User{
			Id: model.NewId(),
		}
		err := s.SetUser(user)
		require.NoError(t, err)
		teamId1 := model.NewId()
		teamId2 := model.NewId()
		teamId3 := model.NewId()
		err = s.SetTeams([]*model.Team{
			{
				Id: teamId1,
			},
			{
				Id: teamId2,
			},
			{
				Id: teamId3,
			},
		})
		require.NoError(t, err)
		err = s.SetTeamMembers(teamId1,
			[]*model.TeamMember{
				{
					TeamId: teamId1,
					UserId: user.Id,
				},
			},
		)
		require.NoError(t, err)
		err = s.SetTeamMembers(teamId2,
			[]*model.TeamMember{
				{
					TeamId: teamId2,
					UserId: user.Id,
				},
			},
		)
		require.NoError(t, err)
		team, err := s.RandomTeam(store.SelectMemberOf)
		require.NoError(t, err)
		assert.Condition(t, func() bool {
			switch team.Id {
			case teamId1, teamId2:
				return true
			default:
				return false
			}
		})
	})

	t.Run("team found which user is not a member of", func(t *testing.T) {
		s := New()
		user := &model.User{
			Id: model.NewId(),
		}
		err := s.SetUser(user)
		require.NoError(t, err)
		teamId1 := model.NewId()
		teamId2 := model.NewId()
		err = s.SetTeams([]*model.Team{
			{
				Id: teamId1,
			},
			{
				Id: teamId2,
			},
		})
		require.NoError(t, err)
		team, err := s.RandomTeam(store.SelectNotMemberOf)
		require.NoError(t, err)
		assert.Condition(t, func() bool {
			switch team.Id {
			case teamId1, teamId2:
				return true
			default:
				return false
			}
		})
	})

	t.Run("no current team", func(t *testing.T) {
		s := New()
		user := &model.User{
			Id: model.NewId(),
		}
		err := s.SetUser(user)
		require.NoError(t, err)

		teamId1 := model.NewId()
		teamId2 := model.NewId()
		teamId3 := model.NewId()
		err = s.SetTeams([]*model.Team{
			{
				Id: teamId1,
			},
			{
				Id: teamId2,
			},
			{
				Id: teamId3,
			},
		})
		require.NoError(t, err)
		err = s.SetTeamMembers(teamId1,
			[]*model.TeamMember{
				{
					TeamId: teamId1,
					UserId: user.Id,
				},
			},
		)
		require.NoError(t, err)
		err = s.SetTeamMembers(teamId3,
			[]*model.TeamMember{
				{
					TeamId: teamId3,
					UserId: user.Id,
				},
			},
		)
		require.NoError(t, err)

		err = s.SetCurrentTeam(&model.Team{
			Id: teamId1,
		})
		require.NoError(t, err)

		for i := 0; i < 10; i++ {
			team, err := s.RandomTeam(store.SelectMemberOf | store.SelectNotCurrent)
			require.NoError(t, err)
			require.NotNil(t, team)
			require.Equal(t, teamId3, team.Id)
		}
	})
}

func TestRandomChannel(t *testing.T) {
	t.Run("basic any channel", func(t *testing.T) {
		s := New()
		s.SetUser(&model.User{})
		id1 := model.NewId()
		id2 := model.NewId()
		err := s.SetTeams([]*model.Team{
			{
				Id: "t1",
			},
		})
		require.NoError(t, err)
		err = s.SetChannels([]*model.Channel{
			{Id: id1, TeamId: "t1"},
			{Id: id2, TeamId: "t1"},
		})
		require.NoError(t, err)
		ch, err := s.RandomChannel("t1", store.SelectMemberOf|store.SelectNotMemberOf)
		require.NoError(t, err)
		assert.Condition(t, func() bool {
			switch ch.Id {
			case id1, id2:
				return true
			default:
				return false
			}
		})
	})

	t.Run("emptyslice", func(t *testing.T) {
		s := New()
		s.SetUser(&model.User{})
		err := s.SetTeams([]*model.Team{
			{
				Id: "t1",
			},
		})
		require.NoError(t, err)
		_, err = s.RandomChannel("t1", store.SelectMemberOf|store.SelectNotMemberOf)
		require.True(t, errors.Is(err, ErrChannelStoreEmpty))
	})

	t.Run("user not set", func(t *testing.T) {
		s := New()
		channel, err := s.RandomChannel(model.NewId(), store.SelectMemberOf)
		require.Error(t, err)
		require.Empty(t, channel)
		require.Equal(t, ErrUserNotSet, err)
	})

	t.Run("team not found", func(t *testing.T) {
		s := New()
		user := &model.User{
			Id: model.NewId(),
		}
		err := s.SetUser(user)
		require.NoError(t, err)
		channel, err := s.RandomChannel(model.NewId(), store.SelectMemberOf)
		require.Error(t, err)
		require.Empty(t, channel)
		require.Equal(t, ErrTeamNotFound, err)
	})

	t.Run("no channel found", func(t *testing.T) {
		s := New()
		user := &model.User{
			Id: model.NewId(),
		}
		err := s.SetUser(user)
		require.NoError(t, err)
		teamId := model.NewId()
		err = s.SetTeams([]*model.Team{
			{
				Id: teamId,
			},
		})
		require.NoError(t, err)
		err = s.SetChannels([]*model.Channel{
			{Id: model.NewId(), TeamId: teamId},
			{Id: model.NewId(), TeamId: teamId},
		})
		require.NoError(t, err)
		channel, err := s.RandomChannel(teamId, store.SelectMemberOf)
		require.Error(t, err)
		require.Empty(t, channel)
		require.Equal(t, ErrChannelStoreEmpty, err)
	})

	t.Run("channel found which is the user a member of", func(t *testing.T) {
		s := New()
		user := &model.User{
			Id: model.NewId(),
		}
		err := s.SetUser(user)
		require.NoError(t, err)
		teamId := model.NewId()
		err = s.SetTeams([]*model.Team{
			{
				Id: teamId,
			},
		})
		require.NoError(t, err)
		channelId1 := model.NewId()
		channelId2 := model.NewId()
		channelId3 := model.NewId()
		err = s.SetChannels([]*model.Channel{
			{Id: channelId1, TeamId: teamId},
			{Id: channelId2, TeamId: teamId},
			{Id: channelId3, TeamId: teamId},
		})
		require.NoError(t, err)
		err = s.SetChannelMembers(&model.ChannelMembers{
			{
				ChannelId: channelId1,
				UserId:    user.Id,
			},
			{
				ChannelId: channelId2,
				UserId:    user.Id,
			},
		})
		require.NoError(t, err)
		channel, err := s.RandomChannel(teamId, store.SelectMemberOf)
		require.NoError(t, err)
		assert.Condition(t, func() bool {
			switch channel.Id {
			case channelId1, channelId2:
				return true
			default:
				return false
			}
		})
	})

	t.Run("channel found which is the user is not a member of", func(t *testing.T) {
		s := New()
		user := &model.User{
			Id: model.NewId(),
		}
		err := s.SetUser(user)
		require.NoError(t, err)
		teamId := model.NewId()
		err = s.SetTeams([]*model.Team{
			{
				Id: teamId,
			},
		})
		require.NoError(t, err)
		channelId1 := model.NewId()
		channelId2 := model.NewId()
		err = s.SetChannels([]*model.Channel{
			{Id: channelId1, TeamId: teamId},
			{Id: channelId2, TeamId: teamId},
		})
		require.NoError(t, err)
		channel, err := s.RandomChannel(teamId, store.SelectNotMemberOf)
		require.NoError(t, err)
		assert.Condition(t, func() bool {
			switch channel.Id {
			case channelId1, channelId2:
				return true
			default:
				return false
			}
		})
	})

	t.Run("no current channel", func(t *testing.T) {
		s := New()
		user := &model.User{
			Id: model.NewId(),
		}
		err := s.SetUser(user)
		require.NoError(t, err)
		teamId := model.NewId()
		err = s.SetTeams([]*model.Team{
			{
				Id: teamId,
			},
		})
		require.NoError(t, err)
		channelId1 := model.NewId()
		channelId2 := model.NewId()
		channelId3 := model.NewId()
		err = s.SetChannels([]*model.Channel{
			{Id: channelId1, TeamId: teamId},
			{Id: channelId2, TeamId: teamId},
			{Id: channelId3, TeamId: teamId},
		})
		require.NoError(t, err)
		err = s.SetChannelMembers(&model.ChannelMembers{
			{
				ChannelId: channelId1,
				UserId:    user.Id,
			},
			{
				ChannelId: channelId3,
				UserId:    user.Id,
			},
		})
		require.NoError(t, err)

		err = s.SetCurrentChannel(&model.Channel{
			Id: channelId1,
		})
		require.NoError(t, err)

		for i := 0; i < 10; i++ {
			channel, err := s.RandomChannel(teamId, store.SelectMemberOf|store.SelectNotCurrent)
			require.NoError(t, err)
			require.NotNil(t, channel)
			require.Equal(t, channelId3, channel.Id)
		}
	})
}
