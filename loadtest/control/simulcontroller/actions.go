// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"math"
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

func (c *SimulController) connect() error {
	errChan, err := c.user.Connect()
	if err != nil {
		return fmt.Errorf("connect failed %w", err)
	}
	c.connected <- struct{}{}
	go func() {
		for err := range errChan {
			c.status <- c.newErrorStatus(err)
		}
	}()
	go c.wsEventHandler()
	go c.periodicActions()
	return nil
}

func (c *SimulController) disconnect() error {
	// one for ws loop and one for periodic actions loop
	select {
	case <-c.connected:
		c.disconnected <- struct{}{}
		c.disconnected <- struct{}{}
		<-c.waitwebsocket
	default:
		return errors.New("user is not connected")
	}
	err := c.user.Disconnect()
	if err != nil {
		return fmt.Errorf("disconnect failed %w", err)
	}
	return nil
}

func (c *SimulController) reload(full bool) control.UserActionResponse {
	if full {
		if err := c.disconnect(); err != nil {
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

func (c *SimulController) logout() control.UserActionResponse {
	err := c.disconnect()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	ok, err := c.user.Logout()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if !ok {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("user did not logout"))}
	}
	return control.UserActionResponse{Info: "logged out"}
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
