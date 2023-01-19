// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/mattermost/mattermost-server/v6/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newStore(tb testing.TB) *MemStore {
	tb.Helper()
	s, err := New(nil)
	require.NoError(tb, err)
	require.NotNil(tb, s)
	return s
}

func TestNew(t *testing.T) {
	t.Run("NewMemStore", func(t *testing.T) {
		s := newStore(t)
		require.NotNil(t, s)
	})
}

func TestUser(t *testing.T) {
	s := newStore(t)

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
		var uusrs []*model.User
		for _, u := range s.users {
			uusrs = append(uusrs, u)
		}
		require.ElementsMatch(t, usrs, uusrs)
	})

	t.Run("SetPreferences", func(t *testing.T) {
		p := model.Preferences{
			{UserId: "user-id-1", Category: "category-1", Name: "name-1", Value: "value-1"},
			{UserId: "user-id-2", Category: "category-2", Name: "name-2", Value: "value-2"},
		}
		err := s.SetPreferences(p)
		require.NoError(t, err)
		pp, err := s.Preferences()
		require.NoError(t, err)
		require.Equal(t, p, pp)
	})

	t.Run("Post", func(t *testing.T) {
		p, err := s.Post("someid")
		require.Empty(t, p)
		require.Equal(t, ErrPostNotFound, err)

		err = s.SetPost(&model.Post{Id: "someid"})
		require.NoError(t, err)

		p, err = s.Post("someid")
		require.NoError(t, err)
		require.Equal(t, "someid", p.Id)
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

	t.Run("DeletePost", func(t *testing.T) {
		s := newStore(t)
		require.Empty(t, s.posts)
		p := &model.Post{Id: model.NewId()}
		err := s.SetPost(p)
		require.NoError(t, err)
		require.NotEmpty(t, s.posts)
		err = s.DeletePost(p.Id)
		require.NoError(t, err)
		require.Empty(t, s.posts)
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
		cleanStore := newStore(t)
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

	t.Run("PostsIdsSince", func(t *testing.T) {
		posts := make([]*model.Post, 10)
		for i := 0; i < 10; i++ {
			posts[i] = &model.Post{
				Id:       model.NewId(),
				CreateAt: int64(i),
			}
		}
		postsIdsSince, err := s.PostsIdsSince(0)
		require.NoError(t, err)
		require.Empty(t, postsIdsSince)
		err = s.SetPosts(posts)
		require.NoError(t, err)
		postsIdsSince, err = s.PostsIdsSince(5)
		require.NoError(t, err)
		require.Len(t, postsIdsSince, 4)
		for i := 6; i < 10; i++ {
			require.Contains(t, postsIdsSince, posts[i].Id)
		}
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

func TestStoreConsistency(t *testing.T) {
	config := &Config{}
	config.SetDefaults()
	t.Run("Posts", func(t *testing.T) {
		config.MaxStoredPosts = 3
		s, err := New(config)
		require.NotNil(t, s)
		require.NoError(t, err)

		for i := 0; i < config.MaxStoredPosts; i++ {
			err := s.SetPost(&model.Post{Id: fmt.Sprintf("%d", i+1)})
			require.NoError(t, err)
		}

		require.Len(t, s.postsQueue.data, config.MaxStoredPosts)
		require.Len(t, s.posts, config.MaxStoredPosts)

		for i := 0; i < config.MaxStoredPosts; i++ {
			id := fmt.Sprintf("%d", i+1)
			require.Equal(t, id, s.posts[id].Id)
		}

		s.SetPost(&model.Post{Id: "1"})
		s.SetPost(&model.Post{Id: "2"})
		s.SetPost(&model.Post{Id: "1"})
		s.DeletePost("2")
		s.SetPost(&model.Post{Id: "3"})

		require.Len(t, s.postsQueue.data, config.MaxStoredPosts)
		require.Len(t, s.posts, config.MaxStoredPosts-1)

		p, err := s.Post("1")
		require.NoError(t, err)
		require.NotNil(t, p)
	})

	t.Run("Lengths", func(t *testing.T) {
		config.MaxStoredPosts = 100
		s, err := New(config)
		require.NotNil(t, s)
		require.NoError(t, err)

		for i := 0; i < config.MaxStoredPosts; i++ {
			err := s.SetPost(&model.Post{Id: fmt.Sprintf("%d", i+1)})
			require.NoError(t, err)
		}

		require.Len(t, s.posts, config.MaxStoredPosts)
		require.Len(t, s.postsQueue.data, config.MaxStoredPosts)

		for i := 0; i < config.MaxStoredPosts; i++ {
			err := s.DeletePost(fmt.Sprintf("%d", i+1))
			require.NoError(t, err)
		}

		require.Len(t, s.posts, 0)
		require.Len(t, s.postsQueue.data, config.MaxStoredPosts)
	})
}

func TestChannel(t *testing.T) {
	t.Run("Store channel", func(t *testing.T) {
		s := newStore(t)
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
		s := newStore(t)
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

func TestCurrentChannel(t *testing.T) {
	s := newStore(t)
	channel, err := s.CurrentChannel()
	require.Nil(t, channel)
	require.EqualError(t, err, ErrChannelNotFound.Error())
	err = s.SetCurrentChannel(nil)
	require.Error(t, err)
	c := &model.Channel{
		Id: "ch" + model.NewId(),
	}
	err = s.SetCurrentChannel(c)
	require.NoError(t, err)
	channel, err = s.CurrentChannel()
	require.Nil(t, err)
	require.Equal(t, c, channel)
	require.Condition(t, func() bool {
		return channel != c
	})
}

func TestReactions(t *testing.T) {
	t.Run("SetReactions", func(t *testing.T) {
		s := newStore(t)
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

	t.Run("SetReaction", func(t *testing.T) {
		s := newStore(t)
		postId := model.NewId()
		userId := model.NewId()
		emojiName := "testemoji"
		reaction := &model.Reaction{
			UserId:    userId,
			PostId:    postId,
			EmojiName: emojiName,
		}
		err := s.SetReaction(reaction)
		require.NoError(t, err)
		reactions, err := s.Reactions(postId)
		require.NoError(t, err)
		require.Equal(t, []model.Reaction{*reaction}, reactions)
	})

	t.Run("DeleteReaction", func(t *testing.T) {
		s := newStore(t)
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
}

func TestId(t *testing.T) {
	s := newStore(t)

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
	s := newStore(t)

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
		err = s.SetChannelMembers(expected)
		require.NoError(t, err)
		members, err := s.ChannelMembers(channelId)
		require.NoError(t, err)
		require.Equal(t, expected, members)
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
		s := newStore(t)
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
		err = s.SetChannelMembers(channelMembers)
		require.NoError(t, err)
		require.Equal(t, 2, len(s.channelMembers[channel.Id]))
		err = s.RemoveChannelMember(channel.Id, channelMember1.UserId)
		require.NoError(t, err)
		members, err := s.ChannelMembers(channel.Id)
		require.NoError(t, err)
		require.Equal(t, 1, len(s.channelMembers[channel.Id]))
		require.Equal(t, channelMember2, members[0])
	})
}

func TestCurrentTeam(t *testing.T) {
	s := newStore(t)
	team, err := s.CurrentTeam()
	require.Nil(t, team)
	require.NoError(t, err)
	err = s.SetCurrentTeam(nil)
	require.Error(t, err)
	tm := &model.Team{
		Id: "tm" + model.NewId(),
	}
	err = s.SetCurrentTeam(tm)
	require.NoError(t, err)
	team, err = s.CurrentTeam()
	require.Nil(t, err)
	require.Equal(t, tm, team)
	require.Condition(t, func() bool {
		return team != tm
	})
}

func TestTeamMembers(t *testing.T) {
	s := newStore(t)

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
	s := newStore(t)

	t.Run("SetConfig", func(t *testing.T) {
		config := &model.Config{}
		s.SetConfig(config)
		require.Equal(t, s.Config(), *config)
	})
}

func TestStoreDeadlock(t *testing.T) {
	s := newStore(t)

	myId := model.NewId()
	id := model.NewId()
	id2 := model.NewId()

	err := s.SetUser(&model.User{
		Id: myId,
	})
	require.NoError(t, err)

	err = s.SetUsers([]*model.User{
		{Id: id},
		{Id: id2},
	})
	require.NoError(t, err)

	doneChan := make(chan struct{})

	go func() {
		users, err := s.RandomUsers(2)
		require.NoError(t, err)
		require.Len(t, users, 2)
		doneChan <- struct{}{}
	}()

	go func() {
		err = s.SetCurrentTeam(nil)
		require.Error(t, err)
		doneChan <- struct{}{}
	}()

	doneCount := 0
outer:
	for {
		select {
		case <-doneChan:
			doneCount++
			if doneCount == 2 {
				break outer
			}
		case <-time.After(2 * time.Second):
			require.Fail(t, "deadlock occurred")
			break outer
		}
	}
	require.Equal(t, 2, doneCount)
}

func TestChannelStats(t *testing.T) {
	s := newStore(t)

	channelId := model.NewId()
	stats := model.ChannelStats{
		ChannelId: channelId,
	}

	t.Run("Set", func(t *testing.T) {
		err := s.SetChannelStats(channelId, &stats)
		require.NoError(t, err)
	})

	t.Run("Get", func(t *testing.T) {
		storedStats, err := s.ChannelStats(channelId)
		require.NoError(t, err)
		require.Equal(t, &stats, storedStats)
	})

	t.Run("Clear", func(t *testing.T) {
		s.Clear()

		storedStats, err := s.ChannelStats(channelId)
		require.NoError(t, err)
		require.Nil(t, storedStats)

		err = s.SetCurrentChannel(&model.Channel{Id: channelId})
		require.NoError(t, err)

		err = s.SetChannelStats(channelId, &stats)
		require.NoError(t, err)

		s.Clear()

		storedStats, err = s.ChannelStats(channelId)
		require.NoError(t, err)
		require.Equal(t, &stats, storedStats)
	})
}

func TestThreads(t *testing.T) {
	t.Run("SetThreads", func(t *testing.T) {
		s := newStore(t)
		id := model.NewId()
		thread := &model.ThreadResponse{
			PostId: id,
		}
		err := s.SetThreads([]*model.ThreadResponse{thread})
		require.NoError(t, err)
		th, err := s.Thread(id)
		require.NoError(t, err)
		require.Equal(t, *thread, *th)
	})

	t.Run("MarkAllThreadsInTeamAsRead", func(t *testing.T) {
		s := newStore(t)
		now := model.GetMillis()
		teamId1 := model.NewId()
		teamId2 := model.NewId()
		channelId1 := model.NewId()
		channelId2 := model.NewId()
		channel1 := &model.Channel{Id: channelId1, TeamId: teamId1}
		channel2 := &model.Channel{Id: channelId2, TeamId: teamId2}
		err := s.SetChannels([]*model.Channel{channel1, channel2})
		require.NoError(t, err)

		threadId1 := model.NewId()
		threadId2 := model.NewId()
		threadId3 := model.NewId()

		thread1 := &model.ThreadResponse{PostId: threadId1, Post: &model.Post{ChannelId: channelId1}, UnreadReplies: 1, UnreadMentions: 1}
		thread2 := &model.ThreadResponse{PostId: threadId2, Post: &model.Post{ChannelId: channelId1}, UnreadReplies: 2, UnreadMentions: 2}
		thread3 := &model.ThreadResponse{PostId: threadId3, Post: &model.Post{ChannelId: channelId2}, UnreadReplies: 3, UnreadMentions: 3}

		err = s.SetThreads([]*model.ThreadResponse{thread1, thread2, thread3})
		require.NoError(t, err)

		err = s.MarkAllThreadsInTeamAsRead(teamId1)
		require.NoError(t, err)

		th1, err := s.Thread(threadId1)
		require.NoError(t, err)
		require.Equal(t, int64(0), th1.UnreadMentions)
		require.Equal(t, int64(0), th1.UnreadReplies)
		require.GreaterOrEqual(t, th1.LastViewedAt, now)

		th2, err := s.Thread(threadId2)
		require.NoError(t, err)
		require.Equal(t, int64(0), th2.UnreadMentions)
		require.Equal(t, int64(0), th2.UnreadReplies)
		require.GreaterOrEqual(t, th2.LastViewedAt, now)

		th3, err := s.Thread(threadId3)
		require.NoError(t, err)
		require.Equal(t, int64(3), th3.UnreadMentions)
		require.Equal(t, int64(3), th3.UnreadReplies)
		require.Equal(t, int64(0), th3.LastViewedAt)
	})

	t.Run("ThreadsSorted", func(t *testing.T) {
		s := newStore(t)
		threadId1 := model.NewId()
		threadId2 := model.NewId()
		threadId3 := model.NewId()
		epoch := model.GetMillis()
		thread1 := &model.ThreadResponse{
			PostId:         threadId1,
			UnreadReplies:  1,
			UnreadMentions: 1,
			LastReplyAt:    epoch,
		}
		thread2 := &model.ThreadResponse{
			PostId:         threadId2,
			UnreadReplies:  0,
			UnreadMentions: 0,
			LastReplyAt:    epoch + 1000,
		}
		thread3 := &model.ThreadResponse{
			PostId:         threadId3,
			UnreadReplies:  3,
			UnreadMentions: 3,
			LastReplyAt:    epoch + 2000,
		}
		err := s.SetThreads([]*model.ThreadResponse{thread1, thread2, thread3})
		require.NoError(t, err)

		threads, err := s.ThreadsSorted(true, false)
		require.NoError(t, err)
		require.Len(t, threads, 2)
		require.Equal(t, threads[0].PostId, threadId3)
		require.Equal(t, threads[1].PostId, threadId1)

		threads, err = s.ThreadsSorted(false, true)
		require.NoError(t, err)
		require.Len(t, threads, 3)
		require.Equal(t, threads[0].PostId, threadId1)
		require.Equal(t, threads[1].PostId, threadId2)
		require.Equal(t, threads[2].PostId, threadId3)

	})
}

func TestPostsWithAckRequests(t *testing.T) {
	s := newStore(t)
	ch := make(chan bool)
	n := 10
	ackPosts := make(map[string]bool, n)
	var mux sync.RWMutex

	for i := 0; i < n; i++ {
		// Concurrently create ack and regular posts
		go func() {
			p := &model.Post{
				Id:      model.NewId(),
				Message: "ack post",
				Metadata: &model.PostMetadata{
					Priority: &model.PostPriority{
						Priority:     model.NewString(model.PostPriorityUrgent),
						RequestedAck: model.NewBool(true),
					},
				},
			}
			mux.Lock()
			err := s.SetPost(p)
			require.NoError(t, err)
			ackPosts[p.Id] = true
			mux.Unlock()

			err = s.SetPost(&model.Post{
				Id:      model.NewId(),
				Message: "regular post",
			})
			require.NoError(t, err)
			ch <- true
		}()
	}

	for i := 0; i < n; i++ {
		<-ch

		// Concurrently read ack posts
		posts, err := s.PostsWithAckRequests()
		require.NoError(t, err)
		for _, p := range posts {
			mux.RLock()
			assert.Contains(t, ackPosts, p)
			mux.RUnlock()
		}
	}
}
