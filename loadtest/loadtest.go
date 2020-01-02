// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"sync"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

// LoadTester is a structure holding all the state needed to run a load-test
type LoadTester struct {
	controllersMut  sync.RWMutex
	controllers     []control.UserController
	config          *config.LoadTestConfig
	wg              sync.WaitGroup
	status          chan control.UserStatus
	startedMut      sync.RWMutex
	started         bool
	statusClosedMut sync.RWMutex
	statusClosed    bool
	newController   NewController
}

// NewController is a factory function that returns a new
// control.UserController given an id and a channel of control.UserStatus
// It is passed during LoadTester initialization to provide a way to create
// concrete UserController values from within the loadtest package without the
// need of those being passed from the upper layer (the user of this API).
type NewController func(int, chan<- control.UserStatus) control.UserController

func (lt *LoadTester) handleStatus() {
	lt.withStatusLock(func() {
		lt.statusClosed = false
	})
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

// AddUser increments by one the number of concurrently active users
func (lt *LoadTester) AddUser() error {
	if !lt.hasStarted() {
		return ErrNotRunning
	}
	lt.controllersMut.Lock()
	defer lt.controllersMut.Unlock()
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

// RemoveUser decrements by one the number of concurrently active users
func (lt *LoadTester) RemoveUser() error {
	if !lt.hasStarted() {
		return ErrNotRunning
	}
	lt.controllersMut.Lock()
	defer lt.controllersMut.Unlock()

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

// Run starts the execution of a new load-test
func (lt *LoadTester) Run() error {
	lt.startedMut.Lock()
	if lt.started {
		lt.startedMut.Unlock()
		return ErrAlreadyRunning
	}
	lt.started = true
	lt.startedMut.Unlock()

	go lt.handleStatus()
	for i := 0; i < lt.config.UsersConfiguration.InitialActiveUsers; i++ {
		if err := lt.AddUser(); err != nil {
			mlog.Error(err.Error())
		}
	}
	return nil
}

// Stop terminates the current load-test
func (lt *LoadTester) Stop() error {
	if !lt.hasStarted() {
		return ErrNotRunning
	}

	var controllers []control.UserController
	lt.controllersMut.RLock()
	controllers = lt.controllers
	lt.controllersMut.RUnlock()
	for range controllers {
		if err := lt.RemoveUser(); err != nil {
			mlog.Error(err.Error())
		}
	}
	lt.wg.Wait()
	lt.startedMut.Lock()
	lt.started = false
	lt.startedMut.Unlock()
	lt.withStatusLock(func() {
		if !lt.statusClosed {
			close(lt.status)
		}
		lt.statusClosed = true
	})

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

// Util methods to help manage the guarded variables.

func (lt *LoadTester) hasStarted() bool {
	lt.startedMut.RLock()
	defer lt.startedMut.RUnlock()
	return lt.started
}

func (lt *LoadTester) withStatusLock(f func()) {
	lt.statusClosedMut.Lock()
	defer lt.statusClosedMut.Unlock()
	f()
}
