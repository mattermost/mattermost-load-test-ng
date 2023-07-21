// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	seed := SetRandomSeed()
	fmt.Printf("Seed value is: %d\n", seed)
	os.Exit(m.Run())
}

func TestRandomUsers(t *testing.T) {
	s := newStore(t)
	myId := model.NewId()
	err := s.SetUser(&model.User{
		Id: myId,
	})
	require.NoError(t, err)

	id1 := model.NewId()
	id2 := model.NewId()
	id3 := model.NewId()
	id4 := model.NewId()
	err = s.SetUsers([]*model.User{
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

	t.Run("two users without current user", func(t *testing.T) {
		s := newStore(t)
		myId := model.NewId()
		id := model.NewId()
		id2 := model.NewId()

		err = s.SetUser(&model.User{
			Id: myId,
		})
		require.NoError(t, err)

		err = s.SetUsers([]*model.User{
			{Id: id},
			{Id: id2},
		})
		require.NoError(t, err)

		users, err := s.RandomUsers(2)
		require.NoError(t, err)
		require.Len(t, users, 2)
	})

	t.Run("two users with current user", func(t *testing.T) {
		s := newStore(t)
		myId := model.NewId()
		id := model.NewId()

		err = s.SetUser(&model.User{
			Id: myId,
		})
		require.NoError(t, err)

		err = s.SetUsers([]*model.User{
			{Id: myId},
			{Id: id},
		})
		require.NoError(t, err)

		users, err := s.RandomUsers(2)
		require.Equal(t, err, ErrLenMismatch)
		require.Empty(t, users)
	})
}

func TestRandomUser(t *testing.T) {
	t.Run("one user", func(t *testing.T) {
		s := newStore(t)
		myId := model.NewId()
		err := s.SetUser(&model.User{
			Id: myId,
		})
		require.NoError(t, err)
		id := model.NewId()
		err = s.SetUsers([]*model.User{
			{Id: id},
		})
		require.NoError(t, err)
		u, err := s.RandomUser()
		require.NoError(t, err)
		require.Equal(t, id, u.Id)
	})

	t.Run("only current user", func(t *testing.T) {
		s := newStore(t)
		myId := model.NewId()
		err := s.SetUser(&model.User{
			Id: myId,
		})
		require.NoError(t, err)
		err = s.SetUsers([]*model.User{
			{Id: myId},
		})
		require.NoError(t, err)
		u, err := s.RandomUser()
		require.Equal(t, err, ErrLenMismatch)
		require.Empty(t, u)
	})

	t.Run("two users", func(t *testing.T) {
		s := newStore(t)
		myId := model.NewId()
		err := s.SetUser(&model.User{
			Id: myId,
		})
		require.NoError(t, err)
		id1 := model.NewId()
		id2 := model.NewId()
		err = s.SetUsers([]*model.User{
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
	})

	t.Run("three users with current user", func(t *testing.T) {
		s := newStore(t)
		myId := model.NewId()
		id1 := model.NewId()
		id2 := model.NewId()
		me := &model.User{Id: myId}
		err := s.SetUser(me)
		require.NoError(t, err)
		err = s.SetUsers([]*model.User{
			{Id: id1},
			{Id: id2},
			{Id: myId},
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
	})

	t.Run("bad data", func(t *testing.T) {
		s := newStore(t)
		myId := model.NewId()
		err := s.SetUser(&model.User{
			Id: myId,
		})
		require.NoError(t, err)
		s.users[""] = nil
		u, err := s.RandomUser()
		require.Equal(t, ErrInvalidData, err)
		require.Empty(t, u)

		s.users[model.NewId()] = nil
		u, err = s.RandomUser()
		require.Equal(t, ErrInvalidData, err)
		require.Empty(t, u)

		s.users[model.NewId()] = &model.User{}
		u, err = s.RandomUser()
		require.Equal(t, ErrInvalidData, err)
		require.Empty(t, u)
	})
}

func TestRandomPost(t *testing.T) {
	t.Run("select any", func(t *testing.T) {
		s := newStore(t)
		id1 := model.NewId()
		id2 := model.NewId()
		err := s.SetPosts([]*model.Post{
			{Id: id1},
			{Id: id2},
		})
		require.NoError(t, err)
		p, err := s.RandomPost(store.SelectAny)
		require.NoError(t, err)
		assert.Condition(t, func() bool {
			switch p.Id {
			case id1, id2:
				return true
			default:
				return false
			}
		})

		s = newStore(t)
		for i := 0; i < 10; i++ {
			err := s.SetPost(&model.Post{
				Id:   model.NewId(),
				Type: "some_type",
			})
			require.NoError(t, err)
		}

		p, err = s.RandomPost(store.SelectAny)
		require.Equal(t, ErrPostNotFound, err)
		require.Empty(t, p.Clone())

		err = s.SetPost(&model.Post{
			Id: id1,
		})
		require.NoError(t, err)

		p, err = s.RandomPost(store.SelectAny)
		require.NoError(t, err)
		require.Equal(t, id1, p.Id)
	})

	t.Run("select member of", func(t *testing.T) {
		s := newStore(t)
		userId := model.NewId()
		user := model.User{Id: userId}
		err := s.SetUser(&user)
		require.NoError(t, err)

		chanId1 := model.NewId()
		chanId2 := model.NewId()
		err = s.SetChannel(&model.Channel{Id: chanId1})
		require.NoError(t, err)
		err = s.SetChannel(&model.Channel{Id: chanId2})
		require.NoError(t, err)

		postId1 := model.NewId()
		postId2 := model.NewId()
		err = s.SetPosts([]*model.Post{
			{Id: postId1, ChannelId: chanId1},
			{Id: postId2, ChannelId: chanId2},
		})
		require.NoError(t, err)

		// The user is only member of channel 1
		err = s.SetChannelMember(chanId1, &model.ChannelMember{UserId: userId, ChannelId: chanId1})
		require.NoError(t, err)

		p, err := s.RandomPost(store.SelectMemberOf)
		require.NoError(t, err)
		require.Equal(t, postId1, p.Id)
	})

	t.Run("select not DM", func(t *testing.T) {
		s := newStore(t)
		userId := model.NewId()
		user := model.User{Id: userId}
		err := s.SetUser(&user)
		require.NoError(t, err)

		chanId1 := model.NewId()
		chanId2 := model.NewId()
		err = s.SetChannel(&model.Channel{Id: chanId1, Type: model.ChannelTypeOpen})
		require.NoError(t, err)
		err = s.SetChannel(&model.Channel{Id: chanId2, Type: model.ChannelTypeDirect})
		require.NoError(t, err)

		postId1 := model.NewId()
		postId2 := model.NewId()
		err = s.SetPosts([]*model.Post{
			{Id: postId1, ChannelId: chanId1},
			{Id: postId2, ChannelId: chanId2},
		})
		require.NoError(t, err)

		p, err := s.RandomPost(store.SelectNotDirect)
		require.NoError(t, err)
		require.Equal(t, postId1, p.Id)
	})

	t.Run("select not GM", func(t *testing.T) {
		s := newStore(t)
		userId := model.NewId()
		user := model.User{Id: userId}
		err := s.SetUser(&user)
		require.NoError(t, err)

		chanId1 := model.NewId()
		chanId2 := model.NewId()
		err = s.SetChannel(&model.Channel{Id: chanId1})
		require.NoError(t, err)
		err = s.SetChannel(&model.Channel{Id: chanId2, Type: model.ChannelTypeGroup})
		require.NoError(t, err)

		postId1 := model.NewId()
		postId2 := model.NewId()
		err = s.SetPosts([]*model.Post{
			{Id: postId1, ChannelId: chanId1},
			{Id: postId2, ChannelId: chanId2},
		})
		require.NoError(t, err)

		p, err := s.RandomPost(store.SelectNotGroup)
		require.NoError(t, err)
		require.Equal(t, postId1, p.Id)
	})

	t.Run("select member of only public or private channels", func(t *testing.T) {
		s := newStore(t)
		userId := model.NewId()
		user := model.User{Id: userId}
		err := s.SetUser(&user)
		require.NoError(t, err)

		chanId1 := model.NewId()
		chanId2 := model.NewId()
		chanId3 := model.NewId()
		chanId4 := model.NewId()
		err = s.SetChannel(&model.Channel{Id: chanId1, Type: model.ChannelTypeOpen})
		require.NoError(t, err)
		err = s.SetChannel(&model.Channel{Id: chanId2, Type: model.ChannelTypePrivate})
		require.NoError(t, err)
		err = s.SetChannel(&model.Channel{Id: chanId3, Type: model.ChannelTypeDirect})
		require.NoError(t, err)
		err = s.SetChannel(&model.Channel{Id: chanId4, Type: model.ChannelTypeGroup})
		require.NoError(t, err)

		postId1 := model.NewId()
		postId2 := model.NewId()
		postId3 := model.NewId()
		postId4 := model.NewId()
		err = s.SetPosts([]*model.Post{
			{Id: postId1, ChannelId: chanId1},
			{Id: postId2, ChannelId: chanId2},
			{Id: postId3, ChannelId: chanId3},
			{Id: postId4, ChannelId: chanId4},
		})
		require.NoError(t, err)

		// The user is member of channel 1, 3 and 4
		err = s.SetChannelMember(chanId1, &model.ChannelMember{UserId: userId, ChannelId: chanId1})
		require.NoError(t, err)
		err = s.SetChannelMember(chanId3, &model.ChannelMember{UserId: userId, ChannelId: chanId3})
		require.NoError(t, err)
		err = s.SetChannelMember(chanId4, &model.ChannelMember{UserId: userId, ChannelId: chanId4})
		require.NoError(t, err)

		// Only channel 1 satisfies all requirements
		p, err := s.RandomPost(store.SelectMemberOf | store.SelectNotDirect | store.SelectNotGroup)
		require.NoError(t, err)
		require.Equal(t, postId1, p.Id)
	})

	t.Run("nothing to select", func(t *testing.T) {
		s := newStore(t)
		userId := model.NewId()
		user := model.User{Id: userId}
		err := s.SetUser(&user)
		require.NoError(t, err)

		chanId1 := model.NewId()
		chanId2 := model.NewId()
		err = s.SetChannel(&model.Channel{Id: chanId1, Type: model.ChannelTypeDirect})
		require.NoError(t, err)
		err = s.SetChannel(&model.Channel{Id: chanId2, Type: model.ChannelTypeGroup})
		require.NoError(t, err)

		postId1 := model.NewId()
		postId2 := model.NewId()
		err = s.SetPosts([]*model.Post{
			{Id: postId1, ChannelId: chanId1},
			{Id: postId2, ChannelId: chanId2},
		})
		require.NoError(t, err)

		_, err = s.RandomPost(store.SelectNotDirect | store.SelectNotGroup)
		require.ErrorIs(t, err, ErrPostNotFound)
	})
}

func TestRandomPostForChannel(t *testing.T) {
	s := newStore(t)
	post, err := s.RandomPostForChannel("someId")
	require.Empty(t, &post)
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

func TestRandomReplyPostForChannel(t *testing.T) {
	s := newStore(t)
	post, err := s.RandomReplyPostForChannel("someId")
	require.Empty(t, &post)
	require.Equal(t, ErrPostNotFound, err)

	channelId := "ch-" + model.NewId()

	id1 := model.NewId()
	id2 := model.NewId()
	id3 := model.NewId()
	rootId := model.NewId()
	err = s.SetPosts([]*model.Post{
		{
			Id:        id1,
			ChannelId: channelId,
		},
		{Id: id2},
		{
			Id:        id3,
			ChannelId: channelId,
			RootId:    rootId,
		},
	})
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		p, err := s.RandomReplyPostForChannel(channelId)
		require.NoError(t, err)
		require.Equal(t, id3, p.Id)
		require.Equal(t, rootId, p.RootId)
	}

	id4 := model.NewId()
	err = s.SetPost(&model.Post{
		Id:        id4,
		ChannelId: channelId,
		RootId:    rootId,
	})
	require.NoError(t, err)

	for i := 0; i < 10; i++ {
		p, err := s.RandomReplyPostForChannel(channelId)
		require.NoError(t, err)
		assert.Condition(t, func() bool {
			switch p.Id {
			case id3, id4:
				return true
			default:
				return false
			}
		})
	}
}

func TestRandomPostForChannelByUser(t *testing.T) {
	s := newStore(t)
	post, err := s.RandomPostForChannelByUser("chanId", "userId")
	require.Empty(t, &post)
	require.Equal(t, ErrPostNotFound, err)

	channelId := "ch-" + model.NewId()
	userId := model.NewId()

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

	post, err = s.RandomPostForChannelByUser(channelId, "userId")
	require.Empty(t, &post)
	require.Equal(t, ErrPostNotFound, err)

	id3 := model.NewId()
	err = s.SetPosts([]*model.Post{
		{
			Id:        id3,
			ChannelId: channelId,
			UserId:    userId,
		},
	})
	require.NoError(t, err)

	post, err = s.RandomPostForChannelByUser(channelId, userId)
	require.NoError(t, err)
	require.NotEmpty(t, &post)
	require.Equal(t, id3, post.Id)
}

func TestRandomEmoji(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		s := newStore(t)
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
		s := newStore(t)
		_, err := s.RandomEmoji()
		require.Equal(t, ErrEmptySlice, err)
	})
}

func TestRandomChannelMember(t *testing.T) {
	s := newStore(t)
	channelId := model.NewId()
	userId := model.NewId()
	userId2 := model.NewId()
	cms := model.ChannelMembers{
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
	s := newStore(t)
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
	t.Run("EmptyMap", func(t *testing.T) {
		_, err := pickRandomKeyFromMap(map[string]int{})
		require.Equal(t, ErrEmptyMap, err)
	})
}

var errG error

func BenchmarkRandomTeam(b *testing.B) {
	s := newStore(b)
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
		s := newStore(t)
		team, err := s.RandomTeam(store.SelectMemberOf)
		require.Error(t, err)
		require.Empty(t, team)
		require.Equal(t, ErrUserNotSet, err)
	})

	t.Run("team not found", func(t *testing.T) {
		s := newStore(t)
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
		s := newStore(t)
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
		s := newStore(t)
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
		s := newStore(t)
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
		s := newStore(t)
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
		s := newStore(t)
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
		s := newStore(t)
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
		s := newStore(t)
		channel, err := s.RandomChannel(model.NewId(), store.SelectMemberOf)
		require.Error(t, err)
		require.Empty(t, channel)
		require.Equal(t, ErrUserNotSet, err)
	})

	t.Run("team not found", func(t *testing.T) {
		s := newStore(t)
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
		s := newStore(t)
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
		s := newStore(t)
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
		err = s.SetChannelMembers(model.ChannelMembers{
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
		s := newStore(t)
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
		s := newStore(t)
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
		err = s.SetChannelMembers(model.ChannelMembers{
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

	t.Run("channel types", func(t *testing.T) {
		s := newStore(t)
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
		channelId4 := model.NewId()
		err = s.SetChannels([]*model.Channel{
			{Id: channelId1, TeamId: teamId, Type: model.ChannelTypeOpen},
			{Id: channelId2, TeamId: teamId, Type: model.ChannelTypePrivate},
			{Id: channelId3, Type: model.ChannelTypeDirect},
			{Id: channelId4, Type: model.ChannelTypeGroup},
		})
		require.NoError(t, err)
		err = s.SetChannelMembers(model.ChannelMembers{
			{
				ChannelId: channelId1,
				UserId:    user.Id,
			},
			{
				ChannelId: channelId2,
				UserId:    user.Id,
			},
			{
				ChannelId: channelId3,
				UserId:    user.Id,
			},
			{
				ChannelId: channelId4,
				UserId:    user.Id,
			},
		})
		require.NoError(t, err)

		channel, err := s.RandomChannel(teamId, store.SelectMemberOf|store.SelectNotPublic|store.SelectNotPrivate|store.SelectNotDirect)
		require.NoError(t, err)
		require.NotNil(t, channel)
		require.Equal(t, channelId4, channel.Id)

		channel, err = s.RandomChannel(teamId, store.SelectMemberOf|store.SelectNotGroup|store.SelectNotPrivate|store.SelectNotDirect)
		require.NoError(t, err)
		require.NotNil(t, channel)
		require.Equal(t, channelId1, channel.Id)

		channel, err = s.RandomChannel(teamId, store.SelectMemberOf|store.SelectNotDirect)
		require.NoError(t, err)
		require.NotNil(t, channel)
		assert.Condition(t, func() bool {
			switch channel.Id {
			case channelId1, channelId2, channelId4:
				return true
			default:
				return false
			}
		})
	})
}

func TestRandomThread(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		s := newStore(t)
		id1 := model.NewId()
		id2 := model.NewId()
		err := s.SetThreads([]*model.ThreadResponse{
			{PostId: id1},
			{PostId: id2},
		})
		fmt.Println(id1, id2)
		require.NoError(t, err)
		th, err := s.RandomThread()
		require.NoError(t, err)
		assert.Condition(t, func() bool {
			switch th.PostId {
			case id1, id2:
				return true
			default:
				return false
			}
		})
	})
	t.Run("emptyslice", func(t *testing.T) {
		s := newStore(t)
		_, err := s.RandomThread()
		require.Equal(t, ErrThreadNotFound, err)
	})
}
