// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"

	"github.com/mattermost/mattermost-server/v5/model"
)

type userAction struct {
	run       control.UserAction
	frequency int
}

func (c *SimulController) connect() error {
	errChan, err := c.user.Connect()
	if err != nil {
		return fmt.Errorf("connect failed %w", err)
	}
	go func() {
		for err := range errChan {
			c.status <- c.newErrorStatus(err)
		}
	}()

	return nil
}

func (c *SimulController) reload(full bool) control.UserActionResponse {
	if full {
		if err := c.user.Disconnect(); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		c.user.ClearUserData()
		if err := c.connect(); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	resp := control.Reload(c.user)
	if resp.Err != nil {
		return resp
	}

	c.status <- c.newInfoStatus(resp.Info)

	team, err := c.user.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		// If the current team is not set we switch to a random one.
		return c.switchTeam(c.user)
	}

	return loadTeam(c.user, team)
}

func (c *SimulController) login(u user.User) control.UserActionResponse {
	for {
		resp := control.Login(u)
		if resp.Err == nil {
			err := c.connect()
			if err == nil {
				return resp
			}
			c.status <- c.newErrorStatus(err)
		}

		c.status <- c.newErrorStatus(resp.Err)

		idleTimeMs := time.Duration(math.Round(1000 * c.rate))

		select {
		case <-c.stop:
			return control.UserActionResponse{Info: "login canceled"}
		case <-time.After(idleTimeMs * time.Millisecond):
		}
	}
}

func (c *SimulController) joinTeam(u user.User) control.UserActionResponse {
	userStore := u.Store()
	userId := userStore.Id()

	if _, err := u.GetAllTeams(0, 100); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	team, err := u.Store().RandomTeam(store.SelectNotMemberOf)
	if errors.Is(err, memstore.ErrTeamStoreEmpty) {
		c.status <- c.newInfoStatus("no team to join")
		return c.switchTeam(u)
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.AddTeamMember(team.Id, userId); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	c.status <- c.newInfoStatus(fmt.Sprintf("joined team %s", team.Id))

	return c.switchTeam(u)
}

func loadTeam(u user.User, team *model.Team) control.UserActionResponse {
	if _, err := u.GetChannelsForTeamForUser(team.Id, u.Store().Id(), true); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.GetChannelMembersForUser(u.Store().Id(), team.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if _, err := u.GetTeamsUnread(""); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// TODO: use more realistic data.
	var userIds []string
	userIds = append(userIds, u.Store().Id())
	if err := u.GetUsersStatusesByIds(userIds); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("loaded team %s", team.Id)}
}

func (c *SimulController) switchTeam(u user.User) control.UserActionResponse {
	team, err := u.Store().RandomTeam(store.SelectMemberOf | store.SelectNotCurrent)
	if errors.Is(err, memstore.ErrTeamStoreEmpty) {
		return control.UserActionResponse{Info: "no other team to switch to"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.SetCurrentTeam(&team); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	c.status <- c.newInfoStatus(fmt.Sprintf("switched to team %s", team.Id))

	if resp := loadTeam(u, &team); resp.Err != nil {
		return resp
	}

	// We should probably keep track of the last channel viewed in the team but
	// for now we can simplify and randomly pick one each time.

	return switchChannel(u)
}

func (c *SimulController) joinChannel(u user.User) control.UserActionResponse {
	team, err := u.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("current team should be set"))}
	}

	if err := u.GetPublicChannelsForTeam(team.Id, 0, 100); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channel, err := u.Store().RandomChannel(team.Id, store.SelectNotMemberOf)
	if errors.Is(err, memstore.ErrChannelStoreEmpty) {
		return control.UserActionResponse{Info: "no channel to join"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.AddChannelMember(channel.Id, u.Store().Id()); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("joined channel %s", channel.Id)}
}

func viewChannel(u user.User, channel *model.Channel) control.UserActionResponse {
	var currentChanId string
	if current, err := u.Store().CurrentChannel(); err == nil {
		currentChanId = current.Id
		// Somehow the webapp does a view to the current channel before switching.
		if _, err := u.ViewChannel(&model.ChannelView{ChannelId: current.Id}); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	} else if err != memstore.ErrChannelNotFound {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// TODO: use the information returned here to figure out how to properly fetch posts.
	if _, err := u.ViewChannel(&model.ChannelView{ChannelId: channel.Id, PrevChannelId: currentChanId}); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if view, err := u.Store().ChannelView(channel.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if view == 0 {
		if err := u.GetPostsAroundLastUnread(channel.Id, 30, 30); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	} else {
		if err := u.GetPostsSince(channel.Id, time.Now().Add(-1*time.Minute).Unix()*1000); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	if err := u.GetChannelStats(channel.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.SetCurrentChannel(channel); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("viewed channel %s", channel.Id)}
}

func switchChannel(u user.User) control.UserActionResponse {
	team, err := u.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("current team should be set"))}
	}

	channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf|store.SelectNotCurrent)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if resp := viewChannel(u, &channel); resp.Err != nil {
		return control.UserActionResponse{Err: control.NewUserError(resp.Err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("switched to channel %s", channel.Id)}
}

func (c *SimulController) getUsersStatuses() control.UserActionResponse {
	err := c.user.GetUsersStatusesByIds([]string{c.user.Store().Id()})
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: "got statuses"}
}

func editPost(u user.User) control.UserActionResponse {
	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	post, err := u.Store().RandomPostForChannelByUser(channel.Id, u.Store().Id())
	if errors.Is(err, memstore.ErrPostNotFound) {
		return control.UserActionResponse{Info: "no posts to edit"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	message := genMessage(post.RootId != "")
	postId, err := u.PatchPost(post.Id, &model.PostPatch{
		Message: &message,
	})
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("post edited, id %v", postId)}
}

func createPostReply(u user.User) control.UserActionResponse {
	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	post, err := u.Store().RandomPostForChannel(channel.Id)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	var rootId string
	if post.RootId != "" {
		rootId = post.RootId
	} else {
		rootId = post.Id
	}

	// TODO: possibly add some additional idle time here to simulate the
	// user actually taking time to type a post message.
	if err := u.SendTypingEvent(channel.Id, ""); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	postId, err := u.CreatePost(&model.Post{
		Message:   genMessage(true),
		ChannelId: channel.Id,
		CreateAt:  time.Now().Unix() * 1000,
		RootId:    rootId,
	})
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("post reply created, id %v", postId)}
}

func createPost(u user.User) control.UserActionResponse {
	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// TODO: possibly add some additional idle time here to simulate the
	// user actually taking time to type a post message.
	if err := u.SendTypingEvent(channel.Id, ""); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	postId, err := u.CreatePost(&model.Post{
		Message:   genMessage(false),
		ChannelId: channel.Id,
		CreateAt:  time.Now().Unix() * 1000,
	})
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("post created, id %v", postId)}
}

func (c *SimulController) createDirectChannel(u user.User) control.UserActionResponse {
	// Here we make a call to GetUsers to simulate the user opening the users
	// list when creating a direct channel.
	if err := u.GetUsers(0, 100); err != nil {
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

	if err := u.GetChannel(channelId); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.GetChannelMember(channelId, u.Store().Id()); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// We need to update the user's preferences so that
	// on next reload we can properly fetch opened DMs.
	pref := &model.Preferences{
		model.Preference{
			UserId:   u.Store().Id(),
			Category: model.PREFERENCE_CATEGORY_DIRECT_CHANNEL_SHOW,
			Name:     channelId,
			Value:    "true",
		},
	}

	if err := u.UpdatePreferences(pref); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channel, err := u.Store().Channel(channelId)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if resp := viewChannel(u, channel); resp.Err != nil {
		return control.UserActionResponse{Err: control.NewUserError(resp.Err)}
	}

	c.status <- c.newInfoStatus(fmt.Sprintf("direct channel created, id %s", channelId))

	return createPost(u)
}

func (c *SimulController) createGroupChannel(u user.User) control.UserActionResponse {
	// Here we make a call to GetUsers to simulate the user opening the users
	// list when creating a group channel.
	if err := u.GetUsers(0, 100); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// TODO: consider making this number range between an interval.
	numUsers := 2
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

	// We need to update the user's preferences so that
	// on next reload we can properly fetch opened DMs.
	pref := &model.Preferences{
		model.Preference{
			UserId:   u.Store().Id(),
			Category: "group_channel_show", // It looks like there's no constant for this in the model.
			Name:     channelId,
			Value:    "true",
		},
	}

	if err := u.UpdatePreferences(pref); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channel, err := u.Store().Channel(channelId)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if resp := viewChannel(u, channel); resp.Err != nil {
		return control.UserActionResponse{Err: control.NewUserError(resp.Err)}
	}

	c.status <- c.newInfoStatus(fmt.Sprintf("group channel created, id %s with users %+v", channelId, userIds))

	return createPost(u)
}
