// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package loadtest

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-server/v5/mlog"
)

// LoadTester is a structure holding all the state needed to run a load-test.
type LoadTester struct {
	mut           sync.RWMutex
	controllers   []control.UserController
	config        *Config
	wg            sync.WaitGroup
	statusChan    chan control.UserStatus
	status        Status
	newController NewController
}

// NewController is a factory function that returns a new
// control.UserController given an id and a channel of control.UserStatus
// It is passed during LoadTester initialization to provide a way to create
// concrete UserController values from within the loadtest package without the
// need of those being passed from the upper layer (the user of this API).
type NewController func(int, chan<- control.UserStatus) (control.UserController, error)

func (lt *LoadTester) handleStatus(startedChan chan struct{}) {
	// Copy the channel to prevent race conditions.
	statusChan := lt.statusChan
	close(startedChan)
	for st := range statusChan {
		if st.Code == control.USER_STATUS_STOPPED || st.Code == control.USER_STATUS_FAILED {
			lt.wg.Done()
		}
		if st.Code == control.USER_STATUS_ERROR {
			mlog.Info(st.Err.Error(), mlog.Int("controller_id", st.ControllerId), mlog.String("user_id", st.User.Store().Id()))
			atomic.AddInt64(&lt.status.NumErrors, 1)
			continue
		} else if st.Code == control.USER_STATUS_FAILED {
			mlog.Error(st.Err.Error())
			continue
		}
		mlog.Info(st.Info, mlog.Int("controller_id", st.ControllerId), mlog.String("user_id", st.User.Store().Id()))
	}
}

// AddUser increments by one the number of concurrently active users.
func (lt *LoadTester) AddUser() error {
	lt.mut.Lock()
	defer lt.mut.Unlock()

	if lt.status.State != Running {
		return ErrNotRunning
	}
	return lt.addUser()
}

// addUser is an internal API called from Run and AddUser both.
// DO NOT call this by itself, because this method is not protected by a mutex.
func (lt *LoadTester) addUser() error {
	activeUsers := len(lt.controllers)
	if activeUsers == lt.config.UsersConfiguration.MaxActiveUsers {
		return ErrMaxUsersReached
	}
	lt.status.NumUsers++
	lt.status.NumUsersAdded++
	userId := activeUsers + 1
	// If specified by the config, we randomly pick an existing user again,
	// to simulate multiple sessions.
	if activeUsers != 0 && rand.Int()%lt.config.UsersConfiguration.AvgSessionsPerUser != 0 {
		userID = rand.Intn(activeUsers)
	}
	controller, err := lt.newController(userID, lt.statusChan)
	if err != nil {
		return fmt.Errorf("could not create controller: %w", err)
	}
	if err := controller.SetRate(lt.config.UserControllerConfiguration.Rate); err != nil {
		return fmt.Errorf("loadtest: failed to set controller rate %w", err)
	}
	lt.wg.Add(1)
	go func() {
		controller.Run()
	}()
	lt.controllers = append(lt.controllers, controller)
	return nil
}

// RemoveUser decrements by one the number of concurrently active users.
func (lt *LoadTester) RemoveUser() error {
	lt.mut.Lock()
	defer lt.mut.Unlock()

	if lt.status.State != Running {
		return ErrNotRunning
	}
	return lt.removeUser()
}

// removeUser is an internal API called from Stop and RemoveUser both.
// DO NOT call this by itself, because this method is not protected by a mutex.
func (lt *LoadTester) removeUser() error {
	activeUsers := len(lt.controllers)
	if activeUsers == 0 {
		return ErrNoUsersLeft
	}
	lt.status.NumUsers--
	lt.status.NumUsersRemoved++
	// TODO: Add a way to make how a user is removed decidable from the upper layer (the user of this API),
	// for example by passing a typed constant (e.g. random, first, last).
	controller := lt.controllers[activeUsers-1]
	controller.Stop()
	lt.controllers = lt.controllers[:activeUsers-1]
	return nil
}

// Run starts the execution of a new load-test.
// It returns an error if called again without stopping the test first.
func (lt *LoadTester) Run() error {
	lt.mut.Lock()
	defer lt.mut.Unlock()

	if lt.status.State != Stopped {
		return ErrNotStopped
	}
	lt.status.State = Starting
	lt.status.NumUsersRemoved = 0
	lt.status.NumUsersAdded = 0
	lt.status.NumErrors = 0
	lt.status.StartTime = time.Now()
	lt.statusChan = make(chan control.UserStatus, lt.config.UsersConfiguration.MaxActiveUsers)
	startedChan := make(chan struct{})
	go lt.handleStatus(startedChan)
	<-startedChan
	for i := 0; i < lt.config.UsersConfiguration.InitialActiveUsers; i++ {
		if err := lt.addUser(); err != nil {
			mlog.Error(err.Error())
		}
	}
	lt.status.State = Running
	return nil
}

// Stop terminates the current load-test.
// It returns an error if it is called when the load test has not started.
func (lt *LoadTester) Stop() error {
	lt.mut.Lock()
	defer lt.mut.Unlock()

	if lt.status.State != Running {
		return ErrNotRunning
	}
	lt.status.State = Stopping
	for range lt.controllers {
		if err := lt.removeUser(); err != nil {
			mlog.Error(err.Error())
		}
	}
	lt.wg.Wait()
	close(lt.statusChan)
	lt.status.NumUsers = 0
	lt.status.State = Stopped
	return nil
}

// Status returns information regarding the current state of the load-test.
func (lt *LoadTester) Status() *Status {
	lt.mut.RLock()
	defer lt.mut.RUnlock()
	// We need to construct the struct anew because
	// NumErrors gets incremented in a separate goroutine.
	numErrors := atomic.LoadInt64(&lt.status.NumErrors)

	return &Status{
		State:           lt.status.State,
		NumUsers:        lt.status.NumUsers,
		NumUsersAdded:   lt.status.NumUsersAdded,
		NumUsersRemoved: lt.status.NumUsersRemoved,
		NumErrors:       numErrors,
		StartTime:       lt.status.StartTime,
	}
}

// New creates and initializes a new LoadTester with given config. A factory
// function is also given to enable the creation of UserController values from within the
// loadtest package.
func New(config *Config, nc NewController) (*LoadTester, error) {
	if config == nil || nc == nil {
		return nil, errors.New("nil params passed")
	}

	if err := config.IsValid(); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}

	return &LoadTester{
		config:        config,
		statusChan:    make(chan control.UserStatus, config.UsersConfiguration.MaxActiveUsers),
		newController: nc,
		status:        Status{},
	}, nil
}
