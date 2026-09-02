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
	// The portion of newly created scheduled posts aligned to a shared batch boundary.
	PercentBatchAlignedScheduledPosts float64 `default:"0.5" validate:"range:[0,1]"`
	// The number of minutes between shared scheduled-post batch boundaries.
	ScheduledPostBatchIntervalMinutes int `default:"30" validate:"range:[5,60]"`
	// The minimum number of minutes before a shared scheduled-post batch boundary.
	ScheduledPostBatchMinLeadMinutes int `default:"10" validate:"range:[5,60]"`

	// The IDs of the enabled plugins.
	EnabledPlugins []string
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
