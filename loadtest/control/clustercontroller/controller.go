// Copyright (c) 2020-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package clustercontroller

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

type ClusterController struct {
	id      int
	user    user.User
	stop    chan struct{}
	stopped chan struct{}
	status  chan<- control.UserStatus
	rate    float64
}

func New(id int, user user.User, status chan<- control.UserStatus) (*ClusterController, error) {
	return &ClusterController{
		id:      id,
		user:    user,
		stop:    make(chan struct{}),
		stopped: make(chan struct{}),
		status:  status,
		rate:    1.0,
	}, nil
}

// Run begins performing a set of actions in a loop with a defined wait
// in between the actions. It keeps on doing it until Stop is invoked.
// This is also a blocking function, so it is recommended to invoke it
// inside a goroutine.
func (c *ClusterController) Run() {
	if c.user == nil {
		c.sendFailStatus("controller was not initialized")
		return
	}

	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user started", Code: control.USER_STATUS_STARTED}

	defer func() {
		c.sendStopStatus()
		close(c.stopped)
	}()

	if resp := control.Login(c.user); resp.Err != nil {
		c.status <- c.newErrorStatus(resp.Err)
	} else {
		c.status <- c.newInfoStatus(resp.Info)
	}

	actions := []control.UserAction{
		func(u user.User) control.UserActionResponse {
			err := u.GetLogs(0, 100)
			if err != nil {
				return control.UserActionResponse{
					Err: control.NewUserError(err),
				}
			}
			return control.UserActionResponse{
				Info: "got logs",
			}
		},
		func(u user.User) control.UserActionResponse {
			err := u.GetAnalytics()
			if err != nil {
				return control.UserActionResponse{
					Err: control.NewUserError(err),
				}
			}
			return control.UserActionResponse{
				Info: "got analytics",
			}
		},
		func(u user.User) control.UserActionResponse {
			err := u.GetClusterStatus()
			if err != nil {
				return control.UserActionResponse{
					Err: control.NewUserError(err),
				}
			}
			return control.UserActionResponse{
				Info: "got cluster stats",
			}
		},
		func(u user.User) control.UserActionResponse {
			err := u.GetPluginStatuses()
			if err != nil {
				return control.UserActionResponse{
					Err: control.NewUserError(err),
				}
			}
			return control.UserActionResponse{
				Info: "got plugin statuses",
			}
		},
		func(u user.User) control.UserActionResponse {
			err := u.GetConfig()
			if err != nil {
				return control.UserActionResponse{
					Err: control.NewUserError(err),
				}
			}
			cfg := u.Store().Config()
			// We just flip the enable developer flag to trigger a config update
			// across the cluster.
			*cfg.ServiceSettings.EnableDeveloper = !*cfg.ServiceSettings.EnableDeveloper
			err = u.UpdateConfig(&cfg)
			if err != nil {
				return control.UserActionResponse{
					Err: control.NewUserError(err),
				}
			}
			return control.UserActionResponse{
				Info: "updated config",
			}
		},
	}

	for {
		for _, action := range actions {
			if resp := action(c.user); resp.Err != nil {
				c.status <- c.newErrorStatus(resp.Err)
			} else {
				c.status <- c.newInfoStatus(resp.Info)
			}

			idleTime := time.Duration(math.Round(float64(1000) * c.rate))
			select {
			case <-c.stop:
				return
			case <-time.After(time.Millisecond * idleTime):
			}
		}
	}
}

// SetRate sets the relative speed of execution of actions by the user.
func (c *ClusterController) SetRate(rate float64) error {
	if rate < 0 {
		return errors.New("rate should be a positive value")
	}
	c.rate = rate
	return nil
}

// Stop stops the controller.
func (c *ClusterController) Stop() {
	close(c.stop)
	<-c.stopped
	c.stop = make(chan struct{})
	c.stopped = make(chan struct{})
}

func (c *ClusterController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{
		ControllerId: c.id,
		User:         c.user,
		Code:         control.USER_STATUS_FAILED,
		Err:          errors.New(reason),
	}
}

func (c *ClusterController) sendStopStatus() {
	c.status <- control.UserStatus{
		ControllerId: c.id,
		User:         c.user,
		Info:         "user stopped",
		Code:         control.USER_STATUS_STOPPED,
	}
}

func (c *ClusterController) newInfoStatus(info string) control.UserStatus {
	return control.UserStatus{
		ControllerId: c.id,
		User:         c.user,
		Code:         control.USER_STATUS_INFO,
		Info:         info,
		Err:          nil,
	}
}

func (c *ClusterController) newErrorStatus(err error) control.UserStatus {
	return control.UserStatus{
		ControllerId: c.id,
		User:         c.user,
		Code:         control.USER_STATUS_ERROR,
		Info:         "",
		Err:          err,
	}
}

// InjectAction allows a named UserAction to be injected that is run once, at the next
// available opportunity. These actions can be injected via the coordinator via
// CLI or Rest API.
func (c *ClusterController) InjectAction(actionID string) control.UserActionResponse {
	return control.UserActionResponse{
		Info: fmt.Sprintf("Action %s not supported by ClusterController", actionID),
	}
}

// ensure ClusterController implements UserController interface
var _ control.UserController = (*ClusterController)(nil)
