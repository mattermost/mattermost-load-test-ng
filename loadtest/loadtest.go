// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package loadtest

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/wiggin77/merror"

	"github.com/mattermost/mattermost-server/server/v8/platform/shared/mlog"
)

const (
	connFactor = 4
)

// LoadTester is a structure holding all the state needed to run a load-test.
type LoadTester struct {
	mut           sync.RWMutex
	config        *Config
	wg            sync.WaitGroup
	statusChan    chan control.UserStatus
	status        Status
	newController NewController

	activeControllers []control.UserController
	idleControllers   []control.UserController

	log *mlog.Logger
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
			atomic.AddInt64(&lt.status.NumUsersStopped, 1)
			lt.wg.Done()
		}

		switch st.Code {
		case control.USER_STATUS_ERROR:
			lt.log.Error(st.Err.Error(), mlog.Int("controller_id", st.ControllerId), mlog.String("user_id", st.User.Store().Id()))
			atomic.AddInt64(&lt.status.NumErrors, 1)
		case control.USER_STATUS_FAILED:
			lt.log.Error(st.Err.Error())
		case control.USER_STATUS_WARN:
			lt.log.Warn(st.Warn)
		default:
			lt.log.Info(st.Info, mlog.Int("controller_id", st.ControllerId), mlog.String("user_id", st.User.Store().Id()))
		}
	}
}

// AddUsers attempts to increment by numUsers the number of concurrently active users.
// Returns the number of users successfully added.
func (lt *LoadTester) AddUsers(numUsers int) (int, error) {
	lt.mut.Lock()
	defer lt.mut.Unlock()
	if numUsers <= 0 {
		return 0, ErrInvalidNumUsers
	}
	if lt.status.State != Running {
		return 0, ErrNotRunning
	}
	for i := 0; i < numUsers; i++ {
		if err := lt.addUser(); err != nil {
			return i, err
		}
	}
	return numUsers, nil
}

// addUser is an internal API called from Run and AddUsers both.
// DO NOT call this by itself, because this method is not protected by a mutex.
func (lt *LoadTester) addUser() error {
	activeUsers := len(lt.activeControllers)
	if activeUsers == lt.config.UsersConfiguration.MaxActiveUsers {
		return ErrMaxUsersReached
	}

	var controller control.UserController
	if len(lt.idleControllers) > 0 {
		controller = lt.idleControllers[0]
		lt.idleControllers = lt.idleControllers[1:]
	} else {
		userId := activeUsers + 1
		// If specified by the config, we randomly pick an existing user again,
		// to simulate multiple sessions.
		if activeUsers != 0 && rand.Int()%lt.config.UsersConfiguration.AvgSessionsPerUser != 0 {
			userId = rand.Intn(activeUsers)
		}
		var err error
		controller, err = lt.newController(userId, lt.statusChan)
		if err != nil {
			return err
		}
	}

	rate, err := pickRate(lt.config.UserControllerConfiguration)
	if err != nil {
		return fmt.Errorf("loadtest: failed to pick rate: %w", err)
	}
	if err := controller.SetRate(rate); err != nil {
		return fmt.Errorf("loadtest: failed to set controller rate %w", err)
	}

	lt.status.NumUsers++
	lt.status.NumUsersAdded++
	lt.activeControllers = append(lt.activeControllers, controller)

	lt.wg.Add(1)
	go func() {
		controller.Run()
	}()
	return nil
}

// RemoveUsers attempts to decrement by numUsers the number of concurrently active users.
// Returns the number of users successfully removed.
func (lt *LoadTester) RemoveUsers(numUsers int) (int, error) {
	lt.mut.Lock()
	defer lt.mut.Unlock()
	if numUsers <= 0 {
		return 0, ErrInvalidNumUsers
	}
	if lt.status.State != Running {
		return 0, ErrNotRunning
	}
	return lt.removeUsers(numUsers)
}

// removeUsers is an internal API called from Stop and RemoveUsers both.
// DO NOT call this by itself, because this method is not protected by a mutex.
func (lt *LoadTester) removeUsers(numUsers int) (int, error) {
	activeUsers := len(lt.activeControllers)

	var err error
	if numUsers > activeUsers {
		numUsers = activeUsers
		err = ErrNoUsersLeft
	}

	var wg sync.WaitGroup
	wg.Add(numUsers)
	// TODO: Add a way to make how a user is removed decidable from the upper layer (the user of this API),
	// for example by passing a typed constant (e.g. random, first, last).
	for i := 0; i < numUsers; i++ {
		go func(i int) {
			defer wg.Done()
			lt.activeControllers[activeUsers-1-i].Stop()
		}(i)
	}
	wg.Wait()

	lt.idleControllers = append(lt.idleControllers, lt.activeControllers[activeUsers-numUsers:]...)
	lt.activeControllers = lt.activeControllers[:activeUsers-numUsers]
	lt.status.NumUsers -= int64(numUsers)
	lt.status.NumUsersRemoved += int64(numUsers)

	return numUsers, err
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
	lt.status.NumUsersStopped = 0
	lt.status.NumErrors = 0
	lt.status.StartTime = time.Now()
	lt.statusChan = make(chan control.UserStatus, lt.config.UsersConfiguration.MaxActiveUsers)
	startedChan := make(chan struct{})
	go lt.handleStatus(startedChan)
	<-startedChan
	for i := 0; i < lt.config.UsersConfiguration.InitialActiveUsers; i++ {
		if err := lt.addUser(); err != nil {
			lt.log.Error(err.Error())
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

	if _, err := lt.removeUsers(len(lt.activeControllers)); err != nil {
		lt.log.Error(err.Error())
	}

	lt.wg.Wait()
	close(lt.statusChan)
	lt.idleControllers = make([]control.UserController, 0)
	lt.status.NumUsers = 0
	lt.status.State = Stopped
	return nil
}

// Status returns information regarding the current state of the load-test.
func (lt *LoadTester) Status() *Status {
	lt.mut.RLock()
	defer lt.mut.RUnlock()
	// We need to construct the struct anew because
	// NumErrors and NumUsersStopped get incremented in a separate goroutine.
	numErrors := atomic.LoadInt64(&lt.status.NumErrors)
	numStopped := atomic.LoadInt64(&lt.status.NumUsersStopped)

	return &Status{
		State:           lt.status.State,
		NumUsers:        lt.status.NumUsers,
		NumUsersAdded:   lt.status.NumUsersAdded,
		NumUsersRemoved: lt.status.NumUsersRemoved,
		NumUsersStopped: numStopped,
		NumErrors:       numErrors,
		StartTime:       lt.status.StartTime,
	}
}

func (lt *LoadTester) InjectAction(action string) error {
	lt.mut.RLock()
	defer lt.mut.RUnlock()

	if lt.status.State != Running {
		return ErrNotRunning
	}

	merr := merror.New()

	for _, ctrl := range lt.activeControllers {
		if err := ctrl.InjectAction(action); err != nil {
			merr.Append(err)
		}
	}
	return merr.ErrorOrNil()
}

// MaxHTTPConns returns the maximum number of HTTP connections to be used for the given number of users.
func MaxHTTPConns(maxUsers int) int {
	return maxUsers / connFactor
}

// New creates and initializes a new LoadTester with given config. A factory
// function is also given to enable the creation of UserController values from within the
// loadtest package.
func New(config *Config, nc NewController, log *mlog.Logger) (*LoadTester, error) {
	if config == nil || nc == nil || log == nil {
		return nil, errors.New("nil params passed")
	}

	if err := defaults.Validate(config); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}

	var rlimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		return nil, fmt.Errorf("could not get the Rlimit value: %w", err)
	}

	if maxActiveUsers := config.UsersConfiguration.MaxActiveUsers; uint64(
		maxActiveUsers+MaxHTTPConns(maxActiveUsers)) > rlimit.Max {
		return nil, fmt.Errorf("MaxActiveUsers is not compatible with max Rlimit value. "+
			"MaxActiveUsers = %d, max_Rlimit = %d. Suggested value is %d. "+
			"You can check how to raise this value in the documentation at: "+
			"https://github.com/mattermost/mattermost-load-test-ng/blob/master/docs/faq.md#agent-failing-with-maxactiveusers-is-not-compatible-with-max-rlimit-value-error",
			maxActiveUsers, rlimit.Max, nextPowerOf2(maxActiveUsers+MaxHTTPConns(maxActiveUsers)+1))
	}

	return &LoadTester{
		config:            config,
		statusChan:        make(chan control.UserStatus, config.UsersConfiguration.MaxActiveUsers),
		newController:     nc,
		status:            Status{},
		activeControllers: make([]control.UserController, 0),
		idleControllers:   make([]control.UserController, 0),
		log:               log,
	}, nil
}
