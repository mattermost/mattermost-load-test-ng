// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package browsercontroller

import (
	"github.com/mattermost/mattermost-load-test-ng/defaults"
)

// BrowserLogSettings holds information to be used to initialize the logger for the LTBrowser API
// refer to /browser/src/utils/log.ts
type BrowserLogSettings struct {
	EnableConsole bool   `default:"false"`
	ConsoleLevel  string `default:"error" validate:"oneof:{trace, debug, info, warn, error, fatal}"`
	EnableFile    bool   `default:"true"`
	FileLevel     string `default:"debug" validate:"oneof:{trace, debug, info, warn, error, fatal}"`
	FileLocation  string `default:"browseragent.log"`
}

// Config holds information needed to run a BrowserController.
type Config struct {
	// The ID of the simulation to run.
	SimulationId string `default:"mattermostPostAndScroll" validate:"notempty"`
	// Whether to run the browser in headless mode.
	RunInHeadless bool `default:"true"`
	// The timeout in milliseconds for browser simulations.
	SimulationTimeoutMs int `default:"60000" validate:"range:[0,]"`
	// Whether to enable plugins in the browser simulation.
	EnabledPlugins bool `default:"false"`
	// Log settings for the LTBrowser API
	LogSettings BrowserLogSettings
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will return a config with default values.
func ReadConfig(configFilePath string) (*Config, error) {
	var cfg Config

	if err := defaults.ReadFrom(configFilePath, "./config/browsercontroller.json", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
