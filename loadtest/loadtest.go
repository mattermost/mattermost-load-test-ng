// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"sync"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

// LoadTester is a structure holding all the state needed to run a load-test.
type LoadTester struct {
	controllers   []control.UserController
	config        *config.LoadTestConfig
	wg            sync.WaitGroup
	status        chan control.UserStatus
	started       bool
	newController NewController
}

// NewController is a factory function that returns a new
// control.UserController given an id and a channel of control.UserStatus
// It is passed during LoadTester initialization to provide a way to create
// concrete UserController values from within the loadtest package without the
// need of those being passed from the upper layer (the user of this API).
type NewController func(int, chan<- control.UserStatus) control.UserController

func (lt *LoadTester) handleStatus() {
	for st := range lt.status {
		if st.Code == control.USER_STATUS_STOPPED || st.Code == control.USER_STATUS_FAILED {
			lt.wg.Done()
		}
		if st.Code == control.USER_STATUS_ERROR {
			mlog.Info(st.Err.Error(), mlog.Int("controller_id", st.ControllerId))
			continue
		} else if st.Code == control.USER_STATUS_FAILED {
			mlog.Error(st.Err.Error())
			continue
		}
		mlog.Info(st.Info, mlog.Int("controller_id", st.ControllerId))
	}
}

// AddUser increments by one the number of concurrently active users.
func (lt *LoadTester) AddUser() error {
	if !lt.started {
		return ErrNotRunning
	}
	activeUsers := len(lt.controllers)
	if activeUsers == lt.config.UsersConfiguration.MaxActiveUsers {
		return ErrMaxUsersReached
	}
	controller := lt.newController(activeUsers+1, lt.status)
	lt.wg.Add(1)
	go func() {
		controller.Run()
	}()
	lt.controllers = append(lt.controllers, controller)
	return nil
}

// RemoveUser decrements by one the number of concurrently active users.
func (lt *LoadTester) RemoveUser() error {
	if !lt.started {
		return ErrNotRunning
	}
	activeUsers := len(lt.controllers)
	if activeUsers == 0 {
		return ErrNoUsersLeft
	}
	// TODO: Add a way to make how a user is removed decidable from the upper layer (the user of this API),
	// for example by passing a typed constant (e.g. random, first, last).
	controller := lt.controllers[activeUsers-1]
	controller.Stop()
	lt.controllers = lt.controllers[:activeUsers-1]
	return nil
}

// Run starts the execution of a new load-test.
func (lt *LoadTester) Run() error {
	// NOTE: we are currently not guarding against access from multiple goroutines.
	// TODO: Access to LoadTester should be made goroutine safe.
	if lt.started {
		return ErrAlreadyRunning
	}
	go lt.handleStatus()
	lt.started = true
	for i := 0; i < lt.config.UsersConfiguration.InitialActiveUsers; i++ {
		if err := lt.AddUser(); err != nil {
			mlog.Error(err.Error())
		}
	}
	return nil
}

// Stop terminates the current load-test.
func (lt *LoadTester) Stop() error {
	if !lt.started {
		return ErrNotRunning
	}
	for range lt.controllers {
		if err := lt.RemoveUser(); err != nil {
			mlog.Error(err.Error())
		}
	}
	lt.wg.Wait()
	close(lt.status)
	lt.started = false
	return nil
}

// New creates and initializes a new LoadTester with given config. A factory
// function is also given to enable the creation of UserController values from within the
// loadtest package.
func New(config *config.LoadTestConfig, nc NewController) *LoadTester {
	if config == nil || nc == nil {
		return nil
	}
	return &LoadTester{
		config:        config,
		status:        make(chan control.UserStatus, config.UsersConfiguration.MaxActiveUsers),
		newController: nc,
	}
}
