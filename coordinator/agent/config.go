// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
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
	LoadTestConfig loadtest.Config
}

// IsValid checks whether a LoadAgentConfig is valid or not.
// Returns an error if the validation fails.
func (c LoadAgentConfig) IsValid() error {
	if c.Id == "" {
		return fmt.Errorf("Id should not be empty")
	}
	if c.ApiURL == "" {
		return fmt.Errorf("ApiURL should not be empty")
	}
	return c.LoadTestConfig.IsValid()
}
