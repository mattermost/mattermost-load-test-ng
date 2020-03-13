// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"math"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

// SimulController is a simulative implementation of a UserController.
type SimulController struct {
	id     int
	user   user.User
	stop   chan struct{}
	status chan<- control.UserStatus
	rate   float64
}

// New creates and initializes a new SimulController with given parameters.
// An id is provided to identify the controller, a User is passed as the entity to be controlled and
// a UserStatus channel is passed to communicate errors and information about the user's status.
func New(id int, user user.User, status chan<- control.UserStatus) (*SimulController, error) {
	return &SimulController{
		id:     id,
		user:   user,
		stop:   make(chan struct{}),
		status: status,
		rate:   1.0,
	}, nil
}

// Run begins performing a set of user actions in a loop.
// It keeps on doing it until Stop() is invoked.
// This is also a blocking function, so it is recommended to invoke it
// inside a goroutine.
func (c *SimulController) Run() {
	if c.user == nil {
		c.sendFailStatus("controller was not initialized")
		return
	}

	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user started", Code: control.USER_STATUS_STARTED}

	defer c.sendStopStatus()

	for {
		// TODO: implement picking an action and run it.

		// TODO: implement picking a non constant idle time.
		idleTime := time.Duration(math.Round(1000 * c.rate))

		select {
		case <-c.stop:
			return
		case <-time.After(time.Millisecond * idleTime):
		}
	}
}

// SetRate sets the relative speed of execution of actions by the user.
func (c *SimulController) SetRate(rate float64) error {
	if rate < 0 {
		return errors.New("rate should be a positive value")
	}
	c.rate = rate
	return nil
}

// Stop stops the controller.
func (c *SimulController) Stop() {
	if err := c.user.Disconnect(); err != nil {
		c.status <- c.newErrorStatus(err)
	}
	close(c.stop)
}

func (c *SimulController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Code: control.USER_STATUS_FAILED, Err: errors.New(reason)}
}

func (c *SimulController) sendStopStatus() {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}
