// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simplecontroller

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
)

// Config holds the the rate and user actions definitions that will be run by
// the SimpleController.
type Config struct {
	// Actions are the user action definitions that will be run by the controller.
	Actions []actionDefinition
}

type actionDefinition struct {
	// ActionId is the key of an action which is mapped to a user action
	// implementation.
	ActionId string `default:"Login" validate:"notempty"`
	// RunPeriod determines how often the action will be performed.
	RunPeriod int `default:"20" validate:"range:[0,]"`
	// WaitAfterMs is the wait time after the action is performed.
	WaitAfterMs int `default:"1000" validate:"range:[0,]"`
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
