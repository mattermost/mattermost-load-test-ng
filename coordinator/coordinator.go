// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

// Coordinator is the object used to coordinate a cluster of
// load-test agents.
type Coordinator struct {
	config  *CoordinatorConfig
	cluster *cluster.LoadAgentCluster
	monitor *performance.Monitor
}

// Run starts a cluster of load-test agents.
func (c *Coordinator) Run() error {
	mlog.Info("coordinator: ready to drive a cluster of load-test agents", mlog.Int("num_agents", len(c.config.ClusterConfig.Agents)))

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	if err := c.cluster.Run(); err != nil {
		mlog.Error("coordinator: running cluster failed", mlog.Err(err))
		c.cluster.Shutdown()
		return err
	}
	defer c.cluster.Shutdown()

	monitorChan, err := c.monitor.Run()
	if err != nil {
		mlog.Error("coordinator: running monitor failed", mlog.Err(err))
		return err
	}
	defer c.monitor.Stop()

	perfStatus := <-monitorChan

	for {
		status := c.cluster.Status()
		mlog.Info("coordinator: cluster status:", mlog.Int("active_users", status.ActiveUsers), mlog.Int64("errors", status.NumErrors))

		if !perfStatus.Alert {
			if status.ActiveUsers < c.config.ClusterConfig.MaxActiveUsers {
				// TODO: make the choice of this value a bit smarter.
				inc := 8
				diff := c.config.ClusterConfig.MaxActiveUsers - status.ActiveUsers
				if diff < inc {
					inc = diff
				}
				mlog.Info("coordinator: incrementing active users", mlog.Int("num_users", inc))
				err := c.cluster.IncrementUsers(inc)
				if err != nil {
					mlog.Error("coordinator: failed to increment users", mlog.Err(err))
				}
			}
		} else {
			mlog.Info("coordinator: performance degradation alert")
			dec := 4
			err := c.cluster.DecrementUsers(dec)
			if err != nil {
				mlog.Error("coordinator: failed to decrement users", mlog.Err(err))
			}
		}

		select {
		case <-interruptChannel:
			mlog.Info("coordinator: shutting down")
			return nil
		case perfStatus = <-monitorChan:
		}
	}
}

// New creates and initializes a new Coordinator for the given config.
// An error is returned if the initialization fails.
func New(config *CoordinatorConfig) (*Coordinator, error) {
	if config == nil {
		return nil, fmt.Errorf("coordinator: config should not be nil")
	}
	if ok, err := config.IsValid(); !ok {
		return nil, err
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
		config:  config,
		cluster: cluster,
		monitor: monitor,
	}, nil
}
