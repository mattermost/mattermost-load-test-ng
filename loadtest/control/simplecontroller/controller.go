// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simplecontroller

import (
	"errors"
	"sync/atomic"
	"time"

	"github.com/mattermost/mattermost-load-test/loadtest/user"
)

type SimpleController struct {
	user user.User
	done int32
}

func (c *SimpleController) Init(user user.User) {
	c.user = user
}

func (c *SimpleController) Run(status chan<- user.UserStatus) {
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
			run:       c.logout,
			waitAfter: 1000,
		},
	}

	status <- user.UserStatus{User: c.user, Info: "user started", Code: user.STATUS_STARTED}

	defer c.sendStopStatus(status)

	for {
		for i := 0; i < len(actions); i++ {
			if c.shouldStop() {
				return
			}
			actions[i].run(status)
			time.Sleep(actions[i].waitAfter * time.Millisecond)
		}

		// status <- user.UserStatus{User: c.user, Info: "user loop done", Code: user.STATUS_DONE}
	}
}

func (c *SimpleController) Stop() {
	atomic.StoreInt32(&c.done, 1)
}

func (c *SimpleController) shouldStop() bool {
	return atomic.LoadInt32(&c.done) == 1
}

func (c *SimpleController) sendFailStatus(status chan<- user.UserStatus, reason string) {
	status <- user.UserStatus{User: c.user, Code: user.STATUS_FAILED, Err: errors.New(reason)}
}

func (c *SimpleController) sendStopStatus(status chan<- user.UserStatus) {
	status <- user.UserStatus{User: c.user, Info: "user stopped", Code: user.STATUS_STOPPED}
}
