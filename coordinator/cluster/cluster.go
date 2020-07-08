// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package cluster

import (
	"errors"
	"fmt"
	"sync"

	client "github.com/mattermost/mattermost-load-test-ng/api/client/agent"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"

	"github.com/mattermost/mattermost-server/v5/mlog"
)

// LoadAgentCluster is the object holding information about all the load-test
// agents available in the cluster.
type LoadAgentCluster struct {
	config   LoadAgentClusterConfig
	ltConfig loadtest.Config
	agents   []*client.Agent
	errMap   map[*client.Agent]errorTrack
	log      *mlog.Logger
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
	errMap := make(map[*client.Agent]errorTrack)
	for i := 0; i < len(agents); i++ {
		agent, err := client.New(config.Agents[i].Id, config.Agents[i].ApiURL, nil)
		if err != nil {
			return nil, fmt.Errorf("cluster: failed to create api client: %w", err)
		}
		agents[i] = agent
		errMap[agent] = errorTrack{}

		// We check if the agent has already been created.
		if _, err := agent.Status(); err == nil {
			continue
		}

		if err := createAgent(agent, ltConfig); err != nil {
			return nil, err
		}
	}

	return &LoadAgentCluster{
		agents:   agents,
		config:   config,
		ltConfig: ltConfig,
		errMap:   errMap,
		log:      log,
	}, nil
}

// Run starts all the load-test agents available in the cluster.
func (c *LoadAgentCluster) Run() error {
	for _, agent := range c.agents {
		if _, err := agent.Run(); err != nil {
			return fmt.Errorf("cluster: failed to start agent: %w", err)
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
	return nil
}

// Shutdown stops and destroys all the load-test agents available in the cluster.
// It makes sure agent.Destroy() is called once for every agent in the cluster.
func (c *LoadAgentCluster) Shutdown() {
	var wg sync.WaitGroup
	wg.Add(len(c.agents))
	for _, ag := range c.agents {
		go func(ag *client.Agent) {
			defer wg.Done()
			if _, err := ag.Destroy(); err != nil {
				c.log.Error("cluster: failed to stop agent", mlog.Err(err))
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

	amounts, err := getUsersAmounts(c.agents)
	if err != nil {
		return err
	}
	dist, err := additionDistribution(amounts, n)
	if err != nil {
		return fmt.Errorf("cluster: cannot add users to any agent: %w", err)
	}
	for i, inc := range dist {
		c.log.Info("cluster: adding users to agent", mlog.Int("num_users", inc), mlog.String("agent_id", c.config.Agents[i].Id))
		if _, err := c.agents[i].AddUsers(inc); err != nil {
			// Most probably the agent crashed, so we just start it again.
			if _, err := c.agents[i].Run(); err != nil {
				c.log.Error("agent restart failed", mlog.Err(err))
				return fmt.Errorf("cluster: failed to add users to agent: %w", err)
			}
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
	return status, nil
}
