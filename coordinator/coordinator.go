// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"fmt"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance"
	"github.com/mattermost/mattermost-load-test-ng/defaults"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

// Coordinator is the object used to coordinate a cluster of
// load-test agents.
type Coordinator struct {
	mut      sync.RWMutex
	stopChan chan struct{}
	doneChan chan struct{}
	status   Status
	config   *Config
	cluster  *cluster.LoadAgentCluster
	monitor  *performance.Monitor
}

// Run starts a cluster of load-test agents.
// It returns a channel to signal when the coordinator is done.
// It is not safe to call this again after a call to Stop().
func (c *Coordinator) Run() (<-chan struct{}, error) {
	c.mut.Lock()
	defer c.mut.Unlock()

	if c.status.State == Done {
		return nil, ErrAlreadyDone
	} else if c.status.State != Stopped {
		return nil, ErrNotStopped
	}

	mlog.Info("coordinator: ready to drive a cluster of load-test agents", mlog.Int("num_agents", len(c.config.ClusterConfig.Agents)))

	if err := c.cluster.Run(); err != nil {
		mlog.Error("coordinator: running cluster failed", mlog.Err(err))
		return nil, err
	}

	monitorChan := c.monitor.Run()

	var lastActionTime, lastAlertTime time.Time
	var supportedUsers int

	// For now we are keeping all these values constant but in the future they
	// might change based on the state of the feedback loop.
	// Ideally we want the value of users to increment/decrement to react
	// to the speed at which metrics are changing.

	// The value of users to be incremented at each iteration.
	// TODO: It should be proportional to the maximum number of users expected to test.
	incValue := c.config.NumUsersInc
	// The value of users to be decremented at each iteration.
	// TODO: It should be proportional to the maximum number of users expected to test.
	decValue := c.config.NumUsersDec
	// The timespan to wait after a performance degradation alert before
	// incrementing or decrementing users again.
	restTime := time.Duration(c.config.RestTimeSec) * time.Second

	go func() {
		defer func() {
			c.monitor.Stop()
			c.cluster.Shutdown()
			close(c.doneChan)
			c.mut.Lock()
			c.status.State = Done
			c.mut.Unlock()
		}()

		for {
			var perfStatus performance.Status

			select {
			case <-c.stopChan:
				mlog.Info("coordinator: shutting down")
				return
			case perfStatus = <-monitorChan:
			}

			if perfStatus.Alert {
				lastAlertTime = time.Now()
			}

			status := c.cluster.Status()
			mlog.Info("coordinator: cluster status:", mlog.Int("active_users", status.ActiveUsers), mlog.Int64("errors", status.NumErrors))

			// TODO: supportedUsers should be estimated in a more clever way in the future.
			// For now we say that the supported number of users is the number of active users that ran
			// for the defined timespan without causing any performance degradation alert.
			if !lastAlertTime.IsZero() && !perfStatus.Alert && hasPassed(lastAlertTime, restTime) && hasPassed(lastActionTime, restTime) {
				supportedUsers = status.ActiveUsers
			}

			mlog.Info("coordinator: supported users", mlog.Int("supported_users", supportedUsers))

			// We give the feedback loop some rest time in case of performance
			// degradation alerts. We want metrics to stabilize before incrementing/decrementing users again.
			if lastAlertTime.IsZero() || lastActionTime.IsZero() || hasPassed(lastActionTime, restTime) {
				if perfStatus.Alert {
					mlog.Info("coordinator: decrementing active users", mlog.Int("num_users", decValue))
					if err := c.cluster.DecrementUsers(decValue); err != nil {
						mlog.Error("coordinator: failed to decrement users", mlog.Err(err))
					} else {
						lastActionTime = time.Now()
					}
				} else if lastAlertTime.IsZero() || hasPassed(lastAlertTime, restTime) {
					if status.ActiveUsers < c.config.ClusterConfig.MaxActiveUsers {
						inc := min(incValue, c.config.ClusterConfig.MaxActiveUsers-status.ActiveUsers)
						mlog.Info("coordinator: incrementing active users", mlog.Int("num_users", inc))
						if err := c.cluster.IncrementUsers(inc); err != nil {
							mlog.Error("coordinator: failed to increment users", mlog.Err(err))
						} else {
							lastActionTime = time.Now()
						}
					}
				}
			} else {
				mlog.Info("coordinator: waiting for metrics to stabilize")
			}
		}
	}()

	c.status.StartTime = time.Now()
	c.status.State = Running

	return c.doneChan, nil
}

// Stop stops the coordinator.
// It returns an error if the coordinator was not running.
func (c *Coordinator) Stop() error {
	c.mut.Lock()
	defer c.mut.Unlock()
	if c.status.State != Running {
		return ErrNotRunning
	}
	close(c.stopChan)
	<-c.doneChan
	c.status.State = Done
	return nil
}

// Status returns the coordinator's status.
func (c *Coordinator) Status() Status {
	c.mut.RLock()
	defer c.mut.RUnlock()
	clusterStatus := c.cluster.Status()
	return Status{
		State:       c.status.State,
		StartTime:   c.status.StartTime,
		ActiveUsers: clusterStatus.ActiveUsers,
		NumErrors:   clusterStatus.NumErrors,
	}
}

// New creates and initializes a new Coordinator for the given config.
// An error is returned if the initialization fails.
func New(config *Config) (*Coordinator, error) {
	if config == nil {
		return nil, fmt.Errorf("coordinator: config should not be nil")
	}
	if err := defaults.Validate(config); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}

	cluster, err := cluster.New(config.ClusterConfig)
	if err != nil {
		return nil, fmt.Errorf("coordinator: failed to create cluster: %w", err)
	}

	monitor, err := performance.NewMonitor(config.MonitorConfig)
	if err != nil {
		return nil, fmt.Errorf("coordinator: failed to create performance monitor: %w", err)
	}

	return &Coordinator{
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
		config:   config,
		cluster:  cluster,
		monitor:  monitor,
	}, nil
}
