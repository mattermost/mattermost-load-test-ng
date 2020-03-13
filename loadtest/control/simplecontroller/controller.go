// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simplecontroller

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost-server/v5/mlog"
)

// SimpleController is a very basic implementation of a controller.
// Currently, it just performs a pre-defined set of actions in a loop.
type SimpleController struct {
	id     int
	user   user.User
	stop   chan struct{}
	status chan<- control.UserStatus
	rate   float64
	config *Config
}

// New creates and initializes a new SimpleController with given parameters.
// An id is provided to identify the controller, a User is passed as the entity to be controlled and
// a UserStatus channel is passed to communicate errors and information about the user's status.
func New(id int, user user.User, config *Config, status chan<- control.UserStatus) (*SimpleController, error) {
	if config == nil || user == nil {
		return nil, errors.New("nil params passed")
	}

	if err := config.IsValid(); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}

	return &SimpleController{
		id:     id,
		user:   user,
		stop:   make(chan struct{}),
		status: status,
		rate:   1.0,
		config: config,
	}, nil
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

	// Start listening for websocket events.
	go c.wsEventHandler()

	actions, err := parseActions(c, c.config.Actions)
	if err != nil {
		c.sendFailStatus(fmt.Sprintf("could not parse actions: %s", err.Error()))
		return
	}

	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user started", Code: control.USER_STATUS_STARTED}

	defer c.sendStopStatus()

	if resp := control.SignUp(c.user); resp.Err != nil {
		c.status <- c.newErrorStatus(resp.Err)
	} else {
		c.status <- c.newInfoStatus(resp.Info)
	}

	if resp := control.Login(c.user); resp.Err != nil {
		c.status <- c.newErrorStatus(resp.Err)
	} else {
		c.status <- c.newInfoStatus(resp.Info)
		c.connect()
	}

	cycleCount := 1 // keeps a track of how many times the entire cycle of actions have been completed.
	mlog.Info(fmt.Sprintf("FREAKING ACTIONS: %d", len(actions)))
	for {
		for i := 0; i < len(actions); i++ {
			if cycleCount%actions[i].runFrequency == 0 {
				// run the action if runFrequency is not set, or else it's set and it's a multiple
				// of the cycle count.
				if resp := actions[i].run(c.user); resp.Err != nil {
					c.status <- c.newErrorStatus(resp.Err)
				} else {
					c.status <- c.newInfoStatus(resp.Info)
				}

				idleTime := time.Duration(math.Round(float64(actions[i].waitAfter) * c.rate))

				select {
				case <-c.stop:
					return
				case <-time.After(time.Millisecond * idleTime):
				}
			}
		}
		cycleCount++
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
	if err := c.user.Disconnect(); err != nil {
		c.status <- c.newErrorStatus(err)
	}
	c.user.Cleanup()
	close(c.stop)
}

func (c *SimpleController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Code: control.USER_STATUS_FAILED, Err: errors.New(reason)}
}

func (c *SimpleController) sendStopStatus() {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}
