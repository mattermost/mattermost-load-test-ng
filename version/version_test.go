// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package version

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetInfo(t *testing.T) {
	info := GetInfo()

	// When running in test mode, we expect default values
	require.Equal(t, "unknown", info.Commit, "Commit should be 'unknown' in test environment")
	require.False(t, info.Modified, "Modified should be false in test environment")
	require.True(t, info.BuildTime.IsZero(), "BuildTime should be zero in test environment")
}

func TestVersionInfoString(t *testing.T) {
	// Test with a known VersionInfo
	now := time.Now()
	info := VersionInfo{
		Commit:    "abc123",
		BuildTime: now,
		Modified:  false,
	}

	str := info.String()

	// Check that the string contains the expected information
	require.Contains(t, str, "Commit: abc123")
	require.Contains(t, str, "Build Time: "+now.Format("2006-01-02 15:04:05"))
	require.NotContains(t, str, "(modified)")

	// Test with modified flag set to true
	info.Modified = true
	str = info.String()
	require.Contains(t, str, "(modified)")

	// Test with zero build time
	info.BuildTime = time.Time{}
	str = info.String()
	require.Contains(t, str, "Build Time: unknown")
}
