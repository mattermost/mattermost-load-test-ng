// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package noopcontroller

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

// NoopController is a very basic implementation of a controller.
// NoopController, it just performs a pre-defined set of actions in a loop.
type NoopController struct {
	id            int
	user          user.User
	status        chan<- control.UserStatus
	rate          float64
	stopChan      chan struct{}   // this channel coordinates the stop sequence of the controller
	stoppedChan   chan struct{}   // blocks until controller cleans up everything
	connectedFlag int32           // indicates that the controller is connected
	wg            *sync.WaitGroup // to keep the track of every goroutine created by the controller
}

// New creates and initializes a new NoopController with given parameters.
// An id is provided to identify the controller, a User is passed as the entity to be controlled and
// a UserStatus channel is passed to communicate errors and information about the user's status.
func New(id int, user user.User, status chan<- control.UserStatus) (*NoopController, error) {
	if user == nil {
		return nil, errors.New("nil params passed")
	}

	return &NoopController{
		id:          id,
		user:        user,
		status:      status,
		rate:        1.0,
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
		wg:          &sync.WaitGroup{},
	}, nil
}

// Run begins performing a set of user actions in a loop.
// It keeps on doing it until Stop() is invoked.
// This is also a blocking function, so it is recommended to invoke it
// inside a goroutine.
func (c *NoopController) Run() {
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

	for i := 0; i < len(initActions); i++ {
		idleTime := time.Duration(math.Round(float64(1000) * c.rate))

		select {
		case <-c.stopChan:
			return
		case <-time.After(time.Millisecond * idleTime):
		}

		if resp := initActions[i].run(c.user); resp.Err != nil {
			c.status <- c.newErrorStatus(resp.Err)
			i--
		} else {
			c.status <- c.newInfoStatus(resp.Info)
		}
	}

	for {
		if res, err := c.user.GetMe(); err != nil {
			c.status <- c.newErrorStatus(err)
		} else {
			c.status <- c.newInfoStatus(res)
		}

		idleTime := time.Duration(math.Round(float64(1000) * c.rate))
		select {
		case <-c.stopChan:
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
	close(c.stopChan)
	<-c.stoppedChan
	// re-initialize for the next use
	c.stopChan = make(chan struct{})
	c.stoppedChan = make(chan struct{})
}

func (c *NoopController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Code: control.USER_STATUS_FAILED, Err: errors.New(reason)}
}

func (c *NoopController) sendStopStatus() {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}
