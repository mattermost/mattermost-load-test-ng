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

// SimpleController is a very basic implementation of a controller.
// Currently, it just performs a pre-defined set of actions in a loop.
type SimpleController struct {
	id     int
	user   user.User
	stop   chan struct{}
	status chan<- control.UserStatus
	rate   float64
}

// New creates and initializes a new SimpleController with given parameters.
// An id is provided to identify the controller, a User is passed as the entity to be controlled and
// a UserStatus channel is passed to communicate errors and information about the user's status.
func New(id int, user user.User, status chan<- control.UserStatus) *SimpleController {
	return &SimpleController{
		id:     id,
		user:   user,
		stop:   make(chan struct{}),
		status: status,
		rate:   1.0,
	}
}

// Run begins performing a set of actions in a loop with a defined wait
// in between the actions. It keeps on doing it until Stop is invoked.
// This is also a blocking function, so it is recommended to invoke it
// inside a goroutine.
func (c *SimpleController) Run() {
	if c.user == nil {
		c.sendFailStatus("controller was not initialized")
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
			run: func() control.UserStatus {
				return c.reload(false)
			},
		},
		{
			run:       c.joinTeam,
			waitAfter: 1000,
		},
		{
			run:       c.addReaction,
			waitAfter: 1000,
		},
		{
			run:       c.removeReaction,
			waitAfter: 1000,
		},
		{
			run:       c.searchPosts,
			waitAfter: 1000,
		},
		{
			run:       c.searchChannels,
			waitAfter: 1000,
		},
		{
			run:       c.searchUsers,
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
			run:       c.logout,
			waitAfter: 1000,
		},
	}

	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user started", Code: control.USER_STATUS_STARTED}

	defer c.sendStopStatus()

	for {
		for i := 0; i < len(actions); i++ {
			c.status <- actions[i].run()

			idleTime := time.Duration(math.Round(float64(actions[i].waitAfter) * c.rate))

			select {
			case <-c.stop:
				return
			case <-time.After(time.Millisecond * idleTime):
			}
		}
	}
}

// SetRate sets the relative speed of execution of actions by the user.
func (c *SimpleController) SetRate(rate float64) error {
	if rate < 0 {
		return errors.New("rate should be a positive value")
	}
	c.rate = rate
	return nil
}

// Stop stops the controller.
func (c *SimpleController) Stop() {
	close(c.stop)
}

func (c *SimpleController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Code: control.USER_STATUS_FAILED, Err: &control.ControlError{Err: errors.New(reason)}}
}

func (c *SimpleController) sendStopStatus() {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}
