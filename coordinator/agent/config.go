// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package agent

import (
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
)

// LoadAgentConfig holds information about the load-test agent instance.
type LoadAgentConfig struct {
	// A sring that identifies the load-test agent instance.
	Id string `default:"lt0" validate:"alpha"`
	// The API URL used to control the specified load-test instance.
	ApiURL string `default:"http://localhost:4000" validate:"url"`
	// The configuration for the load-test to run.
	LoadTestConfig loadtest.Config
}
