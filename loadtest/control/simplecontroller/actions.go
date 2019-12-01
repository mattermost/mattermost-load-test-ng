// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simplecontroller

import (
	"errors"
	"fmt"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost-server/model"
)

type UserAction struct {
	run       func() user.UserStatus
	waitAfter time.Duration
}

func (c *SimpleController) signUp() user.UserStatus {
	if c.user.Store().Id() != "" {
		return user.UserStatus{User: c.user, Info: "user already signed up"}
	}

	email := fmt.Sprintf("testuser%d@example.com", c.user.Id())
	username := fmt.Sprintf("testuser%d", c.user.Id())
	password := "testpwd"

	err := c.user.SignUp(email, username, password)
	if err != nil {
		return user.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
	}

	return user.UserStatus{User: c.user, Info: "signed up"}
}

func (c *SimpleController) login() user.UserStatus {
	// return here if already logged in
	err := c.user.Login()
	if err != nil {
		return user.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
	}

	err = c.user.Connect()
	if err != nil {
		return user.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
	}

	return user.UserStatus{User: c.user, Info: "logged in"}
}

func (c *SimpleController) logout() user.UserStatus {
	// return here if already logged out

	err := c.user.Disconnect()
	if err != nil {
		return user.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
	}

	ok, err := c.user.Logout()
	if err != nil {
		return user.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
	}

	if !ok {
		return user.UserStatus{User: c.user, Err: errors.New("User did not logout"), Code: user.STATUS_ERROR}
	}

	return user.UserStatus{User: c.user, Info: "logged out"}
}

func (c *SimpleController) createPost() user.UserStatus {
	postId, err := c.user.CreatePost(&model.Post{
		Message: "Lorem ipsum dolor sit amet, consectetur adipiscing elit",
	})
	if err != nil {
		return user.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
	}

	return user.UserStatus{User: c.user, Info: fmt.Sprintf("post created, id %v", postId)}
}

func (c *SimpleController) createGroupChannel() user.UserStatus {
	channelId, err := c.user.CreateGroupChannel([]string{}) // TODO: populate memberIds parameter with other users
	if err != nil {
		return user.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
	}

	return user.UserStatus{User: c.user, Info: fmt.Sprintf("group channel created, id %v", channelId)}
}

func (c *SimpleController) viewChannel() user.UserStatus {
	return user.UserStatus{User: c.user, Err: errors.New("not implemented"), Code: user.STATUS_ERROR}
	/*
		channel, err := c.user.Store().Channel("") // TODO: fetch channel randomly?
		if err != nil {
			return user.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
		}

		channelViewResponse, err := c.user.ViewChannel(&model.ChannelView{
			ChannelId: channel.Id,
			PrevChannelId: "",
		})
		if err != nil {
			return user.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
		}

		return user.UserStatus{User: c.user, Info: fmt.Sprintf("channel viewed. result: %v", channelViewResponse.ToJson())}
	*/
}
