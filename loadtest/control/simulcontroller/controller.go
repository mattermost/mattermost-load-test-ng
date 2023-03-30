// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
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
	serverVersion  string          // stores the current server version
	isGQLEnabled   bool
}

// New creates and initializes a new SimulController with given parameters.
// An id is provided to identify the controller, a User is passed as the entity to be controlled and
// a UserStatus channel is passed to communicate errors and information about the user's status.
func New(id int, user user.User, config *Config, status chan<- control.UserStatus) (*SimulController, error) {
	if config == nil || user == nil {
		return nil, errors.New("nil params passed")
	}

	if err := defaults.Validate(config); err != nil {
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
		if err := c.disconnect(); err != nil {
			c.status <- c.newErrorStatus(control.NewUserError(err))
		}
		c.user.ClearUserData()
		c.sendStopStatus()
		close(c.stoppedChan)
	}()

	c.serverVersion, _ = c.user.Store().ServerVersion()
	err := c.user.GetClientConfig()
	if err != nil {
		c.status <- c.newErrorStatus(err)
	}
	c.isGQLEnabled = c.user.Store().ClientConfig()["FeatureFlagGraphQL"] == "true"

	initActions := []userAction{
		{
			run: c.loginOrSignUp,
		},
		{
			run: c.initialJoinTeam,
		},
	}

	for i := 0; i < len(initActions); i++ {
		select {
		case <-c.stopChan:
			return
		case <-time.After(control.PickIdleTimeMs(c.config.MinIdleTimeMs, c.config.AvgIdleTimeMs, 1.0)):
		}

		if resp := initActions[i].run(c.user); resp.Err != nil {
			c.status <- c.newErrorStatus(resp.Err)
			i--
		} else {
			c.status <- c.newInfoStatus(resp.Info)
		}
	}

	actions := []userAction{
		{
			run:       switchChannel,
			frequency: 4,
		},
		{
			run:       c.switchTeam,
			frequency: 3,
		},
		{
			run:       c.scrollChannel,
			frequency: 2,
		},
		{
			run:       openDirectOrGroupChannel,
			frequency: 2,
		},
		{
			run:       unreadCheck,
			frequency: 1.5,
		},
		{
			run:       c.createPost,
			frequency: 1.5,
		},
		{
			run:       c.joinChannel,
			frequency: 0.8,
		},
		{
			run:       c.searchChannels,
			frequency: 0.5,
		},
		{
			run:       c.addReaction,
			frequency: 0.5,
		},
		{
			run:       c.fullReload,
			frequency: 0.2,
		},
		{
			run:       c.createDirectChannel,
			frequency: 0.25,
		},
		{
			run:       c.logoutLogin,
			frequency: 0.1,
		},
		{
			run:       searchUsers,
			frequency: 0.1,
		},
		{
			run:       searchPosts,
			frequency: 0.1,
		},
		{
			run:       c.createPostReminder,
			frequency: 0.002,
		},
		{
			run:       editPost,
			frequency: 0.1,
		},
		{
			run:       deletePost,
			frequency: 0.06,
		},
		{
			run:       c.updateCustomStatus,
			frequency: 0.05,
		},
		{
			run:       c.removeCustomStatus,
			frequency: 0.05,
		},
		{
			run:       c.createSidebarCategory,
			frequency: 0.06,
		},
		{
			run:       c.updateSidebarCategory,
			frequency: 0.06,
		},
		{
			run:       searchGroupChannels,
			frequency: 0.1,
		},
		{
			run:       c.createGroupChannel,
			frequency: 0.05,
		},
		{
			run:       createPrivateChannel,
			frequency: 0.022,
		},
		{
			run:       control.CreatePublicChannel,
			frequency: 0.011,
		},
		{
			run:       c.viewGlobalThreads,
			frequency: 5.4,
		},
		{
			run:       c.followThread,
			frequency: 0.041,
		},
		{
			run:       c.unfollowThread,
			frequency: 0.055,
		},
		{
			run:       c.viewThread,
			frequency: 4.8,
		},
		{
			run:       c.markAllThreadsInTeamAsRead,
			frequency: 0.013,
		},
		{
			run:       c.updateThreadRead,
			frequency: 1.17,
		},
		{
			run:       c.getInsights,
			frequency: 0.011,
		},
	}

	for {
		action, err := pickAction(actions)
		if err != nil {
			panic(fmt.Sprintf("simulcontroller: failed to pick action %s", err.Error()))
		}

		if action.minServerVersion != "" {
			supported, err := control.IsVersionSupported(action.minServerVersion, c.serverVersion)
			if err != nil {
				c.status <- c.newErrorStatus(err)
			} else if !supported {
				continue
			}
		}

		if resp := action.run(c.user); resp.Err != nil {
			c.status <- c.newErrorStatus(resp.Err)
		} else {
			c.status <- c.newInfoStatus(resp.Info)
		}

		select {
		case <-c.stopChan:
			return
		case <-time.After(control.PickIdleTimeMs(c.config.MinIdleTimeMs, c.config.AvgIdleTimeMs, c.rate)):
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
	// re-initialize for the next use
	c.stopChan = make(chan struct{})
	c.stoppedChan = make(chan struct{})
}

func (c *SimulController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Code: control.USER_STATUS_FAILED, Err: errors.New(reason)}
}

func (c *SimulController) sendStopStatus() {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}
