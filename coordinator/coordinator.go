// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"

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
	log      *mlog.Logger
}

// Run starts a cluster of load-test agents.
// It returns a channel to signal when the coordinator is done.
// It returns an error if the coordinator is not in Stopped state.
func (c *Coordinator) Run() (<-chan struct{}, error) {
	c.mut.Lock()
	defer c.mut.Unlock()

	if c.status.State == Done {
		return nil, ErrAlreadyDone
	} else if c.status.State != Stopped {
		return nil, ErrNotStopped
	}

	c.log.Info("coordinator: ready to drive a cluster of load-test agents", mlog.Int("num_agents", len(c.config.ClusterConfig.Agents)))

	if err := c.cluster.Run(); err != nil {
		c.log.Error("coordinator: running cluster failed", mlog.Err(err))
		return nil, err
	}

	monitorChan := c.monitor.Run()

	var lastActionTime, lastAlertTime time.Time

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

	// TODO: considering making the following values configurable.

	// The threshold at which we consider the load-test done and we are ready to
	// give an answer. The value represents the slope of the best fit line for
	// the gathered samples. This value approaching zero means we have found
	// an equilibrium point.
	stopThreshold := 0.1
	// The timespan to consider when calculating the best fit line. A higher
	// value means considering a higher number of samples which improves the precision of
	// the final result.
	samplesTimeRange := 30 * time.Minute

	go func() {
		var supported int

		defer func() {
			c.monitor.Stop()
			clusterStatus, err := c.cluster.Status()
			if err != nil {
				c.log.Error("coordinator: cluster status error:", mlog.Err(err))
			}
			c.cluster.Shutdown()
			close(c.doneChan)
			c.mut.Lock()
			c.status.State = Done
			c.status.SupportedUsers = supported
			c.status.StopTime = time.Now()
			if clusterStatus.NumErrors > 0 {
				c.status.NumErrors = clusterStatus.NumErrors
			}
			c.mut.Unlock()
		}()

		var samples []point

		for {
			var perfStatus performance.Status

			select {
			case <-c.stopChan:
				c.log.Info("coordinator: shutting down")
				return
			case perfStatus = <-monitorChan:
			}

			if perfStatus.Alert {
				lastAlertTime = time.Now()
			}

			status, err := c.cluster.Status()
			if err != nil {
				c.log.Error("coordinator: cluster status error:", mlog.Err(err))
				continue
			}
			c.log.Info("coordinator: cluster status:", mlog.Int("active_users", status.ActiveUsers), mlog.Int64("errors", status.NumErrors))

			if !lastAlertTime.IsZero() {
				samples = append(samples, point{
					x: time.Now(),
					y: status.ActiveUsers,
				})
				latest := getLatestSamples(samples, samplesTimeRange)
				if len(latest) > 0 && len(latest) < len(samples) && math.Abs(slope(latest)) < stopThreshold {
					c.log.Info("coordinator done!")
					supported = int(math.Round(avg(latest)))
					c.log.Info(fmt.Sprintf("estimated number of supported users is %d", supported))
					return
				}
				// We replace older samples which are not needed anymore.
				if len(samples) >= 2*len(latest) {
					copy(samples, latest)
					samples = samples[:len(latest)]
				}
			}

			// We give the feedback loop some rest time in case of performance
			// degradation alerts. We want metrics to stabilize before incrementing/decrementing users again.
			if lastAlertTime.IsZero() || lastActionTime.IsZero() || hasPassed(lastActionTime, restTime) {
				if perfStatus.Alert {
					c.log.Info("coordinator: decrementing active users", mlog.Int("num_users", decValue))
					if err := c.cluster.DecrementUsers(decValue); err != nil {
						c.log.Error("coordinator: failed to decrement users", mlog.Err(err))
					} else {
						lastActionTime = time.Now()
					}
				} else if lastAlertTime.IsZero() || hasPassed(lastAlertTime, restTime) {
					if status.ActiveUsers < c.config.ClusterConfig.MaxActiveUsers {
						inc := min(incValue, c.config.ClusterConfig.MaxActiveUsers-status.ActiveUsers)
						c.log.Info("coordinator: incrementing active users", mlog.Int("num_users", inc))
						if err := c.cluster.IncrementUsers(inc); err != nil {
							c.log.Error("coordinator: failed to increment users", mlog.Err(err))
						} else {
							lastActionTime = time.Now()
						}
					} else if lastAlertTime.IsZero() || hasPassed(lastAlertTime, restTime) {
						if status.ActiveUsers < c.config.ClusterConfig.MaxActiveUsers {
							inc := min(incValue, c.config.ClusterConfig.MaxActiveUsers-status.ActiveUsers)
							c.log.Info("coordinator: incrementing active users", mlog.Int("num_users", inc))
							if err := c.cluster.IncrementUsers(inc); err != nil {
								c.log.Error("coordinator: failed to increment users", mlog.Err(err))
							} else {
								lastActionTime = time.Now()
							}
						}
					}
				} else {
					c.log.Info("coordinator: waiting for metrics to stabilize")
				}
			}
		}
	}()

	c.status.StartTime = time.Now()
	c.status.State = Running

	return c.doneChan, nil
}

// Stop stops the coordinator.
// It returns an error in case of failure.
func (c *Coordinator) Stop() error {
	c.mut.Lock()
	defer c.mut.Unlock()
	if c.status.State != Running {
		return ErrNotRunning
	}
	clusterStatus, err := c.cluster.Status()
	if err != nil {
		return fmt.Errorf("coordinator: failed to get cluster status: %w", err)
	}
	close(c.stopChan)
	<-c.doneChan
	c.status = Status{
		State:          Done,
		StartTime:      c.status.StartTime,
		StopTime:       time.Now(),
		ActiveUsers:    clusterStatus.ActiveUsers,
		NumErrors:      clusterStatus.NumErrors,
		SupportedUsers: c.status.SupportedUsers,
	}
	return nil
}

// Status returns the coordinator's status.
func (c *Coordinator) Status() (Status, error) {
	c.mut.RLock()
	defer c.mut.RUnlock()
	if c.status.State != Running {
		return c.status, nil
	}
	clusterStatus, err := c.cluster.Status()
	if err != nil {
		return Status{}, fmt.Errorf("coordinator: failed to get cluster status: %w", err)
	}
	return Status{
		State:          c.status.State,
		StartTime:      c.status.StartTime,
		StopTime:       c.status.StopTime,
		ActiveUsers:    clusterStatus.ActiveUsers,
		NumErrors:      clusterStatus.NumErrors,
		SupportedUsers: c.status.SupportedUsers,
	}, nil
}

// New creates and initializes a new Coordinator for the given config.
// The ltConfig parameter is used to create and configure load-test agents.
// An error is returned if the initialization fails.
func New(config *Config, ltConfig loadtest.Config, log *mlog.Logger) (*Coordinator, error) {
	if config == nil {
		return nil, errors.New("coordinator: config should not be nil")
	}
	if log == nil {
		return nil, errors.New("coordinator: logger should not be nil")
	}
	if err := defaults.Validate(config); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}

	cluster, err := cluster.New(config.ClusterConfig, ltConfig, log)
	if err != nil {
		return nil, fmt.Errorf("coordinator: failed to create cluster: %w", err)
	}

	monitor, err := performance.NewMonitor(config.MonitorConfig, log)
	if err != nil {
		return nil, fmt.Errorf("coordinator: failed to create performance monitor: %w", err)
	}

	return &Coordinator{
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
		config:   config,
		cluster:  cluster,
		monitor:  monitor,
		log:      log,
	}, nil
}
