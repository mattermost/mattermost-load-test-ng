// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package cluster

import (
	"github.com/mattermost/mattermost-load-test-ng/coordinator/agent"
)

// LoadAgentClusterConfig holds information regarding the cluster of load-test
// agents.
type LoadAgentClusterConfig struct {
	// Agents is a list of the load-test agents API endpoints to be used during
	// the load-test. It's length defines the number of load-test instances
	// used during a load-test.
	Agents []agent.LoadAgentConfig `default_size:"1"`
	// MaxActiveUsers defines the upper limit of concurrently active users to run across
	// the whole cluster.
	MaxActiveUsers int `default:"1000" validate:"range:(0,]"`
}
