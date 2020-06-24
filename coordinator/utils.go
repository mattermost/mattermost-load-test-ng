// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"time"
)

// hasPassed reports whether the provided duration
// added with the given time is before the current time or not.
func hasPassed(t time.Time, d time.Duration) bool {
	return time.Now().After(t.Add(d))
}

// min finds the minimum between the provided int values.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getLatestSamples returns all the samples in the time range
// [lastSampleTime-d, lastSampleTime].
func getLatestSamples(samples []point, d time.Duration) []point {
	var k int
	if len(samples) == 0 {
		return samples
	}
	last := samples[len(samples)-1]
	for i := len(samples) - 1; i >= 0; i-- {
		if last.x.Sub(samples[i].x) >= d {
			k = i
			break
		}
	}
	return samples[k:]
}
