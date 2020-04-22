// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"runtime"
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

	semCount := runtime.GOMAXPROCS(0)
	semaphore := make(chan struct{}, semCount)
	// Start listening for websocket events.
	go c.wsEventHandler(semaphore)

	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user started", Code: control.USER_STATUS_STARTED}

	defer func() {
		if err := c.user.Disconnect(); err != nil {
			c.status <- c.newErrorStatus(err)
		}
		c.user.Cleanup()
		c.user.ClearUserData()
		c.sendStopStatus()
		for i := 0; i < semCount; i++ {
			semaphore <- struct{}{}
		}
		close(semaphore)
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
	}

	for _, action := range initActions {
		if resp := action.run(c.user); resp.Err != nil {
			c.status <- c.newErrorStatus(resp.Err)
		} else {
			c.status <- c.newInfoStatus(resp.Info)
		}
		select {
		case <-c.stop:
			return
		default:
		}
	}

	go c.periodicActions()

	actions := []userAction{
		{
			run:       switchChannel,
			frequency: 300,
		},
		{
			run:       c.switchTeam,
			frequency: 110,
		},
		{
			run:       createPost,
			frequency: 55,
		},
		{
			run:       c.createDirectChannel,
			frequency: 1,
		},
		{
			run:       c.createGroupChannel,
			frequency: 1,
		},
		{
			run: func(u user.User) control.UserActionResponse {
				return c.reload(true)
			},
			frequency: 40,
		},
		{
			run: func(u user.User) control.UserActionResponse {
				// logout
				if resp := control.Logout(u); resp.Err != nil {
					c.status <- c.newErrorStatus(resp.Err)
				} else {
					c.status <- c.newInfoStatus(resp.Info)
				}

				u.ClearUserData()

				// login
				if resp := c.login(c.user); resp.Err != nil {
					c.status <- c.newErrorStatus(resp.Err)
				} else {
					c.status <- c.newInfoStatus(resp.Info)
				}

				// reload
				return c.reload(false)
			},
			frequency: 3,
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

		// Randomly selecting a value in the interval
		// [MinIdleTimeMs, AvgIdleTimeMs*2 - MinIdleTimeMs).
		// This will give us an expected value equal to AvgIdleTimeMs.
		// TODO: consider if it makes more sense to select this value using
		// a truncated normal distribution.
		idleMs := rand.Intn(c.config.AvgIdleTimeMs*2-c.config.MinIdleTimeMs*2) + c.config.MinIdleTimeMs

		idleTimeMs := time.Duration(math.Round(float64(idleMs) * c.rate))

		select {
		case <-c.stop:
			return
		case <-time.After(idleTimeMs * time.Millisecond):
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
