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
	if !st.inc("teams", c.config.NumTeams) {
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
		st.dec("teams")
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("created team %s", id)}
}

func (c *GenController) createPublicChannel(u user.User) control.UserActionResponse {
	if !st.inc("channels", c.config.NumChannels) {
		return control.UserActionResponse{Info: "target number of channels reached"}
	}

	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		st.dec("channels")
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
		st.dec("channels")
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	st.storeChannelID(channelId)

	return control.UserActionResponse{Info: fmt.Sprintf("public channel created, id %v", channelId)}
}

func (c *GenController) createPrivateChannel(u user.User) control.UserActionResponse {
	if !st.inc("channels", c.config.NumChannels) {
		return control.UserActionResponse{Info: "target number of channels reached"}
	}

	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		st.dec("channels")
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
		st.dec("channels")
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	st.storeChannelID(channelId)

	return control.UserActionResponse{Info: fmt.Sprintf("private channel created, id %v", channelId)}
}

func (c *GenController) createDirectChannel(u user.User) control.UserActionResponse {
	if !st.inc("channels", c.config.NumChannels) {
		return control.UserActionResponse{Info: "target number of channels reached"}
	}

	// Here we make a call to GetUsers to simulate the user opening the users
	// list when creating a direct channel.
	if _, err := u.GetUsers(0, 100); err != nil {
		st.dec("channels")
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// TODO: make the selection a bit smarter and pick someone
	// we don't have a direct channel with already.
	user, err := u.Store().RandomUser()
	if errors.Is(err, memstore.ErrLenMismatch) {
		st.dec("channels")
		return control.UserActionResponse{Info: "not enough users to create direct channel"}
	} else if err != nil {
		st.dec("channels")
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channelId, err := u.CreateDirectChannel(user.Id)
	if err != nil {
		st.dec("channels")
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("direct channel created, id %s", channelId)}
}

func (c *GenController) createGroupChannel(u user.User) control.UserActionResponse {
	if !st.inc("channels", c.config.NumChannels) {
		return control.UserActionResponse{Info: "target number of channels reached"}
	}

	// Here we make a call to GetUsers to simulate the user opening the users
	// list when creating a direct channel.
	if _, err := u.GetUsers(0, 100); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	numUsers := 2 + rand.Intn(6)
	users, err := u.Store().RandomUsers(numUsers)
	if errors.Is(err, memstore.ErrLenMismatch) {
		st.dec("channels")
		return control.UserActionResponse{Info: "not enough users to create group channel"}
	} else if err != nil {
		st.dec("channels")
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
		st.dec("channels")
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("group channel created, id %s", channelId)}
}

func (c *GenController) createPost(u user.User) control.UserActionResponse {
	if !st.inc("posts", c.config.NumPosts) {
		return control.UserActionResponse{Info: "target number of posts reached"}
	}

	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		st.dec("posts")
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf)
	if errors.Is(err, memstore.ErrChannelStoreEmpty) {
		st.dec("posts")
		return control.UserActionResponse{Info: "no channels in store"}
	} else if err != nil {
		st.dec("posts")
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channelMention := ""
	shouldLongThread := shouldMakeLongRunningThread((channel.Id))
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

	if rand.Float64() < c.config.PercentUrgentPosts {
		post.Metadata = &model.PostMetadata{}
		post.Metadata.Priority = &model.PostPriority{
			Priority:                model.NewString("urgent"),
			RequestedAck:            model.NewBool(false),
			PersistentNotifications: model.NewBool(false),
		}
	}

	postId, err := u.CreatePost(post)
	if err != nil {
		st.dec("posts")
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if shouldLongThread {
		st.setLongRunningThread(postId, channel.Id, channel.TeamId)
	}
	return control.UserActionResponse{Info: fmt.Sprintf("post created, id %v", postId)}
}

func (c *GenController) createReply(u user.User) control.UserActionResponse {
	if !st.inc("posts", c.config.NumPosts) {
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
		root, err := u.Store().RandomPost()
		if err != nil {
			st.dec("posts")
			if err == memstore.ErrPostNotFound {
				return control.UserActionResponse{Info: "no posts in store"}
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
		st.dec("posts")
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("reply created, id %v", postId)}
}

func (c *GenController) addReaction(u user.User) control.UserActionResponse {
	if !st.inc("reactions", c.config.NumReactions) {
		return control.UserActionResponse{Info: "target number of reactions reached"}
	}

	postsIds, err := u.Store().PostsIdsSince(time.Now().Add(-10*time.Second).Unix() * 1000)
	if err != nil {
		st.dec("reactions")
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if len(postsIds) == 0 {
		st.dec("reactions")
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
		st.dec("reactions")
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	for i := 0; i < len(reactions); i++ {
		if reaction.UserId == reactions[i].UserId &&
			reaction.EmojiName == reactions[i].EmojiName {
			st.dec("reactions")
			return control.UserActionResponse{Info: "reaction already added"}
		}
	}

	err = u.SaveReaction(reaction)
	if err != nil {
		st.dec("reactions")
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
	channelID, err := chooseChannel(c.config.ChannelMembersDistribution[idx], u)
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
		err = u.AddChannelMember(channelID, u.Store().Id())
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		resp = control.UserActionResponse{Info: fmt.Sprintf("joined channel %s", channelID)}
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

var errMemberLimitExceeded = errors.New("member limit exceeded")

// chooseChannel will pick a channelID randomly from the range of indexes.
// If the chosen channelID has exceeded the number of channelmembers, it will
// select another one in the range until it has found one.
func chooseChannel(dist ChannelMemberDistribution, u user.User) (string, error) {
	minIndex := int(dist.MinIndexRange * float64(len(st.channels)))
	maxIndex := int(dist.MaxIndexRange * float64(len(st.channels)))

	if maxIndex-minIndex <= 1 {
		return "", errors.New("not enough channels to select from; either increase range or increase number of channels to create")
	}

	var channelID string
	maxTimes := maxIndex - minIndex
	cnt := 0
	for {
		if cnt == maxTimes {
			return "", errMemberLimitExceeded
		}
		target := rand.Intn(maxIndex-minIndex) + minIndex
		// target is guaranteed to be within bounds of st.channels
		channelID = st.channels[target]

		members, err := u.Store().ChannelMembers(channelID)
		if err != nil {
			return "", err
		}
		if len(members) > int(dist.MemberLimit) {
			cnt++
			continue
		}
		return channelID, nil
	}
}
