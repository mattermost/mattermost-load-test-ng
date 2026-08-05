// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"strings"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
)

type RecapsConfiguration struct {
	Enabled                   bool   `default:"false"`
	AgentUsername             string `default:"ai"`
	MaxChannelsPerRecap       int    `default:"3" validate:"range:(0,]"`
	PollIntervalMs            int    `default:"5000" validate:"range:(0,]"`
	PollTimeoutMs             int    `default:"120000" validate:"range:(0,]"`
	MaxScheduledRecapsPerUser int    `default:"1" validate:"range:[0,]"`
	ScheduledRecapChannelMode string `default:"specific" validate:"oneof:{specific,all_unreads}"`
	ScheduledRecapDueTime     string
}

func (c RecapsConfiguration) IsValid() error {
	if c.Enabled && strings.TrimSpace(c.AgentUsername) == "" {
		return errors.New("agent username cannot be empty when recaps are enabled")
	}
	if c.ScheduledRecapDueTime == "" {
		return nil
	}
	if parsed, err := time.Parse("15:04", c.ScheduledRecapDueTime); err != nil || parsed.Format("15:04") != c.ScheduledRecapDueTime {
		return errors.New("scheduled recap due time must use HH:MM format in UTC")
	}
	return nil
}

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

	RecapsConfiguration RecapsConfiguration
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
