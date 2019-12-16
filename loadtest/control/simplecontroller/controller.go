// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simplecontroller

import (
	"errors"
	"math"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

type SimpleController struct {
	user user.User
	stop chan struct{}
	rate float64
}

func (c *SimpleController) Init(user user.User) {
	c.user = user
	c.stop = make(chan struct{})
	c.rate = 1.0
}

func (c *SimpleController) Run(status chan<- control.UserStatus) {
	if c.user == nil {
		c.sendFailStatus(status, "controller was not initialized")
		return
	}

	actions := []UserAction{
		{
			run:       c.signUp,
			waitAfter: 1000,
		},
		{
			run:       c.login,
			waitAfter: 1000,
		},
		{
			run:       c.createPost,
			waitAfter: 1000,
		},
		{
			run:       c.createGroupChannel,
			waitAfter: 1000,
		},
		{
			run:       c.viewChannel,
			waitAfter: 1000,
		},
		{
			run:       c.reload,
			waitAfter: 1000,
		},
		{
			run:       c.logout,
			waitAfter: 1000,
		},
	}

	status <- control.UserStatus{User: c.user, Info: "user started", Code: control.USER_STATUS_STARTED}

	defer c.sendStopStatus(status)

	for {
		for i := 0; i < len(actions); i++ {
			status <- actions[i].run()

			idleTime := time.Duration(math.Round(float64(actions[i].waitAfter) * c.rate))

			select {
			case <-c.stop:
				return
			case <-time.After(time.Millisecond * idleTime):
			}
		}
	}
}

func (c *SimpleController) SetRate(rate float64) error {
	if rate < 0 {
		return errors.New("rate should be a positive value")
	}
	c.rate = rate
	return nil
}

func (c *SimpleController) Stop() {
	close(c.stop)
}

func (c *SimpleController) sendFailStatus(status chan<- control.UserStatus, reason string) {
	status <- control.UserStatus{User: c.user, Code: control.USER_STATUS_FAILED, Err: errors.New(reason)}
}

func (c *SimpleController) sendStopStatus(status chan<- control.UserStatus) {
	status <- control.UserStatus{User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}
