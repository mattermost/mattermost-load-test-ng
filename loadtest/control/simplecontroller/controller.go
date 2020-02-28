// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simplecontroller

import (
	"errors"
	"math"
	"time"

	"github.com/mattermost/mattermost-server/v5/model"

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

	// Start listening for websocket events.
	go func() {
		for ev := range c.user.Events() {
			switch ev.EventType() {
			case model.WEBSOCKET_EVENT_USER_UPDATED:
				// probably do something interesting ?
			case model.WEBSOCKET_EVENT_STATUS_CHANGE:
				// Send a message if the user has come online.
				data := ev.Data // TODO: upgrade the server dependency and move to GetData call
				status, ok := data["status"].(string)
				if !ok || status != "online" {
					continue
				}
				userID, ok := data["user_id"].(string)
				if !ok {
					continue
				}
				c.status <- c.sendDirectMessage(userID)
			default:
				// add other handlers as necessary.
			}
		}
	}()

	actions := []UserAction{
		{
			run: func(u user.User) control.UserActionResponse {
				return c.reload(false)
			},
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          control.JoinTeam,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          control.JoinChannel,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          control.AddReaction,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          control.RemoveReaction,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          control.SearchPosts,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          control.SearchChannels,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          control.SearchUsers,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          control.ViewUser,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          control.CreatePost,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          c.updateProfile,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          control.UpdateProfileImage,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          control.CreateGroupChannel,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          control.CreateDirectChannel,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          control.ViewChannel,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          c.scrollChannel,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          control.LeaveChannel,
			waitAfter:    1000,
			runFrequency: 1,
		},
		{
			run:          control.Logout,
			waitAfter:    1000,
			runFrequency: 20,
		},
		{
			run: func(u user.User) control.UserActionResponse {
				resp := control.Login(u)
				if resp.Err != nil {
					return resp
				}
				c.connect()
				return resp
			},
			waitAfter:    1000,
			runFrequency: 20,
		},
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
	close(c.stop)
}

func (c *SimpleController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Code: control.USER_STATUS_FAILED, Err: &control.ControlError{Err: errors.New(reason)}}
}

func (c *SimpleController) sendStopStatus() {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}
