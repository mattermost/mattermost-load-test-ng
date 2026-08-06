// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestNextScheduledPostBatchBoundary(t *testing.T) {
	tests := []struct {
		name       string
		now        string
		interval   time.Duration
		minLead    time.Duration
		expectedAt string
	}{
		{
			name:       "ordinary",
			now:        "2026-08-05T10:07:00Z",
			interval:   30 * time.Minute,
			minLead:    10 * time.Minute,
			expectedAt: "2026-08-05T10:30:00Z",
		},
		{
			name:       "eligible exactly on boundary",
			now:        "2026-08-05T10:20:00Z",
			interval:   30 * time.Minute,
			minLead:    10 * time.Minute,
			expectedAt: "2026-08-05T10:30:00Z",
		},
		{
			name:       "one millisecond after cutoff",
			now:        "2026-08-05T10:20:00.001Z",
			interval:   30 * time.Minute,
			minLead:    10 * time.Minute,
			expectedAt: "2026-08-05T11:00:00Z",
		},
		{
			name:       "now exactly on boundary",
			now:        "2026-08-05T10:00:00Z",
			interval:   30 * time.Minute,
			minLead:    10 * time.Minute,
			expectedAt: "2026-08-05T10:30:00Z",
		},
		{
			name:       "alternate interval",
			now:        "2026-08-05T10:07:00Z",
			interval:   15 * time.Minute,
			minLead:    5 * time.Minute,
			expectedAt: "2026-08-05T10:15:00Z",
		},
		{
			name:       "same instant in non-UTC representation",
			now:        "2026-08-05T06:07:00-04:00",
			interval:   30 * time.Minute,
			minLead:    10 * time.Minute,
			expectedAt: "2026-08-05T10:30:00Z",
		},
		{
			name:       "cross midnight",
			now:        "2026-08-05T23:45:00Z",
			interval:   30 * time.Minute,
			minLead:    10 * time.Minute,
			expectedAt: "2026-08-06T00:00:00Z",
		},
		{
			name:       "DST spring-forward instant",
			now:        "2026-03-08T01:55:00-05:00",
			interval:   30 * time.Minute,
			minLead:    10 * time.Minute,
			expectedAt: "2026-03-08T07:30:00Z",
		},
		{
			name:       "DST fall-back instant",
			now:        "2026-11-01T01:55:00-04:00",
			interval:   30 * time.Minute,
			minLead:    10 * time.Minute,
			expectedAt: "2026-11-01T06:30:00Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339Nano, tt.now)
			require.NoError(t, err)
			expected, err := time.Parse(time.RFC3339Nano, tt.expectedAt)
			require.NoError(t, err)

			actual := nextScheduledPostBatchBoundary(now, tt.interval, tt.minLead)

			require.Equal(t, expected.UnixMilli(), actual)
			require.GreaterOrEqual(t, time.UnixMilli(actual).Sub(now), tt.minLead)
			require.Zero(t, actual%tt.interval.Milliseconds())
		})
	}
}

func TestScheduledPostTime(t *testing.T) {
	tests := []struct {
		name         string
		batchPercent float64
		batchAligned bool
	}{
		{
			name:         "batch aligned",
			batchPercent: 1,
			batchAligned: true,
		},
		{
			name:         "long-range",
			batchPercent: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{
				PercentBatchAlignedScheduledPosts: tt.batchPercent,
				ScheduledPostBatchIntervalMinutes: 45,
				ScheduledPostBatchMinLeadMinutes:  17,
			}
			controller := &SimulController{config: config}
			now := time.Now()

			actual := controller.scheduledPostTime(now)

			if tt.batchAligned {
				interval := time.Duration(config.ScheduledPostBatchIntervalMinutes) * time.Minute
				minLead := time.Duration(config.ScheduledPostBatchMinLeadMinutes) * time.Minute
				require.Equal(t, nextScheduledPostBatchBoundary(now, interval, minLead), actual)
				return
			}

			require.GreaterOrEqual(t, actual, now.Add(scheduledPostFutureTimeDeltaStart).UnixMilli())
			require.LessOrEqual(t, actual, now.Add(scheduledPostFutureTimeDeltaStart+scheduledPostFutureTimeMaxUntil).UnixMilli())
		})
	}
}
