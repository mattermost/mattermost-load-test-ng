// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package control

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func TestRecurringScheduledPostTimezoneForUser(t *testing.T) {
	tests := []struct {
		name     string
		user     *model.User
		expected string
		fallback bool
	}{
		{
			name: "valid manual timezone",
			user: &model.User{Timezone: model.StringMap{
				"manualTimezone": "Europe/London",
			}},
			expected: "Europe/London",
		},
		{
			name: "valid automatic timezone",
			user: &model.User{Timezone: model.StringMap{
				"useAutomaticTimezone": "true",
				"automaticTimezone":    "Asia/Tokyo",
				"manualTimezone":       "America/New_York",
			}},
			expected: "Asia/Tokyo",
		},
		{
			name: "automatic disabled uses manual timezone",
			user: &model.User{Timezone: model.StringMap{
				"useAutomaticTimezone": "false",
				"automaticTimezone":    "Asia/Tokyo",
				"manualTimezone":       "America/New_York",
			}},
			expected: "America/New_York",
		},
		{
			name: "automatic setting unset uses manual timezone",
			user: &model.User{Timezone: model.StringMap{
				"manualTimezone": "Europe/London",
			}},
			expected: "Europe/London",
		},
		{
			name:     "unset timezone falls back to pool",
			user:     &model.User{},
			fallback: true,
		},
		{
			name: "selected automatic timezone empty falls back to pool",
			user: &model.User{Timezone: model.StringMap{
				"useAutomaticTimezone": "true",
				"manualTimezone":       "Europe/London",
			}},
			fallback: true,
		},
		{
			name: "invalid timezone falls back to pool",
			user: &model.User{Timezone: model.StringMap{
				"manualTimezone": "invalid",
			}},
			fallback: true,
		},
		{
			name: "Local falls back to pool",
			user: &model.User{Timezone: model.StringMap{
				"manualTimezone": "Local",
			}},
			fallback: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := recurringScheduledPostTimezoneForUser(tt.user)

			if tt.fallback {
				require.Contains(t, recurringScheduledPostFallbackTimezones, actual)
			} else {
				require.Equal(t, tt.expected, actual)
			}
			require.NotEqual(t, "Local", actual)
			_, err := time.LoadLocation(actual)
			require.NoError(t, err)
		})
	}
}

func TestRecurringScheduledPostFallbackTimezones(t *testing.T) {
	require.NotEmpty(t, recurringScheduledPostFallbackTimezones)
	require.Contains(t, recurringScheduledPostFallbackTimezones, "UTC")
	require.NotContains(t, recurringScheduledPostFallbackTimezones, "Local")

	for _, timezone := range recurringScheduledPostFallbackTimezones {
		_, err := time.LoadLocation(timezone)
		require.NoError(t, err)
	}
}
