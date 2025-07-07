// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package cluster

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	client "github.com/mattermost/mattermost-load-test-ng/api/client/agent"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"
	"github.com/wiggin77/merror"

	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

// LoadAgentCluster is the object holding information about all the load-test
// agents available in the cluster.
type LoadAgentCluster struct {
	config        LoadAgentClusterConfig
	ltConfig      loadtest.Config
	agents        []*client.Agent
	browserAgents []*client.Agent
	errMap        map[*client.Agent]*errorTrack
	log           *mlog.Logger
}

type errorTrack struct {
	lastError   int64
	totalErrors int64
}

func createAgent(agent *client.Agent, ltConfig loadtest.Config) error {
	// TODO: UserController config should probably come from the upper layer
	// and be passed through.
	var ucConfig interface{}
	var err error
	switch ltConfig.UserControllerConfiguration.Type {
	case loadtest.UserControllerSimple:
		ucConfig, err = simplecontroller.ReadConfig("")
	case loadtest.UserControllerSimulative:
		ucConfig, err = simulcontroller.ReadConfig("")
	}
	if err != nil {
		return fmt.Errorf("cluster: failed to read controller config: %w", err)
	}

	if _, err := agent.Create(&ltConfig, ucConfig); err != nil {
		return fmt.Errorf("cluster: failed to create agent: %w", err)
	}

	return nil
}

// New creates and initializes a new LoadAgentCluster for the given config.
// An error is returned if the initialization fails.
func New(config LoadAgentClusterConfig, ltConfig loadtest.Config, log *mlog.Logger) (*LoadAgentCluster, error) {
	if log == nil {
		return nil, errors.New("logger should not be nil")
	}
	if err := defaults.Validate(config); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}
	agents := make([]*client.Agent, len(config.Agents))
	errMap := make(map[*client.Agent]*errorTrack)
	for i := range len(agents) {
		agent, err := client.New(config.Agents[i].Id, config.Agents[i].ApiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("cluster: failed to create api client for agent: %w", err)
		}
		agents[i] = agent
		errMap[agent] = &errorTrack{}

		// We check if the agent has already been created.
		if _, err := agent.Status(); err == nil {
			continue
		}

		if err := createAgent(agent, ltConfig); err != nil {
			return nil, err
		}
	}

	browserAgents := make([]*client.Agent, len(config.BrowserAgents))

	for i := range len(browserAgents) {
		// TODO: Check once again
		browserAgent, err := client.New(config.BrowserAgents[i].Id, config.BrowserAgents[i].ApiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("cluster: failed to create api client for browser agent: %w", err)
		}
		browserAgents[i] = browserAgent
		errMap[browserAgent] = &errorTrack{}

		if _, err := browserAgent.Status(); err == nil {
			continue
		}

		if err := createAgent(browserAgent, ltConfig); err != nil {
			return nil, err
		}
	}

	return &LoadAgentCluster{
		agents:        agents,
		browserAgents: browserAgents,
		config:        config,
		ltConfig:      ltConfig,
		errMap:        errMap,
		log:           log,
	}, nil
}

// Run starts all the load-test agents available in the cluster.
func (c *LoadAgentCluster) Run() error {
	for _, agent := range c.agents {
		if _, err := agent.Run(); err != nil {
			return fmt.Errorf("cluster: failed to start agent: %w", err)
		}
	}

	for _, browserAgent := range c.browserAgents {
		if _, err := browserAgent.Run(); err != nil {
			return fmt.Errorf("cluster: failed to start browser agent: %w", err)
		}
	}

	return nil
}

// Stop stops all the load-test agents available in the cluster.
func (c *LoadAgentCluster) Stop() error {
	for _, agent := range c.agents {
		if _, err := agent.Stop(); err != nil {
			return fmt.Errorf("cluster: failed to stop agent: %w", err)
		}
	}

	for _, browserAgent := range c.browserAgents {
		if _, err := browserAgent.Stop(); err != nil {
			return fmt.Errorf("cluster: failed to stop browser agent: %w", err)
		}
	}

	return nil
}

// Shutdown stops and destroys all the load-test agents available in the cluster.
// It makes sure agent.Destroy() is called once for every agent in the cluster.
func (c *LoadAgentCluster) Shutdown() {
	var wg sync.WaitGroup
	wg.Add(len(c.agents) + len(c.browserAgents))

	for _, ag := range c.agents {
		go func(ag *client.Agent) {
			defer wg.Done()
			if _, err := ag.Destroy(); err != nil {
				c.log.Error("cluster: failed to stop agent", mlog.Err(err))
			}
		}(ag)
	}

	for _, browserAgent := range c.browserAgents {
		if _, err := browserAgent.Destroy(); err != nil {
			c.log.Error("cluster: failed to stop browser agent", mlog.Err(err))
		}
	}

	wg.Wait()
}

// IncrementUsers increments the total number of active users in the load-test
// custer by the provided amount.
func (c *LoadAgentCluster) IncrementUsers(n int) error {
	if len(c.agents) == 0 {
		return nil
	}

	amounts, err := getUsersAmounts(c.agents)
	if err != nil {
		return err
	}
	dist, err := additionDistribution(amounts, n)
	if err != nil {
		return fmt.Errorf("cluster: cannot add users to any agent: %w", err)
	}

	// Additional logic to check how many users to add to
	// server agents, and how many to add in browser agents.

	for i, inc := range dist {
		c.log.Info("cluster: adding users to agent", mlog.Int("num_users", inc), mlog.String("agent_id", c.config.Agents[i].Id))
		if _, err := c.agents[i].AddUsers(inc); err != nil {
			c.log.Error("cluster: adding users failed", mlog.Err(err))
			// Most probably the agent crashed, so we just start it again.
			if _, err := c.agents[i].Run(); err != nil {
				c.log.Error("agent restart failed", mlog.Err(err))
				return fmt.Errorf("cluster: failed to add users to agent: %w", err)
			}
		}
	}

	// Additional loop here to add to browser agents.

	return nil
}

// DecrementUsers decrements the total number of active users in the load-test
// custer by the provided amount.
func (c *LoadAgentCluster) DecrementUsers(n int) error {
	if len(c.agents) == 0 {
		return nil
	}

	amounts, err := getUsersAmounts(c.agents)
	if err != nil {
		return err
	}
	dist, err := deletionDistribution(amounts, n)
	if err != nil {
		return fmt.Errorf("cluster: cannot add users to any agent: %w", err)
	}
	for i, dec := range dist {
		c.log.Info("cluster: removing users from agent", mlog.Int("num_users", dec), mlog.String("agent_id", c.config.Agents[i].Id))
		if _, err := c.agents[i].RemoveUsers(dec); err != nil {
			// Most probably the agent crashed, so we just start it again.
			if _, err := c.agents[i].Run(); err != nil {
				c.log.Error("agent restart failed", mlog.Err(err))
				return fmt.Errorf("cluster: failed to remove users from agent: %w", err)
			}
		}
	}
	return nil
}

// Status returns the current status of the LoadAgentCluster.
func (c *LoadAgentCluster) Status() (Status, error) {
	var status Status
	for _, agent := range c.agents {
		st, err := agent.Status()
		// Agent probably crashed. We create it again.
		if errors.Is(err, client.ErrAgentNotFound) {
			if err := createAgent(agent, c.ltConfig); err != nil {
				c.log.Error("agent create failed", mlog.Err(err))
			}
		} else if err != nil {
			c.log.Error("cluster: failed to get status for agent:", mlog.Err(err))
		}

		status.ActiveUsers += int(st.NumUsers)
		currentError := st.NumErrors
		errInfo := c.errMap[agent]
		lastError := atomic.LoadInt64(&errInfo.lastError)
		totalErrors := atomic.LoadInt64(&errInfo.totalErrors)
		if currentError < lastError {
			// crash
			// We increment the total accumulated errors by the
			// last error count.
			atomic.AddInt64(&errInfo.totalErrors, lastError)
			totalErrors += lastError
		}
		atomic.StoreInt64(&errInfo.lastError, currentError)

		// Total errors = current errors + past accumulated errors from restarts.
		status.NumErrors += currentError + totalErrors
	}

	for _, browserAgent := range c.browserAgents {
		st, err := browserAgent.Status()
		// Agent probably crashed. We create it again.
		if errors.Is(err, client.ErrAgentNotFound) {
			if err := createAgent(browserAgent, c.ltConfig); err != nil {
				c.log.Error("browser agent create failed", mlog.Err(err))
			}
		} else if err != nil {
			c.log.Error("cluster: failed to get status for browser agent:", mlog.Err(err))
		}

		status.ActiveBrowserUsers += int(st.NumUsers)
		currentError := st.NumErrors
		errInfo := c.errMap[browserAgent]
		lastError := atomic.LoadInt64(&errInfo.lastError)
		totalErrors := atomic.LoadInt64(&errInfo.totalErrors)
		if currentError < lastError {
			// crash
			atomic.AddInt64(&errInfo.totalErrors, lastError)
			totalErrors += lastError
		}
		atomic.StoreInt64(&errInfo.lastError, currentError)

		// Total errors = current errors + past accumulated errors from restarts.
		status.NumBrowserErrors += currentError + totalErrors
	}

	return status, nil
}

// InjectAction injects an action into all the agents. The action is run once,
// at the next possible opportunity.
func (c *LoadAgentCluster) InjectAction(actionID string) error {
	merr := merror.New()
	for _, agent := range c.agents {
		if _, err := agent.InjectAction(actionID); err != nil {
			merr.Append(fmt.Errorf("cluster: failed to inject action %s for agent %s: %w", actionID, agent.Id(), err))
		}
	}
	return merr.ErrorOrNil()
}
