// Copyright (c) 2019 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package loadtest

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type LoadTestConfig struct {
	ConnectionConfiguration ConnectionConfiguration
	LogSettings             LoggerSettings
}

type ConnectionConfiguration struct {
	ServerURL                   string
	WebSocketURL                string
	DriverName                  string
	DataSource                  string
	MaxIdleConns                int
	MaxIdleConnsPerHost         int
	IdleConnTimeoutMilliseconds int
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

func ReadConfig() error {
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

	if err := viper.ReadInConfig(); err != nil {
		return errors.Wrap(err, "unable to read configuration file")
	}

	return nil
}

func GetConfig() (*LoadTestConfig, error) {
	var cfg *LoadTestConfig

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
