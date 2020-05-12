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
	id             int
	user           user.User
	status         chan<- control.UserStatus
	rate           float64
	config         *Config
	stopChan       chan struct{}   // this channel coordinates the stop sequence of the controller
	stoppedChan    chan struct{}   // blocks until controller cleans up everything
	disconnectChan chan struct{}   // notifies disconnection to the ws and periodic goroutines
	connectedFlag  int32           // indicates that the controller is connected
	wg             *sync.WaitGroup // to keep the track of every goroutine created by the controller
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
		id:             id,
		user:           user,
		status:         status,
		rate:           1.0,
		config:         config,
		disconnectChan: make(chan struct{}),
		stopChan:       make(chan struct{}),
		stoppedChan:    make(chan struct{}),
		wg:             &sync.WaitGroup{},
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

	defer func() {
		if resp := c.logout(); resp.Err != nil {
			c.status <- c.newErrorStatus(resp.Err)
		}
		c.user.ClearUserData()
		c.sendStopStatus()
		close(c.stoppedChan)
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
		case <-c.stopChan:
			return
		case <-time.After(pickIdleTimeMs(c.config.MinIdleTimeMs, c.config.AvgIdleTimeMs, c.rate)):
		}

		if resp := action.run(c.user); resp.Err != nil {
			c.status <- c.newErrorStatus(resp.Err)
		} else {
			c.status <- c.newInfoStatus(resp.Info)
		}
	}

	actions := []userAction{
		{
			run:       switchChannel,
			frequency: 120,
		},
		{
			run:       openDirectOrGroupChannel,
			frequency: 50,
		},
		{
			run:       c.switchTeam,
			frequency: 30,
		},
		{
			run:       c.createPost,
			frequency: 25,
		},
		{
			run:       c.createPostReply,
			frequency: 15,
		},
		{
			run:       c.joinChannel,
			frequency: 11,
		},
		{
			run:       c.addReaction,
			frequency: 12,
		},
		{
			run:       c.fullReload,
			frequency: 8,
		},
		{
			run:       editPost,
			frequency: 8,
		},
		{
			run:       c.logoutLogin,
			frequency: 3,
		},
		{
			run:       c.createDirectChannel,
			frequency: 2,
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
		case <-c.stopChan:
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
	close(c.stopChan)
	<-c.stoppedChan
}

func (c *SimulController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Code: control.USER_STATUS_FAILED, Err: errors.New(reason)}
}

func (c *SimulController) sendStopStatus() {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}
