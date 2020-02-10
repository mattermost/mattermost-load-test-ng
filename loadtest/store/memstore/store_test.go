// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"testing"

	"github.com/mattermost/mattermost-server/v5/model"

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

	t.Run("SetUserPrivateData", func(t *testing.T) {
		authdata := "authdata"
		u := &model.User{
			Password:           "password",
			LastPasswordUpdate: 100,
			FirstName:          "firstname",
			LastName:           "lastname",
			AuthData:           &authdata,
			MfaSecret:          "mfasecret",
			Email:              "test@example.com",
			AuthService:        "authservice",
		}

		err := s.SetUser(u)
		require.NoError(t, err)
		u2, err := s.User()
		require.NoError(t, err)
		require.Equal(t, u, u2)

		err = s.SetUser(&model.User{})
		require.NoError(t, err)
		u3, err := s.User()
		require.NoError(t, err)
		require.Equal(t, u, u3)
	})

	t.Run("SetUsers", func(t *testing.T) {
		usrs := []*model.User{
			{Id: model.NewId()},
			{Id: model.NewId()},
			{Id: model.NewId()},
		}
		err := s.SetUsers(usrs)
		require.NoError(t, err)
		uusrs, err := s.Users()
		require.NoError(t, err)
		require.ElementsMatch(t, usrs, uusrs)
	})

	t.Run("SetPreferences", func(t *testing.T) {
		p := model.Preferences{
			{UserId: "user-id-1", Category: "category-1", Name: "name-1", Value: "value-1"},
			{UserId: "user-id-2", Category: "category-2", Name: "name-2", Value: "value-2"},
		}
		err := s.SetPreferences(&p)
		require.NoError(t, err)
		pp, err := s.Preferences()
		require.NoError(t, err)
		require.Equal(t, p, pp)
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

	t.Run("SetPosts", func(t *testing.T) {
		err := s.SetPosts(nil)
		require.Error(t, err)
		err = s.SetPosts([]*model.Post{})
		require.Error(t, err)
		p := []*model.Post{{Id: model.NewId()}}
		err = s.SetPosts(p)
		require.NoError(t, err)
		uu, err := s.Post(p[0].Id)
		require.NoError(t, err)
		require.Equal(t, p[0], uu)
	})

	t.Run("ChannelPosts", func(t *testing.T) {
		channelId := model.NewId()
		postId := model.NewId()
		err := s.SetPosts([]*model.Post{})
		require.Error(t, err)
		channelPosts, err := s.ChannelPosts(channelId)
		require.NoError(t, err)
		require.Nil(t, channelPosts)
		p := []*model.Post{
			{Id: postId, ChannelId: channelId},
			{Id: model.NewId(), ChannelId: model.NewId()},
		}
		err = s.SetPosts(p)
		require.NoError(t, err)
		channelPosts, err = s.ChannelPosts(channelId)
		require.NoError(t, err)
		require.Equal(t, len(channelPosts), 1)
		require.Equal(t, postId, channelPosts[0].Id)
	})

	t.Run("ChannelPostsSorted", func(t *testing.T) {
		cleanStore := New()
		channelId := model.NewId()
		p := []*model.Post{
			{Id: model.NewId(), ChannelId: channelId, CreateAt: 123},
			{Id: model.NewId(), ChannelId: channelId, CreateAt: 234},
		}
		require.NoError(t, cleanStore.SetPosts(p))
		channelPosts, err := cleanStore.ChannelPostsSorted(channelId, true)
		require.NoError(t, err)
		require.Len(t, channelPosts, 2)
		require.Equal(t, p[0].Id, channelPosts[0].Id)
		require.Equal(t, p[1].Id, channelPosts[1].Id)

		channelPosts, err = cleanStore.ChannelPostsSorted(channelId, false)
		require.NoError(t, err)
		require.Len(t, channelPosts, 2)
		require.Equal(t, p[1].Id, channelPosts[0].Id)
		require.Equal(t, p[0].Id, channelPosts[1].Id)
	})

	t.Run("PostsSince", func(t *testing.T) {
		posts := make([]*model.Post, 10)
		for i := 0; i < 10; i++ {
			posts[i] = &model.Post{
				Id:       model.NewId(),
				CreateAt: int64(i),
			}
		}
		postsSince, err := s.PostsSince(0)
		require.NoError(t, err)
		require.Empty(t, postsSince)
		err = s.SetPosts(posts)
		require.NoError(t, err)
		postsSince, err = s.PostsSince(5)
		require.NoError(t, err)
		require.Len(t, postsSince, 4)
		for i := 6; i < 10; i++ {
			require.Contains(t, postsSince, *posts[i])
		}
	})

	t.Run("SetReactions", func(t *testing.T) {
		postId := model.NewId()
		userId := model.NewId()
		emojiName := "testemoji"
		reaction := &model.Reaction{
			UserId:    userId,
			PostId:    postId,
			EmojiName: emojiName,
		}
		err := s.SetReactions(postId, []*model.Reaction{reaction})
		require.NoError(t, err)
		reactions, err := s.Reactions(postId)
		require.NoError(t, err)
		require.Equal(t, *reaction, reactions[0])
	})

	t.Run("DeleteReaction", func(t *testing.T) {
		postId := model.NewId()
		userId := model.NewId()
		emojiName := "testemoji"

		r1 := &model.Reaction{
			UserId:    userId,
			PostId:    postId,
			EmojiName: emojiName,
		}

		reactions := make([]*model.Reaction, 10)

		for i := range reactions {
			reactions[i] = &model.Reaction{
				UserId:    model.NewId(),
				PostId:    postId,
				EmojiName: emojiName,
			}
		}

		err := s.SetReactions(postId, reactions)
		require.NoError(t, err)

		ok, err := s.DeleteReaction(r1)
		require.NoError(t, err)
		require.False(t, ok)

		reactions[4] = r1

		err = s.SetReactions(postId, reactions)
		require.NoError(t, err)

		ok, err = s.DeleteReaction(r1)
		require.NoError(t, err)
		require.True(t, ok)

		storedReactions, err := s.Reactions(postId)
		require.NoError(t, err)
		require.Len(t, storedReactions, 9)
		require.NotContains(t, storedReactions, *r1)
	})

	t.Run("SetTeam", func(t *testing.T) {
		tm := &model.Team{Id: model.NewId()}
		err := s.SetTeam(tm)
		require.NoError(t, err)
		tt, err := s.Team(tm.Id)
		require.NoError(t, err)
		require.Equal(t, tm, tt)
	})

	t.Run("SetTeams", func(t *testing.T) {
		tms := []*model.Team{
			{Id: model.NewId()},
			{Id: model.NewId()},
			{Id: model.NewId()},
		}
		tmsV := make([]model.Team, len(tms))
		for i, tm := range tms {
			tmsV[i] = *tm
		}
		err := s.SetTeams(tms)
		require.NoError(t, err)
		ttms, err := s.Teams()
		require.NoError(t, err)
		require.ElementsMatch(t, tmsV, ttms)
	})
}

func TestChannel(t *testing.T) {
	t.Run("Store channel", func(t *testing.T) {
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

	t.Run("Store channels", func(t *testing.T) {
		s := New()
		err := s.SetChannels(nil)
		require.Error(t, err)
		channel := &model.Channel{Id: model.NewId()}
		err = s.SetChannels([]*model.Channel{channel})
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
		err := s.SetChannelMembers(nil)
		require.Error(t, err)
		userId := model.NewId()
		expected := model.ChannelMembers{
			model.ChannelMember{
				ChannelId: channelId,
				UserId:    userId,
			},
		}
		err = s.SetChannelMembers(&expected)
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
		require.Empty(t, member.UserId)
		expected := model.ChannelMember{
			ChannelId: channelId,
			UserId:    userId,
		}
		err = s.SetChannelMember(channelId, &expected)
		require.NoError(t, err)
		member, err = s.ChannelMember(channelId, userId)
		require.NoError(t, err)
		require.Equal(t, expected, member)
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
		err = s.SetChannelMembers(&channelMembers)
		require.NoError(t, err)
		require.Equal(t, 2, len(s.channelMembers[channel.Id]))
		err = s.RemoveChannelMember(channel.Id, channelMember1.UserId)
		require.NoError(t, err)
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
		require.Empty(t, member.UserId)
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
		require.Equal(t, expected, member)
	})

	t.Run("SetTeamMembers", func(t *testing.T) {
		teamId := model.NewId()
		userId := model.NewId()
		member, err := s.TeamMember(teamId, userId)
		require.NoError(t, err)
		require.Empty(t, member.UserId)
		expected := model.TeamMember{
			TeamId: teamId,
			UserId: userId,
		}
		err = s.SetTeamMembers(teamId, []*model.TeamMember{&expected})
		require.NoError(t, err)
		member, err = s.TeamMember(teamId, userId)
		require.NoError(t, err)
		require.Equal(t, expected, member)
	})
}

func TestConfig(t *testing.T) {
	s := New()

	t.Run("SetConfig", func(t *testing.T) {
		config := &model.Config{}
		s.SetConfig(config)
		require.Equal(t, s.Config(), *config)
	})
}
