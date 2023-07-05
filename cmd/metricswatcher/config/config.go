// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package config

import (
	"os"

	"github.com/mattermost/mattermost-load-test-ng/coordinator/performance/prometheus"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/mattermost/mattermost-server/server/v8/platform/shared/mlog"
	"github.com/spf13/cobra"
)

type MetricsWatcherConfiguration struct {
	LogSettings             logger.Settings
	PrometheusConfiguration prometheus.Configuration
	Queries                 []prometheus.Query `default_size:"1"`
}

func SetupMetricsCheck(cmd *cobra.Command, args []string) {
	configFilePath, _ := cmd.Flags().GetString("config")

	cfg, err := ReadConfig(configFilePath)
	if err != nil {
		mlog.Error("Failed to initialize config", mlog.Err(err))
		os.Exit(1)
	}

	logger.Init(&cfg.LogSettings)
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will return a config with default values.
func ReadConfig(configFilePath string) (*MetricsWatcherConfiguration, error) {
	var cfg MetricsWatcherConfiguration

	if err := defaults.ReadFromJSON(configFilePath, "./config/metricswatcher.json", &cfg); err != nil {
		return nil, err
	}

	if configFilePath == "" {
		cfg.LogSettings.FileLocation = "metricswatcher.log"
	}

	return &cfg, nil
}
