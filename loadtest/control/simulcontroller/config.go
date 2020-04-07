// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/config"

	"github.com/spf13/viper"
)

// Config holds information needed to run a SimulController.
type Config struct {
	// The minium amount of time (in milliseconds) the controlled users
	// will wait between actions.
	MinIdleTimeMs int
	// The average amount of time (in milliseconds) the controlled users
	// will wait between actions.
	AvgIdleTimeMs int
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will search a config file in predefined folders.
func ReadConfig(configFilePath string) (*Config, error) {
	v := viper.New()

	configName := "simulcontroller"
	v.SetConfigName(configName)
	v.AddConfigPath(".")
	v.AddConfigPath("./config/")
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

// IsValid reports whether a given simulcontroller.Config is valid or not.
// Returns an error if the validation fails.
func (c *Config) IsValid() error {
	if c.MinIdleTimeMs <= 0 {
		return fmt.Errorf("MinIdleTimeMs should be greater than zero")
	}

	if c.AvgIdleTimeMs <= c.MinIdleTimeMs {
		return fmt.Errorf("AvgIdleTimeMs should be greater than MinIdleTimeMs")
	}

	return nil
}
