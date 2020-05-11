// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package samplecontroller

import (
	"errors"
	"fmt"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

type SampleController struct {
	id     int
	user   user.User
	stop   chan struct{}
	status chan<- control.UserStatus
}

type userAction struct {
	run       func() control.UserStatus
	waitAfter time.Duration
}

func New(id int, user user.User, status chan<- control.UserStatus) *SampleController {
	return &SampleController{
		id:     id,
		user:   user,
		stop:   make(chan struct{}),
		status: status,
	}
}

func (c *SampleController) Run(started chan struct{}) {
	if c.user == nil {
		c.sendFailStatus("controller was not initialized")
		close(started)
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

	close(started)
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user started", Code: control.USER_STATUS_STARTED}

	defer c.sendStopStatus()

	for {
		for i := 0; i < len(actions); i++ {
			c.status <- actions[i].run()
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

func (c *SampleController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Code: control.USER_STATUS_FAILED, Err: errors.New(reason)}
}

func (c *SampleController) sendStopStatus() {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}

func (c *SampleController) signUp() control.UserStatus {
	if c.user.Store().Id() != "" {
		return control.UserStatus{ControllerId: c.id, User: c.user, Info: "user already signed up", Code: control.USER_STATUS_INFO}
	}

	email := fmt.Sprintf("testuser%d@example.com", c.id)
	username := fmt.Sprintf("testuser%d", c.id)
	password := "testPass123$"

	err := c.user.SignUp(email, username, password)
	if err != nil {
		return control.UserStatus{ControllerId: c.id, User: c.user, Err: err, Code: control.USER_STATUS_ERROR}
	}

	return control.UserStatus{ControllerId: c.id, User: c.user, Info: fmt.Sprintf("signed up: %s", c.user.Store().Id()), Code: control.USER_STATUS_INFO}
}

func (c *SampleController) login() control.UserStatus {
	err := c.user.Login()
	if err != nil {
		return control.UserStatus{ControllerId: c.id, User: c.user, Err: err, Code: control.USER_STATUS_ERROR}
	}

	return control.UserStatus{ControllerId: c.id, User: c.user, Info: "logged in", Code: control.USER_STATUS_INFO}
}

func (c *SampleController) logout() control.UserStatus {
	ok, err := c.user.Logout()
	if err != nil {
		return control.UserStatus{ControllerId: c.id, User: c.user, Err: err, Code: control.USER_STATUS_ERROR}
	}

	if !ok {
		return control.UserStatus{ControllerId: c.id, User: c.user, Err: errors.New("User did not logout"), Code: control.USER_STATUS_ERROR}
	}

	return control.UserStatus{ControllerId: c.id, User: c.user, Info: "logged out", Code: control.USER_STATUS_INFO}
}
