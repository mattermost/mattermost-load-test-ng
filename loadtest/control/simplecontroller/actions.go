// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simplecontroller

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-load-test/loadtest/user"
)

type UserAction struct {
	run       func(status chan<- user.UserStatus) bool
	waitAfter time.Duration
}

func (c *SimpleController) signUp(status chan<- user.UserStatus) bool {
	if c.user.Store().User() != nil {
		return true
	}

	email := fmt.Sprintf("testuser%d@example.com", c.user.Id())
	username := fmt.Sprintf("testuser%d", c.user.Id())
	password := "testpwd"

	err := c.user.SignUp(email, username, password)
	if err != nil {
		status <- user.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
		return false
	}

	status <- user.UserStatus{User: c.user, Info: "signed up"}
	return true
}

func (c *SimpleController) login(status chan<- user.UserStatus) bool {
	// return here if already logged in
	err := c.user.Login()
	if err != nil {
		status <- user.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
		return false
	}

	status <- user.UserStatus{User: c.user, Info: "logged in"}
	return true
}

func (c *SimpleController) logout(status chan<- user.UserStatus) bool {
	// return here if already logged out
	err := c.user.Logout()
	if err != nil {
		status <- user.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
		return false
	}

	status <- user.UserStatus{User: c.user, Info: "logged out"}
	return true
}
