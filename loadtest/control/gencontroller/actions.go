// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gencontroller

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"

	"github.com/mattermost/mattermost-server/server/v8/model"
)

type userAction struct {
	run        control.UserAction
	frequency  int
	idleTimeMs int
}

func logout(u user.User) control.UserActionResponse {
	err := u.Logout()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: "logged out"}
}

func (c *GenController) createTeam(u user.User) control.UserActionResponse {
	if !st.inc(StateTargetTeams, c.config.NumTeams) {
		return control.UserActionResponse{Info: "target number of teams reached"}
	}

	team := &model.Team{
		AllowOpenInvite: true,
		Type:            model.TeamOpen,
	}
	team.Name = "team-" + model.NewId()
	team.DisplayName = team.Name
	id, err := u.CreateTeam(team)
	if err != nil {
		st.dec(StateTargetTeams)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("created team %s", id)}
}

func (c *GenController) createPublicChannel(u user.User) control.UserActionResponse {
	if !st.inc(StateTargetChannels, c.config.NumChannels) {
		return control.UserActionResponse{Info: "target number of channels reached"}
	}

	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		st.dec(StateTargetChannels)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channel := &model.Channel{
		Name:   "ch-" + model.NewId(),
		TeamId: team.Id,
		Type:   model.ChannelTypeOpen,
	}
	channel.DisplayName = channel.Name
	channelId, err := u.CreateChannel(channel)

	if err != nil {
		st.dec(StateTargetChannels)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("public channel created, id %v", channelId)}
}

func (c *GenController) createPrivateChannel(u user.User) control.UserActionResponse {
	if !st.inc(StateTargetChannels, c.config.NumChannels) {
		return control.UserActionResponse{Info: "target number of channels reached"}
	}

	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		st.dec(StateTargetChannels)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channel := &model.Channel{
		Name:   "ch-" + model.NewId(),
		TeamId: team.Id,
		Type:   model.ChannelTypePrivate,
	}
	channel.DisplayName = channel.Name
	channelId, err := u.CreateChannel(channel)

	if err != nil {
		st.dec(StateTargetChannels)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("private channel created, id %v", channelId)}
}

func (c *GenController) createDirectChannel(u user.User) control.UserActionResponse {
	// Here we make a call to GetUsers to simulate the user opening the users
	// list when creating a direct channel.
	if _, err := u.GetUsers(0, 100); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// TODO: make the selection a bit smarter and pick someone
	// we don't have a direct channel with already.
	user, err := u.Store().RandomUser()
	if errors.Is(err, memstore.ErrLenMismatch) {
		return control.UserActionResponse{Info: "not enough users to create direct channel"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channelId, err := u.CreateDirectChannel(user.Id)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("direct channel created, id %s", channelId)}
}

func (c *GenController) createGroupChannel(u user.User) control.UserActionResponse {
	// Here we make a call to GetUsers to simulate the user opening the users
	// list when creating a direct channel.
	if _, err := u.GetUsers(0, 100); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	numUsers := 2 + rand.Intn(6)
	users, err := u.Store().RandomUsers(numUsers)
	if errors.Is(err, memstore.ErrLenMismatch) {
		return control.UserActionResponse{Info: "not enough users to create group channel"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// TODO: this transformation should be done at the store layer
	// by providing something like RandomUsersIds().
	userIds := make([]string, numUsers)
	for i := range users {
		userIds[i] = users[i].Id
	}

	channelId, err := u.CreateGroupChannel(userIds)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("group channel created, id %s", channelId)}
}

func (c *GenController) createPost(u user.User) control.UserActionResponse {
	if !st.inc(StateTargetPosts, c.config.NumPosts) {
		return control.UserActionResponse{Info: "target number of posts reached"}
	}

	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		st.dec(StateTargetPosts)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf)
	if errors.Is(err, memstore.ErrChannelStoreEmpty) {
		st.dec(StateTargetPosts)
		return control.UserActionResponse{Info: "no channels in store"}
	} else if err != nil {
		st.dec(StateTargetPosts)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// Select the post characteristics
	shouldLongThread := shouldMakeLongRunningThread(channel.Id)
	isUrgent := rand.Float64() < c.config.PercentUrgentPosts
	hasFilesAttached := rand.Float64() < 0.02

	channelMention := ""
	if shouldLongThread {
		channelMention = control.PickRandomString([]string{"@all ", "@here ", "@channel "})
	}

	avgWordCount := 34
	minWordCount := 1
	wordCount := rand.Intn(avgWordCount*2-minWordCount*2) + minWordCount

	post := &model.Post{
		Message:   control.GenerateRandomSentences(wordCount) + channelMention,
		ChannelId: channel.Id,
		CreateAt:  time.Now().Unix() * 1000,
	}

	if isUrgent {
		post.Metadata = &model.PostMetadata{}
		post.Metadata.Priority = &model.PostPriority{
			Priority:                model.NewString("urgent"),
			RequestedAck:            model.NewBool(false),
			PersistentNotifications: model.NewBool(false),
		}
	}

	if hasFilesAttached {
		if err := control.AttachFilesToPost(u, post); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	postId, err := u.CreatePost(post)
	if err != nil {
		st.dec(StateTargetPosts)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if shouldLongThread {
		st.setLongRunningThread(postId, channel.Id, channel.TeamId)
	}
	return control.UserActionResponse{Info: fmt.Sprintf("post created, id %v", postId)}
}

func (c *GenController) createPostReminder(u user.User) control.UserActionResponse {
	if !st.inc(StateTargetPostReminders, c.config.NumPostReminders) {
		return control.UserActionResponse{Info: "target number of post reminders reached"}
	}

	post, err := u.Store().RandomPost(store.SelectMemberOf)
	if err != nil {
		st.dec(StateTargetPostReminders)
		if errors.Is(err, memstore.ErrPostNotFound) {
			return control.UserActionResponse{Info: "no posts to set a reminder for"}
		}
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// Going with a hardcoded 10 minute addition for now.
	// Probably there's no need to randomize this yet.
	err = u.CreatePostReminder(u.Store().Id(), post.Id, time.Now().Add(10*time.Minute).Unix())
	if err != nil {
		st.dec(StateTargetPostReminders)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("created post reminder, id %s", post.Id)}
}

func (c *GenController) createReply(u user.User) control.UserActionResponse {
	if !st.inc(StateTargetPosts, c.config.NumPosts) {
		return control.UserActionResponse{Info: "target number of posts reached"}
	}

	var rootId string
	var channelId string
	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf)
	if errors.Is(err, memstore.ErrChannelStoreEmpty) {
		return control.UserActionResponse{Info: "no channels in store"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if rand.Float64() < c.config.PercentRepliesInLongThreads {
		threadInfos := st.getLongRunningThreadsInChannel(channel.Id)
		if len(threadInfos) > 0 {
			rootId = threadInfos[0].Id
			channelId = threadInfos[0].ChannelId
		}
	}
	if rootId == "" {
		root, err := u.Store().RandomPost(store.SelectMemberOf)
		if err != nil {
			st.dec(StateTargetPosts)
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		channelId = root.ChannelId
		if root.RootId != "" {
			rootId = root.RootId
		} else {
			rootId = root.Id
		}
	}

	avgWordCount := 34
	minWordCount := 1
	wordCount := rand.Intn(avgWordCount*2-minWordCount*2) + minWordCount

	postId, err := u.CreatePost(&model.Post{
		Message:   control.GenerateRandomSentences(wordCount),
		ChannelId: channelId,
		CreateAt:  time.Now().Unix() * 1000,
		RootId:    rootId,
	})
	if err != nil {
		st.dec(StateTargetPosts)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("reply created, id %v", postId)}
}

func (c *GenController) addReaction(u user.User) control.UserActionResponse {
	if !st.inc(StateTargetReactions, c.config.NumReactions) {
		return control.UserActionResponse{Info: "target number of reactions reached"}
	}

	postsIds, err := u.Store().PostsIdsSince(time.Now().Add(-10*time.Second).Unix() * 1000)
	if err != nil {
		st.dec(StateTargetReactions)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if len(postsIds) == 0 {
		st.dec(StateTargetReactions)
		return control.UserActionResponse{Info: "no posts to add reaction to"}
	}

	postId := postsIds[rand.Intn(len(postsIds))]
	reaction := &model.Reaction{
		UserId:    u.Store().Id(),
		PostId:    postId,
		EmojiName: []string{"+1", "tada", "point_up", "raised_hands"}[rand.Intn(4)],
	}

	reactions, err := u.Store().Reactions(postId)
	if err != nil {
		st.dec(StateTargetReactions)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	for i := 0; i < len(reactions); i++ {
		if reaction.UserId == reactions[i].UserId &&
			reaction.EmojiName == reactions[i].EmojiName {
			st.dec(StateTargetReactions)
			return control.UserActionResponse{Info: "reaction already added"}
		}
	}

	err = u.SaveReaction(reaction)
	if err != nil {
		st.dec(StateTargetReactions)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("added reaction to post %s", postId)}
}

func (c *GenController) joinChannel(u user.User) control.UserActionResponse {
	collapsedThreads := false

	resp := control.JoinChannel(u)
	if resp.Err != nil {
		return resp
	}

	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf)
	if errors.Is(err, memstore.ErrChannelStoreEmpty) {
		return control.UserActionResponse{Info: "no channels in store"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := c.user.GetPostsForChannel(channel.Id, 0, 60, collapsedThreads); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return resp
}

func (c *GenController) joinTeam(u user.User) control.UserActionResponse {
	userStore := u.Store()
	userId := userStore.Id()
	if _, err := u.GetAllTeams(0, 100); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	team, err := u.Store().RandomTeam(store.SelectNotMemberOf)
	if errors.Is(err, memstore.ErrTeamStoreEmpty) {
		return control.UserActionResponse{Info: "no team to join"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.AddTeamMember(team.Id, userId); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if err := u.GetChannelsForTeam(team.Id, true); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if err := u.GetChannelMembersForUser(userId, team.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("joined team %s", team.Id)}
}

func (c *GenController) createSidebarCategory(u user.User) control.UserActionResponse {
	if !st.inc(StateTargetSidebarCategories, c.config.NumSidebarCategories) {
		return control.UserActionResponse{Info: "target number of sidebar categories reached"}
	}

	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		st.dec(StateTargetSidebarCategories)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	category := &model.SidebarCategoryWithChannels{
		SidebarCategory: model.SidebarCategory{
			UserId:      u.Store().Id(),
			TeamId:      team.Id,
			DisplayName: control.PickRandomWord(),
		},
	}

	sidebarCategory, err := u.CreateSidebarCategory(u.Store().Id(), team.Id, category)
	if err != nil {
		st.dec(StateTargetSidebarCategories)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("created sidebar category, id %s", sidebarCategory.Id)}
}

func (c *GenController) followThread(u user.User) control.UserActionResponse {
	if !st.inc(StateTargetFollowedThreads, c.config.NumFollowedThreads) {
		return control.UserActionResponse{Info: "target number of followed threads reached"}
	}
	collapsedThreads, resp := control.CollapsedThreadsEnabled(u)
	if resp.Err != nil || !collapsedThreads {
		return resp
	}

	// Select a random post from any public or private channel the user is a member of (avoid picking DMs or GMs)
	post, err := u.Store().RandomPost(store.SelectMemberOf | store.SelectNotDirect | store.SelectNotGroup)
	if err != nil {
		st.dec(StateTargetFollowedThreads)
		if errors.Is(err, memstore.ErrPostNotFound) {
			return control.UserActionResponse{Info: "no threads to follow"}
		}
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	threadId := post.RootId
	if threadId == "" {
		threadId = post.Id
	}
	channel, err := u.Store().Channel(post.ChannelId)
	if err != nil {
		st.dec(StateTargetFollowedThreads)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	userId := u.Store().Id()
	if st.isThreadFollowedByUser(threadId, userId) {
		st.dec(StateTargetFollowedThreads)
		return control.UserActionResponse{Info: fmt.Sprintf("thread %s was already followed", threadId)}
	}

	err = u.UpdateThreadFollow(channel.TeamId, threadId, true)
	if err != nil {
		st.dec(StateTargetFollowedThreads)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	st.setThreadFollowedByUser(threadId, userId)

	return control.UserActionResponse{Info: fmt.Sprintf("followed thread %s", threadId)}
}
