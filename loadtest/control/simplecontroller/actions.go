// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simplecontroller

import (
	"errors"
	"fmt"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-server/v5/model"
)

type UserAction struct {
	run       func() control.UserStatus
	waitAfter time.Duration
}

func (c *SimpleController) signUp() control.UserStatus {
	if c.user.Store().Id() != "" {
		return c.newInfoStatus("user already signed up")
	}

	email := fmt.Sprintf("testuser%d@example.com", c.user.Id())
	username := fmt.Sprintf("testuser%d", c.user.Id())
	password := "testPass123$"

	err := c.user.SignUp(email, username, password)
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus("signed up")
}

func (c *SimpleController) login() control.UserStatus {
	// return here if already logged in
	err := c.user.Login()
	if err != nil {
		return c.newErrorStatus(err)
	}

	err = c.user.Connect()
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus("logged in")
}

func (c *SimpleController) logout() control.UserStatus {
	// return here if already logged out

	err := c.user.Disconnect()
	if err != nil {
		return c.newErrorStatus(err)
	}

	ok, err := c.user.Logout()
	if err != nil {
		return c.newErrorStatus(err)
	}

	if !ok {
		return c.newErrorStatus(errors.New("user did not logout"))
	}

	return c.newInfoStatus("logged out")
}

func (c *SimpleController) createPost() control.UserStatus {
	postId, err := c.user.CreatePost(&model.Post{
		Message: "Lorem ipsum dolor sit amet, consectetur adipiscing elit",
	})
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus(fmt.Sprintf("post created, id %v", postId))
}

func (c *SimpleController) createGroupChannel() control.UserStatus {
	channelId, err := c.user.CreateGroupChannel([]string{}) // TODO: populate memberIds parameter with other users
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus(fmt.Sprintf("group channel created, id %v", channelId))
}

func (c *SimpleController) viewChannel() control.UserStatus {
	return c.newErrorStatus(errors.New("not implemented"))
	/*
		channel, err := c.user.Store().Channel("") // TODO: fetch channel randomly?
		if err != nil {
			return control.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
		}

		channelViewResponse, err := c.user.ViewChannel(&model.ChannelView{
			ChannelId: channel.Id,
			PrevChannelId: "",
		})
		if err != nil {
			return control.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
		}

		return control.UserStatus{User: c.user, Info: fmt.Sprintf("channel viewed. result: %v", channelViewResponse.ToJson())}
	*/
}

func (c *SimpleController) reload() control.UserStatus {
	teams, err := c.user.Store().Teams()
	if err != nil {
		return c.newErrorStatus(err)
	}

	// Getting preferences.
	err = c.user.GetPreferences()
	if err != nil {
		return c.newErrorStatus(err)
	}

	chanId := ""
	if len(teams) > 0 {
		// Assuming this is a browser and the user has selected the
		// first team as the default.
		t := teams[0]
		// TODO: This will be populated by GetChannelsForTeamForUser API
		// which is to be implemented. For now, we assume that the user
		// already belongs to some channels, and they are stored in the
		// store's memory.
		chans, err := c.user.Store().Channels(t.Id)
		if err != nil {
			return c.newErrorStatus(err)
		}
		for _, ch := range chans {
			// Assuming the first channel to be the currently
			// selected one.
			if chanId == "" {
				chanId = ch.Id
			}
			// Marking the channels as viewed
			_, err := c.user.ViewChannel(&model.ChannelView{
				ChannelId:     ch.Id,
				PrevChannelId: "",
			})
			if err != nil {
				return c.newErrorStatus(err)
			}
		}
	}

	// TODO: GetConfig
	// TODO: GetLicense

	// Getting the user.
	_, err = c.user.GetMe()
	if err != nil {
		return c.newErrorStatus(err)
	}

	// TODO: GetTeamsForUser
	// TODO: GetTeamMembersForUser
	// TODO: GetRolesByNames
	// TODO: GetWebappPlugins
	// TODO: GetAllTeams
	// TODO: GetChannelsForTeamForUser
	// TODO: GetChannelMembersForUser

	// Getting unread teams.
	_, err = c.user.GetTeamsUnread("")
	if err != nil {
		return c.newErrorStatus(err)
	}

	// Get users by Ids.
	ids, err := c.user.GetUsersByIds([]string{c.user.Store().Id()})
	if err != nil {
		return c.newErrorStatus(err)
	}

	// Get user statuses by Ids.
	if len(ids) > 0 {
		err = c.user.GetUsersStatusesByIds(ids)
		if err != nil {
			return c.newErrorStatus(err)
		}
	}

	// TODO: GetUserStatus

	if chanId != "" {
		// Getting the channel stats.
		err = c.user.GetChannelStats(chanId)
		if err != nil {
			return c.newErrorStatus(err)
		}

		// Getting channel unread.
		_, err = c.user.GetChannelUnread(chanId)
		if err != nil {
			return c.newErrorStatus(err)
		}
	}

	return c.newInfoStatus("page reloaded")
}
