// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package loadtest

import (
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/spf13/viper"
)

type ConnectionConfiguration struct {
	ServerURL     string `default:"http://localhost:8065" validate:"url"`
	WebSocketURL  string `default:"ws://localhost:8065" validate:"url"`
	AdminEmail    string `default:"sysadmin@sample.mattermost.com" validate:"email"`
	AdminPassword string `default:"Sys@dmin-sample1" validate:"text"`
}

// userControllerType describes the type of a UserController.
type userControllerType string

// Available UserController implementations.
const (
	UserControllerSimple     userControllerType = "simple"
	UserControllerSimulative                    = "simulative"
	UserControllerNoop                          = "noop"
)

// UserControllerConfiguration holds information about the UserController to
// run during a load-test.
type UserControllerConfiguration struct {
	// The type of the UserController to run.
	// Possible values:
	//   UserControllerSimple - A simple version of a controller.
	//   UserControllerSimulative - A more realistic controller.
	Type userControllerType `default:"simple" validate:"oneof:{simple,simulative,noop}"`
	// A rate multiplier that will affect the speed at which user actions are
	// executed by the UserController.
	// A Rate of < 1.0 will run actions at a faster pace.
	// A Rate of 1.0 will run actions at the default pace.
	// A Rate > 1.0 will run actions at a slower pace.
	Rate float64 `default:"1.0" validate:"range:(0,)"`
}

type InstanceConfiguration struct {
	NumTeams          int `default:"2" validate:"range:(0,]"`
	NumChannels       int `default:"10"`
	NumTeamAdmins     int `default:"2"`
	TeamAdminInterval int `default:"10"`
}

type UsersConfiguration struct {
	InitialActiveUsers int `default:"0" validate:"range:[0,$MaxActiveUsers]"`
	MaxActiveUsers     int `default:"1000" validate:"range:(0,]"`
	AvgSessionsPerUser int `default:"1" validate:"range:[1,]"`
}

type Config struct {
	ConnectionConfiguration     ConnectionConfiguration
	UserControllerConfiguration UserControllerConfiguration
	InstanceConfiguration       InstanceConfiguration
	UsersConfiguration          UsersConfiguration
	LogSettings                 logger.Settings
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
	v.SetDefault("LogSettings.ConsoleLevel", "ERROR")
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
