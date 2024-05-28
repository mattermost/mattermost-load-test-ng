// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/logger"
)

const defaultConfigLocation = "./config/coordinator.json"

// Config holds the necessary information to drive a cluster of
// load-test agents performing a load-test on a target instance.
type Config struct {
	// ClusterConfig defines the load-test agent cluster configuration.
	ClusterConfig cluster.LoadAgentClusterConfig
	// MonitorConfig holds the performance monitor configuration.
	MonitorConfig performance.MonitorConfig
	// The number of active users to increment at each iteration of the feedback loop.
	// It should be proportional to the maximum number of users expected to test.
	NumUsersInc int `default:"8" validate:"range:(0,]"`
	// The number of users to decrement at each iteration of the feedback loop.
	// It should be proportional to the maximum number of users expected to test.
	NumUsersDec int `default:"8" validate:"range:(0,]"`
	// The number of seconds to wait after a performance degradation alert before
	// incrementing or decrementing users again.
	RestTimeSec int `default:"2" validate:"range:(0,]"`
	LogSettings logger.Settings
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will return a config with default values.
func ReadConfig(configFilePath string) (*Config, error) {
	var cfg Config

	if err := defaults.ReadFrom(configFilePath, defaultConfigLocation, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
