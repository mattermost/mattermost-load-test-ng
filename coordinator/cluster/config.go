// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package cluster

import (
	"fmt"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/agent"
)

// LoadAgentClusterConfig holds information regarding the cluster of load-test
// agents.
type LoadAgentClusterConfig struct {
	// Agents is a list of the load-test agents API endpoints to be used during
	// the load-test. It's length defines the number of load-test instances
	// used during a load-test.
	Agents []agent.LoadAgentConfig
	// MaxActiveUsers defines the upper limit of concurrently active users to run across
	// the whole cluster.
	MaxActiveUsers int
}

// IsValid checks whether a LoadAgentClusterConfig is valid or not.
// Returns an error if the validation fails.
func (c LoadAgentClusterConfig) IsValid() error {
	if len(c.Agents) == 0 {
		return fmt.Errorf("no agents configured: at least one agent should be provided")
	}
	var maxUsers int
	for _, ac := range c.Agents {
		if err := ac.IsValid(); err != nil {
			return fmt.Errorf("agent config validation failed: %w", err)
		}
		maxUsers += ac.LoadTestConfig.UsersConfiguration.MaxActiveUsers
	}

	if c.MaxActiveUsers == 0 {
		return fmt.Errorf("MaxActiveUsers should be > 0")
	}

	if c.MaxActiveUsers > maxUsers {
		return fmt.Errorf("MaxActiveUsers should be less or equal to the sum of active users supported by the agents in the cluster")
	}
	return nil
}
