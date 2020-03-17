// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package noopcontroller

import (
	"errors"
	"math"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

// NoopController is a very basic implementation of a controller.
// NoopController, it just performs a pre-defined set of actions in a loop.
type NoopController struct {
	id     int
	user   user.User
	stop   chan struct{}
	status chan<- control.UserStatus
	rate   float64
}

// New creates and initializes a new SimpleController with given parameters.
// An id is provided to identify the controller, a User is passed as the entity to be controlled and
// a UserStatus channel is passed to communicate errors and information about the user's status.
func New(id int, user user.User, status chan<- control.UserStatus) (*NoopController, error) {
	return &NoopController{
		id:     id,
		user:   user,
		stop:   make(chan struct{}),
		status: status,
		rate:   1.0,
	}, nil
}

// Run begins performing a set of actions in a loop with a defined wait
// in between the actions. It keeps on doing it until Stop is invoked.
// This is also a blocking function, so it is recommended to invoke it
// inside a goroutine.
func (c *NoopController) Run() {
	if c.user == nil {
		c.sendFailStatus("controller was not initialized")
		return
	}

	// Start listening for websocket events.
	go c.wsEventHandler()

	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user started", Code: control.USER_STATUS_STARTED}

	if resp := control.SignUp(c.user); resp.Err != nil {
		c.status <- c.newErrorStatus(resp.Err)
	} else {
		c.status <- c.newInfoStatus(resp.Info)
	}

	if resp := control.Login(c.user); resp.Err != nil {
		c.status <- c.newErrorStatus(resp.Err)
	} else {
		c.status <- c.newInfoStatus(resp.Info)
		errChan := c.user.Connect()
		go func() {
			for err := range errChan {
				c.status <- c.newErrorStatus(err)
			}
		}()
	}

	defer c.sendStopStatus()

	for {

		if res, err := c.user.GetMe(); err != nil {
			c.status <- c.newErrorStatus(err)
		} else {
			c.status <- c.newInfoStatus(res)
		}

		idleTime := time.Duration(math.Round(float64(1000) * c.rate))

		select {
		case <-c.stop:
			return
		case <-time.After(time.Millisecond * idleTime):
		}
	}
}

// SetRate sets the relative speed of execution of actions by the user.
func (c *NoopController) SetRate(rate float64) error {
	if rate < 0 {
		return errors.New("rate should be a positive value")
	}
	c.rate = rate
	return nil
}

// Stop stops the controller.
func (c *NoopController) Stop() {
	if err := c.user.Disconnect(); err != nil {
		c.status <- c.newErrorStatus(err)
	}
	c.user.Cleanup()
	close(c.stop)
}

func (c *NoopController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Code: control.USER_STATUS_FAILED, Err: errors.New(reason)}
}

func (c *NoopController) sendStopStatus() {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}
