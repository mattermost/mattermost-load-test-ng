// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gencontroller

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"

	"github.com/mattermost/mattermost/server/public/model"
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

func (c *GenController) login(u user.User) (res control.UserActionResponse) {
	if !st.inc(StateTargetUsers, int64(c.numUsers)) {
		return control.UserActionResponse{Info: "target number of users reached"}
	}
	defer func() {
		if res.Err != nil || res.Warn != "" {
			st.dec(StateTargetUsers)
		}
	}()

	return control.Login(u)
}

func (c *GenController) createTeam(u user.User) (res control.UserActionResponse) {
	if !st.inc(StateTargetTeams, c.config.NumTeams) {
		return control.UserActionResponse{Info: "target number of teams reached"}
	}
	defer func() {
		if res.Err != nil || res.Warn != "" {
			st.dec(StateTargetTeams)
		}
	}()

	team := &model.Team{
		AllowOpenInvite: true,
		Type:            model.TeamOpen,
	}
	team.Name = "team-" + model.NewId()
	team.DisplayName = team.Name
	id, err := u.CreateTeam(team)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("created team %s", id)}
}

func (c *GenController) createCPAField(u user.User) (res control.UserActionResponse) {
	if !st.inc(StateTargetCPAFields, c.config.NumCPAFields) {
		return control.UserActionResponse{Info: "target number of custom profile fields reached"}
	}
	defer func() {
		if res.Err != nil || res.Warn != "" {
			st.dec(StateTargetCPAFields)
		}
	}()

	// Only sysadmin can create CPA fields
	isSysAdmin, err := u.IsSysAdmin()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if !isSysAdmin {
		return control.UserActionResponse{Warn: "not an admin user, unable to create a CPA field"}
	}

	cpaField := &model.PropertyField{
		Name: control.PickRandomWord() + "_" + control.PickRandomWord(),
		Type: model.PropertyFieldTypeText,
	}
	field, err := u.CreateCPAField(cpaField)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("created CPA field %s", field.ID)}
}

func (c *GenController) createCPAValues(u user.User) (res control.UserActionResponse) {
	fields := u.Store().GetCPAFields()
	if len(fields) == 0 {
		return control.UserActionResponse{Info: "no CPA Fields returned"}
	}
	values := make(map[string]json.RawMessage)

	for _, field := range fields {
		randomText := control.PickRandomWord()
		value, err := json.Marshal(randomText)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}

		values[field.ID] = value
	}

	err := u.PatchCPAValues(values)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("created CPA values for user %s", u.Store().Id())}
}

func (c *GenController) createPublicChannel(u user.User) (res control.UserActionResponse) {
	if !st.inc(StateTargetChannelsPublic, c.config.NumChannelsPublic) {
		return control.UserActionResponse{Info: "target number of public channels reached"}
	}
	defer func() {
		if res.Err != nil || res.Warn != "" {
			st.dec(StateTargetChannelsPublic)
		}
	}()

	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channel := &model.Channel{
		Name:   control.PickRandomWord() + "_" + control.PickRandomWord(),
		TeamId: team.Id,
		Type:   model.ChannelTypeOpen,
	}
	channel.DisplayName = channel.Name
	channelId, err := u.CreateChannel(channel)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	st.storeChannelID(channelId)

	return control.UserActionResponse{Info: fmt.Sprintf("public channel created, id %v", channelId)}
}

func (c *GenController) createPrivateChannel(u user.User) (res control.UserActionResponse) {
	if !st.inc(StateTargetChannelsPrivate, c.config.NumChannelsPrivate) {
		return control.UserActionResponse{Info: "target number of private channels reached"}
	}
	defer func() {
		if res.Err != nil || res.Warn != "" {
			st.dec(StateTargetChannelsPrivate)
		}
	}()

	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channel := &model.Channel{
		Name:   control.PickRandomWord() + "_" + control.PickRandomWord(),
		TeamId: team.Id,
		Type:   model.ChannelTypePrivate,
	}
	channel.DisplayName = channel.Name
	channelId, err := u.CreateChannel(channel)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	st.storeChannelID(channelId)

	return control.UserActionResponse{Info: fmt.Sprintf("private channel created, id %v", channelId)}
}

func (c *GenController) getUsers(u user.User) control.UserActionResponse {
	// Here we make a call to GetUsers to simulate the user opening the users
	// list when creating a direct/group channel.
	if _, err := u.GetUsers(0, c.numUsers); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: "loaded users"}
}

func (c *GenController) createDirectChannel(u user.User) (res control.UserActionResponse) {
	if !st.inc(StateTargetChannelsDM, c.config.NumChannelsDM) {
		return control.UserActionResponse{Info: "target number of DM channels reached"}
	}
	defer func() {
		if res.Err != nil || res.Warn != "" {
			st.dec(StateTargetChannelsDM)
		}
	}()

	userID := u.Store().Id()
	// Make at most twice as many attempts as there are valid users
	// (i.e., those with which we don't have a DM open yet)
	maxAttempts := 2 * (c.numUsers - st.numDMs(userID))
	for i := 0; i < maxAttempts; i++ {
		otherUser, err := u.Store().RandomUser()
		if errors.Is(err, memstore.ErrLenMismatch) {
			return control.UserActionResponse{Warn: "not enough users to create direct channel"}
		} else if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}

		// If it exists, pick another random user
		if st.dmExists(userID, otherUser.Id) {
			continue
		}

		// If it doesn't, create it
		channelId, err := u.CreateDirectChannel(otherUser.Id)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		st.setDM(userID, otherUser.Id)

		return control.UserActionResponse{Info: fmt.Sprintf("direct channel created between %q and %q, with id %q", userID, otherUser.Id, channelId)}
	}

	return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("maximum attempts (%d) reached when randomly picking a user to create a DM with user: %s", maxAttempts, userID))}
}

func (c *GenController) createGroupChannel(u user.User) (res control.UserActionResponse) {
	if !st.inc(StateTargetChannelsGM, c.config.NumChannelsGM) {
		return control.UserActionResponse{Info: "target number of GM channels reached"}
	}
	defer func() {
		if res.Err != nil || res.Warn != "" {
			st.dec(StateTargetChannelsGM)
		}
	}()

	numUsers := 2 + rand.Intn(6)
	users, err := u.Store().RandomUsers(numUsers)
	if errors.Is(err, memstore.ErrLenMismatch) {
		return control.UserActionResponse{Warn: "not enough users to create group channel"}
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

func (c *GenController) createPost(u user.User) (res control.UserActionResponse) {
	if !st.inc(StateTargetPosts, c.config.NumPosts) {
		return control.UserActionResponse{Info: "target number of posts reached"}
	}
	defer func() {
		if res.Err != nil || res.Warn != "" {
			st.dec(StateTargetPosts)
		}
	}()

	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf)
	if errors.Is(err, memstore.ErrChannelStoreEmpty) {
		return control.UserActionResponse{Warn: "no channels in store"}
	} else if err != nil {
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
			Priority:                model.NewPointer("urgent"),
			RequestedAck:            model.NewPointer(false),
			PersistentNotifications: model.NewPointer(false),
		}
	}

	if hasFilesAttached {
		if err := control.AttachFilesToPost(u, post); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	postId, err := u.CreatePost(post)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if shouldLongThread {
		st.setLongRunningThread(postId, channel.Id, channel.TeamId)
	}
	return control.UserActionResponse{Info: fmt.Sprintf("post created, id %v", postId)}
}

func (c *GenController) createPostReminder(u user.User) (res control.UserActionResponse) {
	if !st.inc(StateTargetPostReminders, c.config.NumPostReminders) {
		return control.UserActionResponse{Info: "target number of post reminders reached"}
	}
	defer func() {
		if res.Err != nil || res.Warn != "" {
			st.dec(StateTargetPostReminders)
		}
	}()

	post, err := u.Store().RandomPost(store.SelectMemberOf)
	if err != nil {
		if errors.Is(err, memstore.ErrPostNotFound) {
			return control.UserActionResponse{Warn: "no posts to set a reminder for"}
		}
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// Going with a hardcoded 10 minute addition for now.
	// Probably there's no need to randomize this yet.
	err = u.CreatePostReminder(u.Store().Id(), post.Id, time.Now().Add(10*time.Minute).Unix())
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("created post reminder, id %s", post.Id)}
}

func (c *GenController) createReply(u user.User) (res control.UserActionResponse) {
	if !st.inc(StateTargetPosts, c.config.NumPosts) {
		return control.UserActionResponse{Info: "target number of posts reached"}
	}
	defer func() {
		if res.Err != nil || res.Warn != "" {
			st.dec(StateTargetPosts)
		}
	}()

	var rootId string
	var channelId string
	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf)
	if errors.Is(err, memstore.ErrChannelStoreEmpty) {
		return control.UserActionResponse{Warn: "no channels in store"}
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
			if errors.Is(err, memstore.ErrPostNotFound) {
				return control.UserActionResponse{Warn: "no posts in store"}
			}
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
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("reply created, id %v", postId)}
}

func (c *GenController) addReaction(u user.User) (res control.UserActionResponse) {
	if !st.inc(StateTargetReactions, c.config.NumReactions) {
		return control.UserActionResponse{Info: "target number of reactions reached"}
	}
	defer func() {
		if res.Err != nil || res.Warn != "" {
			st.dec(StateTargetReactions)
		}
	}()

	postsIds, err := u.Store().PostsIdsSince(time.Now().Add(-10*time.Second).Unix() * 1000)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if len(postsIds) == 0 {
		return control.UserActionResponse{Warn: "no posts to add reaction to"}
	}

	postId := postsIds[rand.Intn(len(postsIds))]
	reaction := &model.Reaction{
		UserId:    u.Store().Id(),
		PostId:    postId,
		EmojiName: []string{"+1", "tada", "point_up", "raised_hands"}[rand.Intn(4)],
	}

	reactions, err := u.Store().Reactions(postId)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	for i := 0; i < len(reactions); i++ {
		if reaction.UserId == reactions[i].UserId &&
			reaction.EmojiName == reactions[i].EmojiName {
			return control.UserActionResponse{Warn: "reaction already added"}
		}
	}

	reactionLimit, err := strconv.ParseInt(u.Store().ClientConfig()["UniqueEmojiReactionLimitPerPost"], 10, 64)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if reactionLimit != 0 {
		uniqueEmojiNames := map[string]bool{reaction.EmojiName: true}
		for _, r := range reactions {
			uniqueEmojiNames[r.EmojiName] = true
		}

		if len(uniqueEmojiNames) >= int(reactionLimit) {
			return control.UserActionResponse{Info: "reaction limit reached"}
		}
	}

	err = u.SaveReaction(reaction)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("added reaction to post %s", postId)}
}

func (c *GenController) joinChannel(u user.User) control.UserActionResponse {
	collapsedThreads := false

	// We get the channel range depending on the weighted probability.
	idx, err := control.SelectWeighted(c.channelSelectionWeights)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// We choose a channel from that range.
	channelID, err := chooseChannel(c.config.ChannelMembersDistribution, idx, u)
	if err != nil {
		if err == errMemberLimitExceeded {
			return control.UserActionResponse{Info: "channel range already filled"}
		}
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	cm, err := u.Store().ChannelMember(channelID, u.Store().Id())
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	resp := control.UserActionResponse{Info: "no channel to join"}
	if cm.UserId == "" {
		// We use sysadmin to add channel in case it's a private channel.
		// Otherwise normal users don't have permissions to join a private channel.
		err = c.sysadmin.AddChannelMember(channelID, u.Store().Id())
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		resp = control.UserActionResponse{Info: fmt.Sprintf("joined channel %s", channelID)}

		if err := c.user.GetPostsForChannel(channelID, 0, 60, collapsedThreads); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	return resp
}

func (c *GenController) joinAllTeams(u user.User) control.UserActionResponse {
	userStore := u.Store()
	userId := userStore.Id()
	if _, err := u.GetAllTeams(0, 100); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	teams, err := u.Store().Teams()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	joinedTeamIds := []string{}
	for _, team := range teams {
		// If user is already added to the team we skip the AddTeamMember call but
		// otherwise proceed with fetching the rest of the required entities
		// (e.g. channels, channel members) to guarantee they get loaded in case
		// of retry.
		if !u.Store().IsTeamMember(team.Id, userId) {
			if err := u.AddTeamMember(team.Id, userId); err != nil {
				return control.UserActionResponse{Err: control.NewUserError(err)}
			}
		}

		if err := u.GetChannelsForTeam(team.Id, true); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		if err := u.GetChannelMembersForUser(userId, team.Id); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}

		joinedTeamIds = append(joinedTeamIds, team.Id)
	}

	return control.UserActionResponse{Info: fmt.Sprintf("joined %d teams [%s]", len(joinedTeamIds), strings.Join(joinedTeamIds, ","))}
}

func (c *GenController) createSidebarCategory(u user.User) (res control.UserActionResponse) {
	if !st.inc(StateTargetSidebarCategories, c.config.NumSidebarCategories) {
		return control.UserActionResponse{Info: "target number of sidebar categories reached"}
	}
	defer func() {
		if res.Err != nil || res.Warn != "" {
			st.dec(StateTargetSidebarCategories)
		}
	}()

	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
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
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("created sidebar category, id %s", sidebarCategory.Id)}
}

func (c *GenController) followThread(u user.User) (res control.UserActionResponse) {
	if !st.inc(StateTargetFollowedThreads, c.config.NumFollowedThreads) {
		return control.UserActionResponse{Info: "target number of followed threads reached"}
	}
	defer func() {
		if res.Err != nil || res.Warn != "" {
			st.dec(StateTargetFollowedThreads)
		}
	}()

	collapsedThreads, resp := control.CollapsedThreadsEnabled(u)
	if resp.Err != nil || !collapsedThreads {
		return resp
	}

	// Select a random post from any public or private channel the user is a member of (avoid picking DMs or GMs)
	post, err := u.Store().RandomPost(store.SelectMemberOf | store.SelectNotDirect | store.SelectNotGroup)
	if err != nil {
		if errors.Is(err, memstore.ErrPostNotFound) {
			return control.UserActionResponse{Warn: "no threads to follow"}
		}
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	threadId := post.RootId
	if threadId == "" {
		threadId = post.Id
	}
	channel, err := u.Store().Channel(post.ChannelId)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	userId := u.Store().Id()
	if st.isThreadFollowedByUser(threadId, userId) {
		return control.UserActionResponse{Warn: fmt.Sprintf("thread %s was already followed", threadId)}
	}

	err = u.UpdateThreadFollow(channel.TeamId, threadId, true)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	st.setThreadFollowedByUser(threadId, userId)

	return control.UserActionResponse{Info: fmt.Sprintf("followed thread %s", threadId)}
}

func (c *GenController) getPosts(u user.User) (res control.UserActionResponse) {
	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf)
	if errors.Is(err, memstore.ErrChannelStoreEmpty) {
		return control.UserActionResponse{Warn: "no channels in store"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	collapsedThreads := false
	if err := c.user.GetPostsForChannel(channel.Id, 0, 200, collapsedThreads); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("got posts for channel %q", channel.Id)}
}
