// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"math/rand"
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

func (c *SimulController) connect() {
	errChan := c.user.Connect()
	go func() {
		for err := range errChan {
			c.status <- c.newErrorStatus(err)
		}
	}()
}

func (c *SimulController) reload(full bool) control.UserActionResponse {
	if full {
		err := c.user.Disconnect()
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		c.connect()
	}

	return control.Reload(c.user)
}

func (c *SimulController) joinTeam(u user.User) control.UserActionResponse {
	userStore := u.Store()
	userId := userStore.Id()
	teamIds, err := u.GetAllTeams(0, 100)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	for _, teamId := range teamIds {
		tm, err := userStore.TeamMember(teamId, userId)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		if tm.UserId == "" {
			if err := u.AddTeamMember(teamId, userId); err != nil {
				return control.UserActionResponse{Err: control.NewUserError(err)}
			}
			c.status <- c.newInfoStatus(fmt.Sprintf("joined team %s", teamId))
			break
		}
	}
	return c.switchTeam(u)
}

func (c *SimulController) switchTeam(u user.User) control.UserActionResponse {
	team, err := u.Store().RandomTeam(store.SelectMemberOf | store.SelectNotCurrent)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if current, err := u.Store().CurrentChannel(); err == nil {
		// Somehow the webapp does a view to the current channel before switching.
		if _, err := u.ViewChannel(&model.ChannelView{ChannelId: current.Id}); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	} else if err != memstore.ErrChannelNotFound {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

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

	if err := u.SetCurrentTeam(&team); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	c.status <- c.newInfoStatus(fmt.Sprintf("switched to team %s", team.Id))

	// We should probably keep track of the last channel viewed in the team but
	// for now we can simplify and randomly pick one each time.

	return switchChannel(u)
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

	if err := u.SetCurrentChannel(&channel); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("switched to channel %s", channel.Id)}
}

func (c *SimulController) getUsersStatuses() control.UserActionResponse {
	err := c.user.GetUsersStatusesByIds([]string{c.user.Store().Id()})
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("got statuses")}
}

func createPost(u user.User) control.UserActionResponse {
	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// TODO: possibly add some additional idle time here to simulate the
	// user actually taking time to type a post message.
	u.SendTypingEvent(channel.Id, "")

	// This is an estimate that comes from stats on community servers.
	// The average length (in words) for a root post (not a reply).
	// TODO: should be part of some advanced configuration.
	avgWordCount := 34
	minWordCount := 1

	// TODO: make a util function out of this behaviour.
	wordCount := rand.Intn(avgWordCount*2-minWordCount*2) + minWordCount

	postId, err := u.CreatePost(&model.Post{
		Message:   control.GenerateRandomSentences(wordCount),
		ChannelId: channel.Id,
		CreateAt:  time.Now().Unix() * 1000,
	})
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("post created, id %v", postId)}
}

func createDirectChannel(u user.User) control.UserActionResponse {
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

	return control.UserActionResponse{Info: fmt.Sprintf("direct channel created, id %s", channelId)}
}

func createGroupChannel(u user.User) control.UserActionResponse {
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

	return control.UserActionResponse{Info: fmt.Sprintf("group channel created, id %s with users %+v", channelId, userIds)}
}
