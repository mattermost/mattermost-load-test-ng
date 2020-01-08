// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package memstore

import (
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomChannel(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		s := New()
		id1 := model.NewId()
		id2 := model.NewId()
		err := s.SetChannels([]*model.Channel{
			{Id: id1, TeamId: "t1"},
			{Id: id2, TeamId: "t1"},
		})
		require.NoError(t, err)
		ch, err := s.RandomChannel("t1")
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
		_, err := s.RandomChannel("t1")
		require.Equal(t, ErrEmptySlice, err)
	})
}

func TestRandomTeam(t *testing.T) {
	s := New()
	id1 := model.NewId()
	id2 := model.NewId()
	err := s.SetTeams([]*model.Team{
		{Id: id1},
		{Id: id2},
	})
	require.NoError(t, err)
	team, err := s.RandomTeam()
	require.NoError(t, err)
	assert.Condition(t, func() bool {
		switch team.Id {
		case id1, id2:
			return true
		default:
			return false
		}
	})
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
	id1 := model.NewId()
	id2 := model.NewId()
	err := s.SetTeams([]*model.Team{
		{Id: id1},
		{Id: id2},
	})
	require.NoError(b, err)

	for i := 0; i < b.N; i++ {
		_, errG = s.RandomTeam()
		require.NoError(b, errG)
	}
}
