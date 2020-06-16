// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package loadtest

import (
	"errors"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/gencontroller"
	"github.com/mattermost/mattermost-load-test-ng/logger"
)

type ConnectionConfiguration struct {
	ServerURL     string `default:"http://localhost:8065" validate:"url"`
	WebSocketURL  string `default:"ws://localhost:8065" validate:"url"`
	AdminEmail    string `default:"sysadmin@sample.mattermost.com" validate:"email"`
	AdminPassword string `default:"Sys@dmin-sample1" validate:"notempty"`
}

// userControllerType describes the type of a UserController.
type userControllerType string

// Available UserController implementations.
const (
	UserControllerSimple     userControllerType = "simple"
	UserControllerSimulative                    = "simulative"
	UserControllerNoop                          = "noop"
	UserControllerGenerative                    = "generative"
	UserControllerCluster                       = "cluster"
)

type RatesDistribution struct {
	Rate       float64 `default:"1.0" validate:"range:[0,)"`
	Percentage float64 `default:"1.0" validate:"range:(0,100]"`
}

// UserControllerConfiguration holds information about the UserController to
// run during a load-test.
type UserControllerConfiguration struct {
	// The type of the UserController to run.
	// Possible values:
	//   UserControllerSimple - A simple version of a controller.
	//   UserControllerSimulative - A more realistic controller.
	//   UserControllerNoop
	//   UserControllerGenerative - A controller used to generate data.
	Type userControllerType `default:"simulative" validate:"oneof:{simple,simulative,noop,cluster,generative}"`
	// A distribution of rate multipliers that will affect the speed at which user actions are
	// executed by the UserController.
	// A Rate of < 1.0 will run actions at a faster pace.
	// A Rate of 1.0 will run actions at the default pace.
	// A Rate > 1.0 will run actions at a slower pace.
	RatesDistribution []RatesDistribution `default_len:"1"`
}

// IsValid reports whether a given UserControllerConfiguration is valid or not.
// Returns an error if the validation fails.
func (ucc *UserControllerConfiguration) IsValid() error {
	var sum float64
	for _, el := range ucc.RatesDistribution {
		sum += el.Percentage
	}
	if len(ucc.RatesDistribution) > 0 && sum != 1 {
		return errors.New("Percentages in RatesDistribution should sum to 1")
	}
	return nil
}

type UsersConfiguration struct {
	InitialActiveUsers int `default:"0" validate:"range:[0,$MaxActiveUsers]"`
	MaxActiveUsers     int `default:"2000" validate:"range:(0,]"`
	AvgSessionsPerUser int `default:"1" validate:"range:[1,]"`
}

type Config struct {
	ConnectionConfiguration     ConnectionConfiguration
	UserControllerConfiguration UserControllerConfiguration
	InstanceConfiguration       gencontroller.Config
	UsersConfiguration          UsersConfiguration
	LogSettings                 logger.Settings
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will return a config with default values.
func ReadConfig(configFilePath string) (*Config, error) {
	var cfg Config

	if err := defaults.ReadFromJSON(configFilePath, "./config/config.json", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
