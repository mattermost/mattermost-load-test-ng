// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package cluster

import (
	"errors"
	"fmt"
	"sync"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/agent"
	"github.com/mattermost/mattermost-load-test-ng/defaults"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

// LoadAgentCluster is the object holding information about all the load-test
// agents available in the cluster.
type LoadAgentCluster struct {
	config LoadAgentClusterConfig
	agents []*agent.LoadAgent
	errMap map[*agent.LoadAgent]errorTrack
}

type errorTrack struct {
	lastError   int64
	totalErrors int64
}

// New creates and initializes a new LoadAgentCluster for the given config.
// An error is returned if the initialization fails.
func New(config LoadAgentClusterConfig) (*LoadAgentCluster, error) {
	if err := defaults.Validate(config); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}
	agents := make([]*agent.LoadAgent, len(config.Agents))
	errMap := make(map[*agent.LoadAgent]errorTrack)
	for i := 0; i < len(agents); i++ {
		agent, err := agent.New(config.Agents[i])
		if err != nil {
			return nil, fmt.Errorf("cluster: failed to create agent: %w", err)
		}
		agents[i] = agent
		errMap[agent] = errorTrack{}
	}

	return &LoadAgentCluster{
		agents: agents,
		config: config,
		errMap: errMap,
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
	var wg sync.WaitGroup
	wg.Add(len(c.agents))
	for _, ag := range c.agents {
		go func(ag *agent.LoadAgent) {
			defer wg.Done()
			if err := ag.Stop(); err != nil {
				mlog.Error("cluster: failed to stop agent", mlog.Err(err))
			}
		}(ag)
	}
	wg.Wait()
}

// IncrementUsers increments the total number of active users in the load-test
// custer by the provided amount.
func (c *LoadAgentCluster) IncrementUsers(n int) error {
	if len(c.agents) == 0 {
		return nil
	}

	dist, err := additionDistribution(c.agents, n)
	if err != nil {
		return fmt.Errorf("cluster: cannot add users to any agent: %w", err)
	}
	for i, inc := range dist {
		mlog.Info("cluster: adding users to agent", mlog.Int("num_users", inc), mlog.String("agent_id", c.config.Agents[i].Id))

		if err := c.agents[i].AddUsers(inc); err != nil {
			// Most probably the agent restarted, so we just start the agent again.
			if errors.Is(err, agent.ErrAgentNotFound) {
				if err := c.agents[i].Start(); err != nil {
					mlog.Error("agent restart failed", mlog.Err(err))
				}
				continue
			}
			return fmt.Errorf("cluster: failed to add users to agent: %w", err)
		}
	}
	return nil
}

// DecrementUsers decrements the total number of active users in the load-test
// custer by the provided amount.
func (c *LoadAgentCluster) DecrementUsers(n int) error {
	if len(c.agents) == 0 {
		return nil
	}

	dist, err := deletionDistribution(c.agents, n)
	if err != nil {
		return fmt.Errorf("cluster: cannot add users to any agent: %w", err)
	}
	for i, dec := range dist {
		mlog.Info("cluster: removing users from agent", mlog.Int("num_users", dec), mlog.String("agent_id", c.config.Agents[i].Id))
		if err := c.agents[i].RemoveUsers(dec); err != nil {
			// Most probably the agent restarted, so we just start the agent again.
			if errors.Is(err, agent.ErrAgentNotFound) {
				if err := c.agents[i].Start(); err != nil {
					mlog.Error("agent restart failed", mlog.Err(err))
				}
				continue
			}
			return fmt.Errorf("cluster: failed to remove users from agent: %w", err)
		}
	}
	return nil
}

// Status returns the current status of the LoadAgentCluster.
func (c *LoadAgentCluster) Status() Status {
	var status Status
	for _, agent := range c.agents {
		st := agent.Status()
		status.ActiveUsers += st.NumUsers
		currentError := st.NumErrors
		errInfo := c.errMap[agent]
		if currentError < errInfo.lastError {
			// crash
			// We increment the total accumulated errors by the
			// last error count.
			errInfo.totalErrors += errInfo.lastError
		}
		errInfo.lastError = currentError
		c.errMap[agent] = errInfo

		// Total errors = current errors + past accumulated errors from restarts.
		status.NumErrors += currentError + c.errMap[agent].totalErrors
	}
	return status
}
