// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package loadtest

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/spf13/viper"
)

type ConnectionConfiguration struct {
	ServerURL     string
	WebSocketURL  string
	AdminEmail    string
	AdminPassword string
}

// IsValid reports whether a given ConnectionConfiguration is valid or not.
// Returns an error if the validation fails.
func (cc *ConnectionConfiguration) IsValid() error {
	if cc.ServerURL == "" {
		return fmt.Errorf("ServerURL is not present in config")
	}
	if cc.WebSocketURL == "" {
		return fmt.Errorf("WebSocketURL is not present in config")
	}
	if cc.AdminEmail == "" {
		return fmt.Errorf("AdminEmail is not present in config")
	}
	if cc.AdminPassword == "" {
		return fmt.Errorf("AdminPassword is not present in config")
	}
	return nil
}

// userControllerType describes the type of a UserController.
type userControllerType string

// Available UserController implementations.
const (
	UserControllerSimple     userControllerType = "simple"
	UserControllerSimulative                    = "simulative"
	UserControllerNoop                          = "noop"
	UserControllerGenerative                    = "generative"
)

// IsValid reports whether a given UserControllerType is valid or not.
// Returns an error if the validation fails.
func (t userControllerType) IsValid() error {
	switch t {
	case UserControllerSimple, UserControllerSimulative, UserControllerNoop, UserControllerGenerative:
		return nil
	case "":
		return fmt.Errorf("UserControllerType cannot be empty")
	default:
		return fmt.Errorf("UserControllerType %s is not valid", t)
	}
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
	Type userControllerType
	// A rate multiplier that will affect the speed at which user actions are
	// executed by the UserController.
	// A Rate of < 1.0 will run actions at a faster pace.
	// A Rate of 1.0 will run actions at the default pace.
	// A Rate > 1.0 will run actions at a slower pace.
	Rate float64
}

// IsValid reports whether a given UserControllerConfiguration is valid or not.
// Returns an error if the validation fails.
func (ucc *UserControllerConfiguration) IsValid() error {
	if err := ucc.Type.IsValid(); err != nil {
		return fmt.Errorf("could not validate configuration: %w", err)
	}
	if ucc.Rate < 0 {
		return fmt.Errorf("rate cannot be < 0")
	}
	return nil
}

type InstanceConfiguration struct {
	NumTeams          int
	NumChannels       int
	NumTeamAdmins     int
	TeamAdminInterval int
}

// IsValid reports whether a given InstanceConfiguration is valid or not.
// Returns an error if the validation fails.
func (ic *InstanceConfiguration) IsValid() error {
	if ic.NumTeams <= 0 {
		return fmt.Errorf("NumTeams cannot be <= 0")
	}
	return nil
}

type UsersConfiguration struct {
	InitialActiveUsers int
	MaxActiveUsers     int
	AvgSessionsPerUser int
}

// IsValid reports whether a given UsersConfiguration is valid or not.
func (uc *UsersConfiguration) IsValid() error {
	if uc.InitialActiveUsers < 0 {
		return fmt.Errorf("InitialActiveUsers cannot be < 0")
	}
	if uc.MaxActiveUsers <= 0 {
		return fmt.Errorf("MaxActiveUsers cannot be <= 0")
	}
	if uc.InitialActiveUsers > uc.MaxActiveUsers {
		return fmt.Errorf("InitialActiveUsers cannot be greater than MaxActiveUsers")
	}
	if uc.AvgSessionsPerUser < 1 {
		return fmt.Errorf("AvgSessionsPerUser cannot be < 1")
	}
	return nil
}

type Config struct {
	ConnectionConfiguration     ConnectionConfiguration
	UserControllerConfiguration UserControllerConfiguration
	InstanceConfiguration       InstanceConfiguration
	UsersConfiguration          UsersConfiguration
	LogSettings                 logger.Settings
}

// IsValid reports whether a config is valid or not.
// Returns an error if the validation fails.
func (c *Config) IsValid() error {
	if err := c.ConnectionConfiguration.IsValid(); err != nil {
		return fmt.Errorf("invalid connection configuration: %w", err)
	}

	if err := c.InstanceConfiguration.IsValid(); err != nil {
		return fmt.Errorf("invalid instance configuration: %w", err)
	}

	if err := c.UserControllerConfiguration.IsValid(); err != nil {
		return fmt.Errorf("invalid user controller configuration: %w", err)
	}

	if err := c.UsersConfiguration.IsValid(); err != nil {
		return fmt.Errorf("invalid users configuration: %w", err)
	}

	return nil
}

func ReadConfig(configFilePath string) (*Config, error) {
	v := viper.New()

	configName := "config"
	v.SetConfigName(configName)
	v.AddConfigPath(".")
	v.AddConfigPath("./config/")
	// This is needed for the calls from the terraform package to find the config.
	v.AddConfigPath("../../config")
	v.SetEnvPrefix("mmloadtest")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("LogSettings.EnableConsole", true)
	v.SetDefault("LogSettings.ConsoleLevel", "INFO")
	v.SetDefault("LogSettings.ConsoleJson", true)
	v.SetDefault("LogSettings.EnableFile", true)
	v.SetDefault("LogSettings.FileLevel", "INFO")
	v.SetDefault("LogSettings.FileJson", true)
	v.SetDefault("LogSettings.FileLocation", "loadtest.log")

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
