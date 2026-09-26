// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"strings"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost/server/public/model"
)

// maxScheduledRecapsPageSize matches the server's maximum per_page value.
const maxScheduledRecapsPageSize = 200

// RecapsConfiguration controls the built-in AI Recaps load-test actions.
type RecapsConfiguration struct {
	// Enabled adds the recap actions to the simulated controller.
	Enabled bool `default:"false"`
	// AgentUsername identifies the Agents bot used to process recaps.
	AgentUsername string `default:"ai"`
	// MaxChannelsPerRecap limits how many channels one recap can include.
	MaxChannelsPerRecap int `default:"3"`
	// PollIntervalMs is the delay between recap status requests.
	PollIntervalMs int `default:"5000"`
	// PollTimeoutMs is the maximum time spent polling one recap.
	PollTimeoutMs int `default:"120000"`
	// MaxScheduledRecapsPerUser limits schedules created for one user.
	MaxScheduledRecapsPerUser int `default:"1"`
	// ScheduledRecapChannelMode selects specific channels or all unread channels.
	ScheduledRecapChannelMode string `default:"specific"`
	// ScheduledRecapDueTime aligns schedules at an optional HH:MM UTC time.
	ScheduledRecapDueTime string
}

func (c RecapsConfiguration) IsValid() error {
	if !c.Enabled {
		return nil
	}
	if strings.TrimSpace(c.AgentUsername) == "" {
		return errors.New("agent username cannot be empty when recaps are enabled")
	}
	if c.MaxChannelsPerRecap <= 0 {
		return errors.New("max channels per recap must be greater than zero")
	}
	if c.PollIntervalMs <= 0 {
		return errors.New("poll interval must be greater than zero")
	}
	if c.PollTimeoutMs <= 0 {
		return errors.New("poll timeout must be greater than zero")
	}
	if c.PollIntervalMs > c.PollTimeoutMs {
		return errors.New("poll interval cannot exceed poll timeout")
	}
	if c.MaxScheduledRecapsPerUser < 0 || c.MaxScheduledRecapsPerUser > maxScheduledRecapsPageSize {
		return errors.New("max scheduled recaps per user must be between 0 and 200")
	}
	if c.ScheduledRecapChannelMode != model.ChannelModeSpecific && c.ScheduledRecapChannelMode != model.ChannelModeAllUnreads {
		return errors.New("scheduled recap channel mode must be specific or all_unreads")
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

	// RecapsConfiguration configures AI Recaps actions.
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
