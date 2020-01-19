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
// Returns an error if the validtation fails.
func (c LoadAgentClusterConfig) IsValid() (bool, error) {
	if len(c.Agents) == 0 {
		return false, fmt.Errorf("no agents configured. At least one agent should be provided.")
	}
	if c.MaxActiveUsers == 0 {
		return false, fmt.Errorf("MaxActiveUsers should be greated than 0.")
	}
	return true, nil
}
