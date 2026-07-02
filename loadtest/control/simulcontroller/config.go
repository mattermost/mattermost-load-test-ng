// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"github.com/mattermost/mattermost-load-test-ng/defaults"
)

// Config holds information needed to run a SimulController.
type Config struct {
	// The minium amount of time (in milliseconds) the controlled users
	// will wait between actions.
	MinIdleTimeMs int `default:"1000" validate:"range:[0,]"`
	// The average amount of time (in milliseconds) the controlled users
	// will wait between actions.
	AvgIdleTimeMs int `default:"20000" validate:"range:($MinIdleTimeMs,]"`

	// The percentage of root posts that are marked as urgent
	PercentUrgentPosts float64 `default:"0.001" validate:"range:[0,1]"`
	// The percentage of all posts that are replies
	PercentReplies float64 `default:"0.18" validate:"range:[0,1]"`

	// The IDs of the enabled plugins.
	EnabledPlugins []string

	// Path to the MBE channel-ID JSON file emitted by cmd/ltmbebootstrap. When set, the listed
	// channels are steered per MBEChannelWeight; when empty, MBE steering is disabled (default,
	// pre-MBE behavior). See mbe-load-test-plan.md WS4b.
	MBEChannelIdsFile string `default:""`
	// The target share (0 to 1) of channel-targeted post/read actions steered to MBE channels.
	// 0 hard-excludes MBE channels from the channel candidate pool (control/0% arms).
	MBEChannelWeight float64 `default:"0" validate:"range:[0,1]"`
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will return a config with default values.
func ReadConfig(configFilePath string) (*Config, error) {
	var cfg Config

	if err := defaults.ReadFrom(configFilePath, "./config/simulcontroller.json", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
