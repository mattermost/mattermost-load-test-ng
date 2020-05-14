// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/logger"
)

// Config holds the necessary information to drive a cluster of
// load-test agents performing a load-test on a target instance.
type Config struct {
	// ClusterConfig defines the load-test agent cluster configuration.
	ClusterConfig cluster.LoadAgentClusterConfig
	// MonitorConfig holds the performance monitor configuration.
	MonitorConfig performance.MonitorConfig
	// The number of active users to increment at each iteration of the feedback loop.
	// It should be proportional to the maximum number of users expected to test.
	NumUsersInc int `default:"16" validate:"range:(0,]"`
	// The number of users to decrement at each iteration of the feedback loop.
	// It should be proportional to the maximum number of users expected to test.
	NumUsersDec int `default:"16" validate:"range:(0,]"`
	// The number of seconds to wait after a performance degradation alert before
	// incrementing or decrementing users again.
	RestTimeSec int `default:"10" validate:"range:(0,]"`
	LogSettings logger.Settings
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will return a config with default values.
func ReadConfig(configFilePath string) (*Config, error) {
	var cfg Config
	if configFilePath == "" {
		if err := defaults.Set(&cfg); err != nil {
			return nil, err
		}
		return &cfg, nil
	}

	file, err := os.Open(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %w", err)
	}

	err = json.NewDecoder(file).Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("could not decode file: %w", err)
	}

	return &cfg, nil
}
