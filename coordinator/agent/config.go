// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package agent

import (
	"fmt"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
)

// LoadAgentConfig holds information about the load-test agent instance.
type LoadAgentConfig struct {
	// A sring that identifies the load-test agent instance.
	Id string
	// The API URL used to control the specified load-test instance.
	ApiURL string
	// The configuration for the load-test to run.
	LoadTestConfig loadtest.LoadTestConfig
}

// IsValid checks whether a LoadAgentConfig is valid or not.
// Returns an error if the validtation fails.
func (c LoadAgentConfig) IsValid() (bool, error) {
	if c.Id == "" {
		return false, fmt.Errorf("Id should not be empty")
	}
	if c.ApiURL == "" {
		return false, fmt.Errorf("ApiURL should not be empty")
	}
	return c.LoadTestConfig.IsValid()
}
