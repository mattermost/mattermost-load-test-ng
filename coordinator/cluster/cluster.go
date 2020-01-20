// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package cluster

import (
	"fmt"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/agent"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

// LoadAgentCluster is the object holding information about all the load-test
// agents available in the cluster.
type LoadAgentCluster struct {
	config LoadAgentClusterConfig
	agents []*agent.LoadAgent
}

// New creates and initializes a new LoadAgentCluster for the given config.
// An error is returned if the initialization fails.
func New(config LoadAgentClusterConfig) (*LoadAgentCluster, error) {
	if ok, err := config.IsValid(); !ok {
		return nil, err
	}
	agents := make([]*agent.LoadAgent, len(config.Agents))
	for i := 0; i < len(agents); i++ {
		agent, err := agent.New(config.Agents[i])
		if err != nil {
			return nil, fmt.Errorf("cluster: failed to create agent: %w", err)
		}
		agents[i] = agent
	}
	return &LoadAgentCluster{
		agents: agents,
		config: config,
	}, nil
}

// Run starts all the load-test agents available in the cluster.
func (c *LoadAgentCluster) Run() error {
	for _, agent := range c.agents {
		err := agent.Start()
		if err != nil {
			return fmt.Errorf("cluster: failed to start agent: %w", err)
		}
	}
	return nil
}

// Stop stops all the load-test agents available in the cluster.
func (c *LoadAgentCluster) Stop() error {
	for _, agent := range c.agents {
		err := agent.Stop()
		if err != nil {
			return fmt.Errorf("cluster: failed to stop agent: %w", err)
		}
	}
	return nil
}

// Shutdown stops all the load-test agents available in the cluster.
// It differs from Stop() as it won't return early in case of an error.
// It makes sure agent.Stop() is called once for every agent in the cluster.
func (c *LoadAgentCluster) Shutdown() {
	for _, agent := range c.agents {
		if err := agent.Stop(); err != nil {
			mlog.Error("cluster: failed to stop agent", mlog.Err(err))
		}
	}
}

// IncrementUsers increments the total number of active users in the load-test
// custer by the provided amount.
func (c *LoadAgentCluster) IncrementUsers(n int) error {
	if len(c.agents) == 0 {
		return nil
	}
	// TODO: Make this smarter. Implement an algorithm to make sure users are
	// distributed evenly across the agents regardless of the input value and number of
	// agents available.
	inc := int(n / len(c.agents))
	for i, agent := range c.agents {
		mlog.Info("cluster: adding users to agent", mlog.Int("num_users", inc), mlog.String("agent_id", c.config.Agents[i].Id))
		if err := agent.AddUsers(inc); err != nil {
			return fmt.Errorf("cluster: failed to add users to agent: %w", err)
		}
	}
	return nil
}

// DecrementUsers decrements the total number of active users in the load-test
// custer by the provided amount.
func (c *LoadAgentCluster) DecrementUsers(n int) error {
	return fmt.Errorf("cluster: not implemented")
}

// Status returns the current status of the LoadAgentCluster.
func (c *LoadAgentCluster) Status() Status {
	var status Status
	for _, agent := range c.agents {
		st := agent.Status()
		status.ActiveUsers += st.NumUsers
		status.NumErrors += st.NumErrors
	}
	return status
}
