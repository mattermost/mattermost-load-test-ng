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
