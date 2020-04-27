// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simplecontroller

import (
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/config"

	"github.com/spf13/viper"
)

// Config holds the the rate and user actions definitions that will be run by
// the SimpleController.
type Config struct {
	// Actions are the user action definitions that will be run by the controller.
	Actions []actionDefinition `default_size:"1"`
}

type actionDefinition struct {
	// ActionId is the key of an action which is mapped to a user action
	// implementation.
	ActionId string `default:"Login" validate:"text"`
	// RunPeriod determines how often the action will be performed.
	RunPeriod int `default:"20" validate:"range:[0,]"`
	// WaitAfterMs is the wait time after the action is performed.
	WaitAfterMs int `default:"1000" validate:"range:(0,]"`
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will search a config file in predefined folders.
func ReadConfig(configFilePath string) (*Config, error) {
	v := viper.New()

	configName := "simplecontroller"
	v.SetConfigName(configName)
	v.AddConfigPath(".")
	v.AddConfigPath("./config/")
	v.AddConfigPath("./../config/")
	v.AddConfigPath("./../../../config/")
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
