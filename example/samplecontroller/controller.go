// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package samplecontroller

import (
	"errors"
	"fmt"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

type SampleController struct {
	user user.User
	stop chan bool
}

type userAction struct {
	run       func() user.UserStatus
	waitAfter time.Duration
}

func (c *SampleController) Init(user user.User) {
	c.user = user
	c.stop = make(chan bool)
}

func (c *SampleController) Run(status chan<- user.UserStatus) {
	if c.user == nil {
		c.sendFailStatus(status, "controller was not initialized")
		return
	}

	actions := []userAction{
		{
			run:       c.signUp,
			waitAfter: 4000,
		},
		{
			run:       c.login,
			waitAfter: 4000,
		},
		{
			run:       c.logout,
			waitAfter: 4000,
		},
	}

	status <- user.UserStatus{User: c.user, Info: "user started", Code: user.STATUS_STARTED}

	defer c.sendStopStatus(status)

	for {
		for i := 0; i < len(actions); i++ {
			status <- actions[i].run()
			select {
			case <-c.stop:
				return
			case <-time.After(actions[i].waitAfter * time.Millisecond):
			}
		}
	}
}

func (c *SampleController) SetRate(rate float64) error {
	return nil
}

func (c *SampleController) Stop() {
	close(c.stop)
}

func (c *SampleController) sendFailStatus(status chan<- user.UserStatus, reason string) {
	status <- user.UserStatus{User: c.user, Code: user.STATUS_FAILED, Err: errors.New(reason)}
}

func (c *SampleController) sendStopStatus(status chan<- user.UserStatus) {
	status <- user.UserStatus{User: c.user, Info: "user stopped", Code: user.STATUS_STOPPED}
}

func (c *SampleController) signUp() user.UserStatus {
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

	return user.UserStatus{User: c.user, Info: fmt.Sprintf("signed up: %s", c.user.Store().Id())}
}

func (c *SampleController) login() user.UserStatus {
	err := c.user.Login()
	if err != nil {
		return user.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
	}

	return user.UserStatus{User: c.user, Info: "logged in"}
}

func (c *SampleController) logout() user.UserStatus {
	ok, err := c.user.Logout()
	if err != nil {
		return user.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
	}

	if !ok {
		return user.UserStatus{User: c.user, Err: errors.New("User did not logout"), Code: user.STATUS_ERROR}
	}

	return user.UserStatus{User: c.user, Info: "logged out"}
}
