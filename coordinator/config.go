// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/cluster"
	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance"

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
}

var v = viper.New()

func ReadConfig(configFilePath string) error {
	v.SetConfigName("coordinator")
	v.AddConfigPath(".")
	v.AddConfigPath("./config/")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if configFilePath != "" {
		v.SetConfigFile(configFilePath)
	}

	if err := v.ReadInConfig(); err != nil {
		return errors.Wrap(err, "unable to read configuration file")
	}

	return nil
}

func GetConfig() (*Config, error) {
	var cfg *Config

	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// IsValid checks whether a Config is valid or not.
// Returns an error if the validation fails.
func (c *Config) IsValid() error {
	return nil
}
