// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package loadtest

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestRandomFutureTimeSuite(t *testing.T) {
	now := time.Now().UnixMilli()
	tests := []struct {
		name        string
		deltaStart  time.Duration
		maxUntil    time.Duration
		expectedMin int64
		expectedMax int64
	}{
		{
			name:       "Positive Durations",
			deltaStart: 10 * time.Second,
			maxUntil:   5 * time.Minute,
		},
		{
			name:        "Zero Durations",
			deltaStart:  0 * time.Second,
			maxUntil:    0 * time.Second,
			expectedMin: now,
			expectedMax: now,
		},
		{
			name:       "Negative Durations",
			deltaStart: -10 * time.Second,
			maxUntil:   -5 * time.Minute,
		},
		{
			name:       "MaxUntil Zero",
			deltaStart: 10 * time.Second,
			maxUntil:   0 * time.Second,
		},
		{
			name:       "DeltaStart Zero",
			deltaStart: 0 * time.Second,
			maxUntil:   5 * time.Minute,
		},
		{
			name:       "Large Durations",
			deltaStart: 100 * time.Hour,
			maxUntil:   1000 * time.Hour,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := time.Now()
			start := now.Add(tt.deltaStart)
			end := start.Add(tt.maxUntil)

			randomTime := RandomFutureTime(tt.deltaStart, tt.maxUntil)

			if tt.expectedMin != 0 && tt.expectedMax != 0 {
				require.LessOrEqual(t, tt.expectedMin, randomTime)
			} else {
				// checking both ways to allow for negative values
				isBetweenBounds := (randomTime >= start.UnixMilli() && randomTime <= end.UnixMilli()) || (randomTime <= start.UnixMilli() && randomTime >= end.UnixMilli())

				require.True(t, isBetweenBounds, fmt.Sprintf("RandomFutureTime() = %v, want between %v and %v", randomTime, start.UnixMilli(), end.UnixMilli()))
			}
		})
	}
}
