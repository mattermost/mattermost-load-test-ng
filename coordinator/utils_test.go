// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestHasPassed(t *testing.T) {
	tm := time.Now()
	require.False(t, hasPassed(tm, 1*time.Second))
	time.Sleep(1 * time.Second)
	require.True(t, hasPassed(tm, 500*time.Millisecond))
	require.False(t, hasPassed(tm, 2*time.Second))
}

func TestMin(t *testing.T) {
	require.Equal(t, 0, min(0, 1))
	require.Equal(t, 0, min(1, 0))
	require.Equal(t, 0, min(0, 0))
	require.Equal(t, 1, min(1, 1))
	require.Equal(t, 50, min(80, 50))
	require.Equal(t, 30, min(100, 30))
}

func TestGetLatestSamples(t *testing.T) {
	samples := []point{}
	require.Empty(t, getLatestSamples(samples, 1*time.Minute))

	samples = []point{
		{time.Unix(0, 0), 0},
		{time.Unix(1, 0), 4},
		{time.Unix(2, 0), 8},
	}
	expected := []point{
		{time.Unix(1, 0), 4},
		{time.Unix(2, 0), 8},
	}
	require.Equal(t, expected, getLatestSamples(samples, 1*time.Second))

	samples = []point{
		{time.Unix(0, 0), 0},
		{time.Unix(10, 0), 1},
		{time.Unix(20, 0), 2},
		{time.Unix(30, 0), 3},
		{time.Unix(40, 0), 4},
	}
	require.Equal(t, samples, getLatestSamples(samples, 40*time.Second))

	samples = []point{
		{time.Unix(0, 0), 0},
		{time.Unix(10, 0), 1},
		{time.Unix(20, 0), 2},
		{time.Unix(30, 0), 3},
		{time.Unix(40, 0), 4},
	}
	expected = []point{
		{time.Unix(20, 0), 2},
		{time.Unix(30, 0), 3},
		{time.Unix(40, 0), 4},
	}
	require.Equal(t, expected, getLatestSamples(samples, 20*time.Second))
}
