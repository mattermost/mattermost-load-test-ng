// Copyright (c) 2019 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package config

import (
	"os"
	"strings"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Configuration struct {
	ConnectionConfiguration ConnectionConfiguration
	InstanceConfiguration   InstanceConfiguration
	UsersConfiguration      UsersConfiguration
	LogSettings             LoggerSettings
}

type ConnectionConfiguration struct {
	ServerURL                   string
	WebSocketURL                string
	PrometheusURL               string
	DriverName                  string
	DataSource                  string
	AdminEmail                  string
	AdminPassword               string
	MaxIdleConns                int
	MaxIdleConnsPerHost         int
	IdleConnTimeoutMilliseconds int
}

type InstanceConfiguration struct {
	NumTeams int
}

type UsersConfiguration struct {
	InitialActiveUsers int
	MaxActiveUsers     int
}

type LoggerSettings struct {
	EnableConsole bool
	ConsoleJson   bool
	ConsoleLevel  string
	EnableFile    bool
	FileJson      bool
	FileLevel     string
	FileLocation  string
}

func Setup(cmd *cobra.Command, args []string) {
	configFilePath, _ := cmd.Flags().GetString("config")

	if err := ReadConfig(configFilePath); err != nil {
		mlog.Error("Failed to initialize config", mlog.Err(err))
		os.Exit(1)
	}

	cfg, err := GetConfig()
	if err != nil {
		mlog.Error("Failed to get logging config: %s\n", mlog.Err(err))
		os.Exit(1)
	}

	initLogger(&cfg.LogSettings)
}

func ReadConfig(configFilePath string) error {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config/")
	viper.SetEnvPrefix("mmloadtest")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	viper.SetDefault("LogSettings.EnableConsole", true)
	viper.SetDefault("LogSettings.ConsoleLevel", "INFO")
	viper.SetDefault("LogSettings.ConsoleJson", true)
	viper.SetDefault("LogSettings.EnableFile", true)
	viper.SetDefault("LogSettings.FileLevel", "INFO")
	viper.SetDefault("LogSettings.FileJson", true)
	viper.SetDefault("LogSettings.FileLocation", "loadtest.log")
	viper.SetDefault("ConnectionConfiguration.MaxIdleConns", 100)
	viper.SetDefault("ConnectionConfiguration.MaxIdleConnsPerHost", 128)
	viper.SetDefault("ConnectionConfiguration.IdleConnTimeoutMilliseconds", 90000)

	if configFilePath != "" {
		viper.SetConfigFile(configFilePath)
	}

	if err := viper.ReadInConfig(); err != nil {
		return errors.Wrap(err, "unable to read configuration file")
	}

	return nil
}

func GetConfig() (*Configuration, error) {
	var cfg *Configuration

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
