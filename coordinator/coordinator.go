// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
)

// Coordinator is the object used to coordinate a cluster of
// load-test agents.
type Coordinator struct {
	config  *CoordinatorConfig
	cluster *cluster.LoadAgentCluster
	// monitor *Monitor
}

// Run starts a cluster of load-test agents.
func (c *Coordinator) Run() error {
	fmt.Printf("coordinator: ready to drive a cluster of %d load-test agents\n", len(c.config.ClusterConfig.Agents))

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	if err := c.cluster.Run(); err != nil {
		fmt.Printf("coordinator: running cluster failed \n%s\n", err.Error())
		c.cluster.Shutdown()
		return err
	}

	for {
		status := c.cluster.Status()
		fmt.Printf("coordinator: cluster status: %d active users, %d errors\n", status.ActiveUsers, status.NumErrors)

		if status.ActiveUsers < c.config.ClusterConfig.MaxActiveUsers {
			// TODO: make the choice of this value a bit smarter.
			inc := 10
			diff := c.config.ClusterConfig.MaxActiveUsers - status.ActiveUsers
			if diff < inc {
				inc = diff
			}
			fmt.Printf("coordinator: incrementing active users by %d\n", inc)
			err := c.cluster.IncrementUsers(inc)
			if err != nil {
				fmt.Println(err.Error())
			}
		}

		select {
		case <-interruptChannel:
			fmt.Printf("coordinator: shutting down\n")
			c.cluster.Shutdown()
			return nil
		case <-time.After(1 * time.Second):
		}

		// TODO: implement performance monitoring and act on them to complete feedback loop.
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
		return nil, fmt.Errorf("coordinator: failed to create cluster \n%w", err)
	}
	return &Coordinator{
		config:  config,
		cluster: cluster,
	}, nil
}
