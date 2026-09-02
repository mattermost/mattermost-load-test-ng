// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/stretchr/testify/require"
)

func TestScheduledPostBatchConfigValidation(t *testing.T) {
	defaultConfig := &Config{}
	require.NoError(t, defaults.Set(defaultConfig))
	require.Equal(t, 0.5, defaultConfig.PercentBatchAlignedScheduledPosts)
	require.Equal(t, 30, defaultConfig.ScheduledPostBatchIntervalMinutes)
	require.Equal(t, 10, defaultConfig.ScheduledPostBatchMinLeadMinutes)
	require.NoError(t, defaults.Validate(defaultConfig))

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{
			name: "defaults",
		},
		{
			name: "percent lower boundary",
			mutate: func(config *Config) {
				config.PercentBatchAlignedScheduledPosts = 0
			},
		},
		{
			name: "percent upper boundary",
			mutate: func(config *Config) {
				config.PercentBatchAlignedScheduledPosts = 1
			},
		},
		{
			name: "percent below range",
			mutate: func(config *Config) {
				config.PercentBatchAlignedScheduledPosts = -0.01
			},
			wantErr: true,
		},
		{
			name: "percent above range",
			mutate: func(config *Config) {
				config.PercentBatchAlignedScheduledPosts = 1.01
			},
			wantErr: true,
		},
		{
			name: "interval lower boundary",
			mutate: func(config *Config) {
				config.ScheduledPostBatchIntervalMinutes = 5
			},
		},
		{
			name: "interval upper boundary",
			mutate: func(config *Config) {
				config.ScheduledPostBatchIntervalMinutes = 60
			},
		},
		{
			name: "interval below range",
			mutate: func(config *Config) {
				config.ScheduledPostBatchIntervalMinutes = 4
			},
			wantErr: true,
		},
		{
			name: "interval above range",
			mutate: func(config *Config) {
				config.ScheduledPostBatchIntervalMinutes = 61
			},
			wantErr: true,
		},
		{
			name: "minimum lead lower boundary",
			mutate: func(config *Config) {
				config.ScheduledPostBatchMinLeadMinutes = 5
			},
		},
		{
			name: "minimum lead upper boundary",
			mutate: func(config *Config) {
				config.ScheduledPostBatchMinLeadMinutes = 60
			},
		},
		{
			name: "minimum lead below range",
			mutate: func(config *Config) {
				config.ScheduledPostBatchMinLeadMinutes = 4
			},
			wantErr: true,
		},
		{
			name: "minimum lead above range",
			mutate: func(config *Config) {
				config.ScheduledPostBatchMinLeadMinutes = 61
			},
			wantErr: true,
		},
		{
			name: "interval and minimum lead validate independently",
			mutate: func(config *Config) {
				config.ScheduledPostBatchIntervalMinutes = 5
				config.ScheduledPostBatchMinLeadMinutes = 60
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			require.NoError(t, defaults.Set(config))
			if tt.mutate != nil {
				tt.mutate(config)
			}

			err := defaults.Validate(config)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
