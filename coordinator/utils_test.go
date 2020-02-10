// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
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
