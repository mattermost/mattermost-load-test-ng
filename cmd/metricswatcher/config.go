// Copyright (c) 2019 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package main

import (
	"github.com/mattermost/mattermost-load-test-ng/logger"
	"os"
	"strings"

	"github.com/mattermost/mattermost-server/v5/mlog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)




type MetricsCheckConfig struct {
	LogSettings             logger.LoggerSettings
	PrometheusConfiguration PrometheusConfiguration
}


type PrometheusConfiguration struct {
	PrometheusURL                 string
	MetricsUpdateIntervalInMS     int
	HealthcheckUpdateIntervalInMS int
}

func setupMetricsCheck(cmd *cobra.Command, args []string) {
	configFilePath, _ := cmd.Flags().GetString("config")

	if err := ReadConfig(configFilePath); err != nil {
		mlog.Error("Failed to initialize config", mlog.Err(err))
		os.Exit(1)
	}

	cfg, err := GetMetricsCheckConfig()

	if err != nil {
		mlog.Error("Failed to get logging config:", mlog.Err(err))
		os.Exit(1)
	}

	logger.InitLogger(&cfg.LogSettings)
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
	viper.SetDefault("LogSettings.FileLocation", "metricscheck.log")

	viper.SetDefault("PrometheusConfiguration.PrometheusURL", "http://localhost:9090")
	viper.SetDefault("PrometheusConfiguration.MetricsUpdateIntervalInMS", 5000)
	viper.SetDefault("PrometheusConfiguration.HealthcheckUpdateIntervalInMS", 60000)


	if configFilePath != "" {
		viper.SetConfigFile(configFilePath)
	}

	if err := viper.ReadInConfig(); err != nil {
		return errors.Wrap(err, "unable to read configuration file")
	}

	return nil
}


func GetMetricsCheckConfig() (*MetricsCheckConfig, error) {
	var cfg *MetricsCheckConfig

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
