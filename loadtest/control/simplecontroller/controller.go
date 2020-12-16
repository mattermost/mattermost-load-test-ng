// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simplecontroller

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

// SimpleController is a very basic implementation of a controller.
// Currently, it just performs a pre-defined set of actions in a loop.
type SimpleController struct {
	id            int
	user          user.User
	admin         user.User
	status        chan<- control.UserStatus
	rate          float64
	actions       []*UserAction
	stopChan      chan struct{}   // this channel coordinates the stop sequence of the controller
	stoppedChan   chan struct{}   // blocks until controller cleans up everything
	connectedFlag int32           // indicates that the controller is connected
	wg            *sync.WaitGroup // to keep the track of every goroutine created by the controller
}

// New creates and initializes a new SimpleController with given parameters.
// An id is provided to identify the controller, a User is passed as the entity to be controlled and
// a UserStatus channel is passed to communicate errors and information about the user's status.
func New(id int, user, admin user.User, config *Config, status chan<- control.UserStatus) (*SimpleController, error) {
	if config == nil || user == nil {
		return nil, errors.New("nil params passed")
	}

	sc := &SimpleController{
		id:          id,
		user:        user,
		admin:       admin,
		status:      status,
		stopChan:    make(chan struct{}),
		stoppedChan: make(chan struct{}),
		rate:        1.0,
		wg:          &sync.WaitGroup{},
	}
	if err := sc.createActions(config.Actions); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}
	return sc, nil
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

	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user started", Code: control.USER_STATUS_STARTED}

	defer func() {
		if err := c.disconnect(); err != nil {
			c.status <- c.newErrorStatus(err)
		}
		c.user.ClearUserData()
		c.sendStopStatus()
		close(c.stoppedChan)
	}()

	initActions := []UserAction{
		{
			run: control.SignUp,
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
		},
		{
			run: func(user.User) control.UserActionResponse {
				resp := control.Login(c.admin)
				if resp.Err != nil {
					return resp
				}
				c.connect()
				return resp
			},
		},
	}

	for _, action := range initActions {
		if resp := action.run(c.user); resp.Err != nil {
			c.status <- c.newErrorStatus(resp.Err)
		} else {
			c.status <- c.newInfoStatus(resp.Info)
		}

		select {
		case <-c.stopChan:
			return
		default:
		}
	}

	if len(c.actions) == 0 {
		<-c.stopChan
		return
	}

	cycleCount := 1 // keeps a track of how many times the entire cycle of actions have been completed.
	for {
		for i := 0; i < len(c.actions); i++ {
			if cycleCount%c.actions[i].runPeriod == 0 {
				// run the action if runPeriod is not set, or else it's set and it's a multiple
				// of the cycle count.
				if resp := c.actions[i].run(c.user); resp.Err != nil {
					c.status <- c.newErrorStatus(resp.Err)
				} else {
					c.status <- c.newInfoStatus(resp.Info)
				}

				idleTime := time.Duration(math.Round(float64(c.actions[i].waitAfter) * c.rate))

				select {
				case <-c.stopChan:
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
	close(c.stopChan)
	<-c.stoppedChan
	// re-initialize for the next use
	c.stopChan = make(chan struct{})
	c.stoppedChan = make(chan struct{})

}

func (c *SimpleController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Code: control.USER_STATUS_FAILED, Err: errors.New(reason)}
}

func (c *SimpleController) sendStopStatus() {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}

func (c *SimpleController) createActions(definitions []actionDefinition) error {
	var actions []*UserAction
	actionMap := map[string]control.UserAction{
		"AddReaction":          control.AddReaction,
		"CreateDirectChannel":  control.CreateDirectChannel,
		"CreateGroupChannel":   control.CreateGroupChannel,
		"CreatePost":           control.CreatePost,
		"CreatePostReply":      control.CreatePostReply,
		"CreatePrivateChannel": control.CreatePrivateChannel,
		"CreatePublicChannel":  control.CreatePublicChannel,
		"GetPinnedPosts":       control.GetPinnedPosts,
		"JoinChannel":          control.JoinChannel,
		"JoinTeam":             control.JoinTeam,
		"LeaveChannel":         control.LeaveChannel,
		"Login": func(u user.User) control.UserActionResponse {
			resp := control.Login(u)
			if resp.Err != nil {
				return resp
			}
			c.connect()
			return resp
		},
		"Logout": func(u user.User) control.UserActionResponse {
			ok, err := u.Logout()
			if err != nil {
				return control.UserActionResponse{Err: control.NewUserError(err)}
			}

			if !ok {
				return control.UserActionResponse{Err: control.NewUserError(errors.New("user did not logout"))}
			}

			c.disconnect()
			u.ClearUserData()
			return control.UserActionResponse{Info: "logged out"}
		},
		"EditPost": control.EditPost,
		"Reload": func(u user.User) control.UserActionResponse {
			return c.reload(true)
		},
		"RemoveReaction":     control.RemoveReaction,
		"ScrollChannel":      c.scrollChannel,
		"SearchChannels":     control.SearchChannels,
		"SearchPosts":        control.SearchPosts,
		"SearchUsers":        control.SearchUsers,
		"SignUp":             control.SignUp,
		"UpdateProfile":      c.updateProfile,
		"UpdateProfileImage": control.UpdateProfileImage,
		"ViewChannel":        control.ViewChannel,
		"ViewUser":           control.ViewUser,
		"FetchStaticAssets":  control.FetchStaticAssets,
		"UpdateTeam":         c.updateTeam,
		"MessageExport":      c.messageExport,
	}

	for _, def := range definitions {
		run, ok := actionMap[def.ActionId]
		if !ok {
			return fmt.Errorf("could not find action %q", def.ActionId)
		}

		if def.RunPeriod == 0 {
			continue
		} else if def.RunPeriod < 0 {
			return fmt.Errorf("could not create action from %s, run period needs to be > 0", def.ActionId)
		}

		actions = append(actions, &UserAction{
			run:       run,
			waitAfter: time.Duration(def.WaitAfterMs),
			runPeriod: def.RunPeriod,
		})
	}
	c.actions = actions
	return nil
}
