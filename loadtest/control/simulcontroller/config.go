// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
)

// Config holds information needed to run a SimulController.
type Config struct {
	// The minium amount of time (in milliseconds) the controlled users
	// will wait between actions.
	MinIdleTimeMs int `default:"1000" validate:"range:[0,]"`
	// The average amount of time (in milliseconds) the controlled users
	// will wait between actions.
	AvgIdleTimeMs int `default:"5000" validate:"range:($MinIdleTimeMs,]"`
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
