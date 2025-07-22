// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package cluster

import (
	"errors"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
)

// LoadAgentConfig holds information about the load-test agent instance.
type LoadAgentConfig struct {
	// A sring that identifies the load-test agent instance.
	Id string `default:"lt0" validate:"notempty"`
	// The API URL used to control the specified load-test instance.
	ApiURL string `default:"http://localhost:4000" validate:"url"`
}

// LoadAgentClusterConfig holds information regarding the cluster of load-test
// agents.
type LoadAgentClusterConfig struct {
	// Agents is a list of the load-test agents API endpoints to be used during
	// the load-test. Its length defines the number of load-test instances
	// used during a load-test.
	Agents []LoadAgentConfig `default_size:"1"`
	// MaxActiveUsers defines the upper limit of concurrently active users to run across
	// the whole cluster.
	MaxActiveUsers int `default:"1000" validate:"range:[0,]"`
	// BrowserAgents is a list of the browser agents API endpoints to be used during
	// the load-test. Its length defines the number of browser agents instances
	// used during a load-test.
	BrowserAgents []LoadAgentConfig `default_size:"0" validate:"range:[0,]"`
	// MaxActiveBrowserUsers defines the upper limit of concurrently active browser users to run across
	// the whole cluster.
	MaxActiveBrowserUsers int `default:"0" validate:"range:[0,]"`
}

func (c *LoadAgentClusterConfig) IsValid(ltConfig loadtest.Config) error {
	if len(c.Agents) > 0 && ltConfig.UsersConfiguration.MaxActiveUsers*len(c.Agents) < c.MaxActiveUsers {
		return errors.New("coordinator: total MaxActiveUsers in loadTest should not be less than clusterConfig.MaxActiveUsers")
	}

	if len(c.BrowserAgents) > 0 && ltConfig.UsersConfiguration.MaxActiveBrowserUsers*len(c.BrowserAgents) < c.MaxActiveBrowserUsers {
		return errors.New("coordinator: total MaxActiveBrowserUsers in loadTest should not be less than clusterConfig.MaxActiveBrowserUsers")
	}

	return nil
}
