// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
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
	NumUsersInc int
	// The number of users to decrement at each iteration of the feedback loop.
	// It should be proportional to the maximum number of users expected to test.
	NumUsersDec int
	// The number of seconds to wait after a performance degradation alert before
	// incrementing or decrementing users again.
	RestTimeSec int
	LogSettings logger.Settings
}

func ReadConfig(configFilePath string) (*Config, error) {
	v := viper.New()

	v.SetConfigName("coordinator")
	v.AddConfigPath(".")
	v.AddConfigPath("./config/")
	// This is needed for the calls from the terraform package to find the config.
	v.AddConfigPath("../../config")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if configFilePath != "" {
		v.SetConfigFile(configFilePath)
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, errors.Wrap(err, "unable to read configuration file")
	}

	var cfg *Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// IsValid checks whether a Config is valid or not.
// Returns an error if the validation fails.
func (c *Config) IsValid() error {
	if err := c.ClusterConfig.IsValid(); err != nil {
		return fmt.Errorf("cluster config validation failed: %w", err)
	}

	if err := c.MonitorConfig.IsValid(); err != nil {
		return fmt.Errorf("monitor config validation failed: %w", err)
	}

	if c.NumUsersInc <= 0 {
		return fmt.Errorf("NumUsersInc cannot be less than 1")
	}

	if c.NumUsersDec <= 0 {
		return fmt.Errorf("NumUsersDec cannot be less than 1")
	}

	if c.RestTimeSec <= 0 {
		return fmt.Errorf("RestTimeSec cannot be less than 1")
	}

	return nil
}
