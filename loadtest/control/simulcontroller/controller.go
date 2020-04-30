// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

// SimulController is a simulative implementation of a UserController.
type SimulController struct {
	id      int
	user    user.User
	stop    chan struct{}
	stopped chan struct{}
	status  chan<- control.UserStatus
	rate    float64
	config  *Config
}

// New creates and initializes a new SimulController with given parameters.
// An id is provided to identify the controller, a User is passed as the entity to be controlled and
// a UserStatus channel is passed to communicate errors and information about the user's status.
func New(id int, user user.User, config *Config, status chan<- control.UserStatus) (*SimulController, error) {
	if config == nil || user == nil {
		return nil, errors.New("nil params passed")
	}

	if err := config.IsValid(); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}

	return &SimulController{
		id:      id,
		user:    user,
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
		status:  status,
		rate:    1.0,
		config:  config,
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

	var wg sync.WaitGroup
	// Start listening for websocket events.
	wg.Add(1)
	go c.wsEventHandler(&wg)

	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user started", Code: control.USER_STATUS_STARTED}

	defer func() {
		if err := c.user.Disconnect(); err != nil {
			c.status <- c.newErrorStatus(err)
		}
		c.user.Cleanup()
		c.user.ClearUserData()
		wg.Wait()
		c.sendStopStatus()
		close(c.stopped)
	}()

	initActions := []userAction{
		{
			run: control.SignUp,
		},
		{
			run: c.login,
		},
		{
			run: c.joinTeam,
		},
		{
			run: c.joinChannel,
		},
	}

	for _, action := range initActions {
		select {
		case <-c.stop:
			return
		case <-time.After(pickIdleTimeMs(c.config.MinIdleTimeMs, c.config.AvgIdleTimeMs, c.rate)):
		}

		if resp := action.run(c.user); resp.Err != nil {
			c.status <- c.newErrorStatus(resp.Err)
		} else {
			c.status <- c.newInfoStatus(resp.Info)
		}
	}

	go c.periodicActions()

	actions := []userAction{
		{
			run:       openDirectOrGroupChannel,
			frequency: 200,
		},
		{
			run:       c.switchTeam,
			frequency: 110,
		},
		{
			run:       switchChannel,
			frequency: 100,
		},
		{
			run:       c.createPost,
			frequency: 55,
		},
		{
			run:       c.fullReload,
			frequency: 40,
		},
		{
			run:       c.createPostReply,
			frequency: 20,
		},
		{
			run:       c.joinChannel,
			frequency: 11,
		},
		{
			run:       c.addReaction,
			frequency: 6,
		},
		{
			run:       editPost,
			frequency: 3,
		},
		{
			run:       c.logoutLogin,
			frequency: 3,
		},
		{
			run:       c.createDirectChannel,
			frequency: 1,
		},
		{
			run:       c.createGroupChannel,
			frequency: 1,
		},
	}

	for {
		action, err := pickAction(actions)
		if err != nil {
			panic(fmt.Sprintf("simulcontroller: failed to pick action %s", err.Error()))
		}

		if resp := action.run(c.user); resp.Err != nil {
			c.status <- c.newErrorStatus(resp.Err)
		} else {
			c.status <- c.newInfoStatus(resp.Info)
		}

		select {
		case <-c.stop:
			return
		case <-time.After(pickIdleTimeMs(c.config.MinIdleTimeMs, c.config.AvgIdleTimeMs, c.rate)):
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
	close(c.stop)
	<-c.stopped
}

func (c *SimulController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Code: control.USER_STATUS_FAILED, Err: errors.New(reason)}
}

func (c *SimulController) sendStopStatus() {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}
