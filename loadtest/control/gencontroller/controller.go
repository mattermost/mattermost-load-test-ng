// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gencontroller

import (
	"errors"
	"fmt"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

type GenController struct {
	id     int
	user   user.User
	stop   chan struct{}
	status chan<- control.UserStatus
	rate   float64
	config *Config
}

// New creates and initializes a new GenController with given parameters.
// An id is provided to identify the controller, a User is passed as the entity to be controlled and
// a UserStatus channel is passed to communicate errors and information about the user's status.
func New(id int, user user.User, config *Config, status chan<- control.UserStatus) (*GenController, error) {
	if config == nil || user == nil {
		return nil, errors.New("nil params passed")
	}

	if err := config.IsValid(); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}

	sc := &GenController{
		id:     id,
		user:   user,
		stop:   make(chan struct{}),
		status: status,
		rate:   1.0,
		config: config,
	}

	return sc, nil
}

// Run begins performing a set of actions in a loop with a defined wait
// in between the actions. It keeps on doing it until Stop is invoked.
// This is also a blocking function, so it is recommended to invoke it
// inside a goroutine.
func (c *GenController) Run() {
	if c.user == nil {
		c.sendFailStatus("controller was not initialized")
		return
	}

	defer c.sendStopStatus()

	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user started", Code: control.USER_STATUS_STARTED}

	for {
		time.Sleep(1 * time.Second)
		// select {
		// case <-c.stop:
		// 	return
		// default:
		// }
	}
}

// SetRate sets the relative speed of execution of actions by the user.
func (c *GenController) SetRate(rate float64) error {
	if rate < 0 {
		return errors.New("rate should be a positive value")
	}
	c.rate = rate
	return nil
}

// Stop stops the controller.
func (c *GenController) Stop() {
	// close(c.stop)
}

func (c *GenController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Code: control.USER_STATUS_FAILED, Err: errors.New(reason)}
}

func (c *GenController) sendStopStatus() {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}
