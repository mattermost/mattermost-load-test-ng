// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package noopcontroller

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

func getActionList(c *NoopController) []userAction {
	return []userAction{
		{
			name: "SignUp",
			run:  control.SignUp,
		},
		{
			name: "Login",
			run:  c.login,
		},
		{
			name: "JoinTeam",
			run:  c.joinTeam,
		},
		{
			name: "JoinChannel",
			run:  c.joinChannel,
		},
	}
}

func getActionMap(actionList []userAction) map[string]userAction {
	actionMap := make(map[string]userAction)
	for _, action := range actionList {
		actionMap[action.name] = action
	}
	return actionMap
}

// NoopController is a very basic implementation of a controller.
// NoopController, it just performs a pre-defined set of actions in a loop.
type NoopController struct {
	id                 int
	user               user.User
	status             chan<- control.UserStatus
	rate               float64
	actionList         []userAction
	actionMap          map[string]userAction
	injectedActionChan chan userAction
	stopChan           chan struct{}   // this channel coordinates the stop sequence of the controller
	stoppedChan        chan struct{}   // blocks until controller cleans up everything
	connectedFlag      int32           // indicates that the controller is connected
	wg                 *sync.WaitGroup // to keep the track of every goroutine created by the controller
}

// New creates and initializes a new NoopController with given parameters.
// An id is provided to identify the controller, a User is passed as the entity to be controlled and
// a UserStatus channel is passed to communicate errors and information about the user's status.
func New(id int, user user.User, status chan<- control.UserStatus) (*NoopController, error) {
	if user == nil {
		return nil, errors.New("nil params passed")
	}

	controller := &NoopController{
		id:                 id,
		user:               user,
		status:             status,
		rate:               1.0,
		injectedActionChan: make(chan userAction, 10),
		stopChan:           make(chan struct{}),
		stoppedChan:        make(chan struct{}),
		wg:                 &sync.WaitGroup{},
	}

	controller.actionList = getActionList(controller)
	controller.actionMap = getActionMap(controller.actionList)

	return controller, nil
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
		if err := c.disconnect(); err != nil {
			c.status <- c.newErrorStatus(control.NewUserError(err))
		}
		c.user.ClearUserData()
		c.sendStopStatus()
		close(c.stoppedChan)
	}()

	// run init actions
	for i := 0; i < len(c.actionList); i++ {
		idleTime := time.Duration(math.Round(float64(1000) * c.rate))

		select {
		case <-c.stopChan:
			return
		case <-time.After(time.Millisecond * idleTime):
		}

		if resp := c.actionList[i].run(c.user); resp.Err != nil {
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
		case ia := <-c.injectedActionChan: // run injected actions immediately
			c.runAction(ia)
		}
	}
}

func (c *NoopController) runAction(action userAction) {
	if resp := action.run(c.user); resp.Err != nil {
		c.status <- c.newErrorStatus(resp.Err)
	} else {
		c.status <- c.newInfoStatus(resp.Info)
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
	c.injectedActionChan = make(chan userAction, 10)
	c.stopChan = make(chan struct{})
	c.stoppedChan = make(chan struct{})
}

func (c *NoopController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Code: control.USER_STATUS_FAILED, Err: errors.New(reason)}
}

func (c *NoopController) sendStopStatus() {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}

// InjectAction allows a named UserAction to be injected that is run once, at the next
// available opportunity. These actions can be injected via the coordinator via
// CLI or Rest API.
func (c *NoopController) InjectAction(actionID string) control.UserActionResponse {
	var action userAction
	var ok bool

	// include some actions that are not normally supported by NoopController
	switch actionID {
	case "Reload":
		action = userAction{
			name: "Reload",
			run:  func(_ user.User) control.UserActionResponse { return control.Reload(c.user) },
		}
	default:
		action, ok = c.actionMap[actionID]
		if !ok {
			return control.UserActionResponse{
				Info: fmt.Sprintf("Action %s not supported by NoopController", actionID),
			}
		}
	}

	select {
	case c.injectedActionChan <- action:
		return control.UserActionResponse{
			Info: fmt.Sprintf("Action %s queued successfully", actionID),
		}
	case <-time.After(time.Second * 15):
		return control.UserActionResponse{
			Info: fmt.Sprintf("Action %s timed out while queuing", actionID),
			Err:  control.ErrActionTimeout,
		}
	}
}

// ensure NoopController implements UserController interface
var _ control.UserController = (*NoopController)(nil)
