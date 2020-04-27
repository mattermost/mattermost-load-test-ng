// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance"
	"github.com/mattermost/mattermost-load-test-ng/logger"

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
	NumUsersInc int `default:"16" validate:"range:(0,]"`
	// The number of users to decrement at each iteration of the feedback loop.
	// It should be proportional to the maximum number of users expected to test.
	NumUsersDec int `default:"16" validate:"range:(0,]"`
	// The number of seconds to wait after a performance degradation alert before
	// incrementing or decrementing users again.
	RestTimeSec int `default:"10" validate:"range:(0,]"`
	LogSettings logger.Settings
}

func ReadConfig(configFilePath string) (*Config, error) {
	v := viper.New()

	configName := "coordinator"
	v.SetConfigName(configName)
	v.AddConfigPath(".")
	v.AddConfigPath("./config/")
	// This is needed for the calls from the terraform package to find the config.
	v.AddConfigPath("../../config")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if configFilePath != "" {
		v.SetConfigFile(configFilePath)
	}

	if err := config.ReadConfigFile(v, configName); err != nil {
		return nil, err
	}

	var cfg *Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
