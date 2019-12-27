// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"errors"
	"sync"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

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
type NewController func(int, chan<- control.UserStatus) control.UserController

func (lt *LoadTester) handleStatus() {
	for us := range lt.status {
		if us.Code == control.USER_STATUS_STOPPED || us.Code == control.USER_STATUS_FAILED {
			lt.wg.Done()
		}
		if us.Code == control.USER_STATUS_ERROR {
			mlog.Info(us.Err.Error(), mlog.Int("controller_id", us.ControllerId))
			continue
		} else if us.Code == control.USER_STATUS_FAILED {
			mlog.Error(us.Err.Error())
			continue
		}
		mlog.Info(us.Info, mlog.Int("controller_id", us.ControllerId))
	}
}

// AddUser increments the number of concurrently active users
func (lt *LoadTester) AddUser() error {
	if !lt.started {
		return errors.New("LoadTester is not running")
	}
	activeUsers := len(lt.controllers)
	if activeUsers == lt.config.UsersConfiguration.MaxActiveUsers {
		return errors.New("Max active users limit reached")
	}
	controller := lt.newController(activeUsers+1, lt.status)
	lt.wg.Add(1)
	go func() {
		controller.Run()
	}()
	lt.controllers = append(lt.controllers, controller)
	return nil
}

// RemoveUser decrements the number of concurrently active users
func (lt *LoadTester) RemoveUser() error {
	if !lt.started {
		return errors.New("LoadTester is not running")
	}
	activeUsers := len(lt.controllers)
	if activeUsers == 0 {
		return errors.New("No active users left")
	}
	controller := lt.controllers[activeUsers-1]
	controller.Stop()
	lt.controllers = lt.controllers[:activeUsers-1]
	return nil
}

// Run starts the execution of a new load-test
func (lt *LoadTester) Run() error {
	if lt.started {
		return errors.New("LoadTester is already running")
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

// Stop terminates the current load-test
func (lt *LoadTester) Stop() error {
	if !lt.started {
		return errors.New("LoadTester is not running")
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

// New creates and initializes a new LoadTester
func New(config *config.LoadTestConfig, nc NewController) *LoadTester {
	if config == nil {
		return nil
	}
	return &LoadTester{
		config:        config,
		status:        make(chan control.UserStatus, config.UsersConfiguration.MaxActiveUsers),
		newController: nc,
	}
}
