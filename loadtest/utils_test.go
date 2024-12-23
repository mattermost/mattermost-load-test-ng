// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package loadtest

import (
	"testing"
	"time"
)

func TestRandomFutureTime(t *testing.T) {
	deltaStart := 10 * time.Second
	maxUntil := 5 * time.Minute

	now := time.Now()
	start := now.Add(deltaStart)
	end := start.Add(maxUntil)

	randomTime := RandomFutureTime(deltaStart, maxUntil)

	if randomTime < start.Unix() || randomTime > end.Unix() {
		t.Errorf("RandomFutureTime() = %v, want between %v and %v", randomTime, start.Unix(), end.Unix())
	}
}

func TestRandomFutureTimeZeroDuration(t *testing.T) {
	deltaStart := 0 * time.Second
	maxUntil := 0 * time.Second

	now := time.Now()
	expectedTime := now.Unix()

	randomTime := RandomFutureTime(deltaStart, maxUntil)

	if randomTime != expectedTime {
		t.Errorf("RandomFutureTime() = %v, want %v", randomTime, expectedTime)
	}
}

func TestRandomFutureTimeNegativeDuration(t *testing.T) {
	deltaStart := -10 * time.Second
	maxUntil := -5 * time.Minute

	now := time.Now()
	start := now.Add(deltaStart)
	end := start.Add(maxUntil)

	randomTime := RandomFutureTime(deltaStart, maxUntil)

	if randomTime < end.Unix() || randomTime > start.Unix() {
		t.Errorf("RandomFutureTime() = %v, want between %v and %v", randomTime, end.Unix(), start.Unix())
	}
}

func TestRandomFutureTimeMaxUntilZero(t *testing.T) {
	deltaStart := 10 * time.Second
	maxUntil := 0 * time.Second

	now := time.Now()
	expectedTime := now.Add(deltaStart).Unix()

	randomTime := RandomFutureTime(deltaStart, maxUntil)

	if randomTime != expectedTime {
		t.Errorf("RandomFutureTime() = %v, want %v", randomTime, expectedTime)
	}
}

func TestRandomFutureTimeDeltaStartZero(t *testing.T) {
	deltaStart := 0 * time.Second
	maxUntil := 5 * time.Minute

	now := time.Now()
	start := now
	end := start.Add(maxUntil)

	randomTime := RandomFutureTime(deltaStart, maxUntil)

	if randomTime < start.Unix() || randomTime > end.Unix() {
		t.Errorf("RandomFutureTime() = %v, want between %v and %v", randomTime, start.Unix(), end.Unix())
	}
}

func TestRandomFutureTimeLargeDurations(t *testing.T) {
	deltaStart := 100 * time.Hour
	maxUntil := 1000 * time.Hour

	now := time.Now()
	start := now.Add(deltaStart)
	end := start.Add(maxUntil)

	randomTime := RandomFutureTime(deltaStart, maxUntil)

	if randomTime < start.Unix() || randomTime > end.Unix() {
		t.Errorf("RandomFutureTime() = %v, want between %v and %v", randomTime, start.Unix(), end.Unix())
	}
}
