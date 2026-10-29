// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package control

import (
	"testing"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/stretchr/testify/require"
)

func TestFeatureFlagEnabled(t *testing.T) {
	store, err := memstore.New(nil)
	require.NoError(t, err)
	store.SetClientConfig(map[string]string{
		"FeatureFlagEnabledFlag":  "true",
		"FeatureFlagDisabledFlag": "false",
		"UnrelatedSetting":        "false",
	})

	u := userentity.New(userentity.Setup{Store: store}, userentity.Config{
		ServerURL:    "http://localhost:8065",
		WebSocketURL: "ws://localhost:8065",
	})

	require.True(t, FeatureFlagEnabled(u, "EnabledFlag"))
	require.False(t, FeatureFlagEnabled(u, "DisabledFlag"))
	// The server does not send flags that were removed when their feature
	// became generally available.
	require.True(t, FeatureFlagEnabled(u, "RemovedFlag"))
	require.True(t, FeatureFlagEnabled(u, "UnrelatedSetting"))

	for _, check := range []func() (bool, UserActionResponse){
		func() (bool, UserActionResponse) { return ChannelBookmarkEnabled(u) },
		func() (bool, UserActionResponse) { return CustomProfileAttributesEnabled(u) },
	} {
		enabled, resp := check()
		require.NoError(t, resp.Err)
		require.True(t, enabled)
	}

	store.SetClientConfig(map[string]string{
		"FeatureFlagChannelBookmarks":        "false",
		"FeatureFlagCustomProfileAttributes": "false",
	})
	enabled, _ := ChannelBookmarkEnabled(u)
	require.False(t, enabled)
	enabled, _ = CustomProfileAttributesEnabled(u)
	require.False(t, enabled)
}

func TestFeatureFlagEnabledByServerVersion(t *testing.T) {
	store, err := memstore.New(nil)
	require.NoError(t, err)
	// None of the flags is sent, as happens once they are removed.
	store.SetClientConfig(map[string]string{})

	u := userentity.New(userentity.Setup{Store: store}, userentity.Config{
		ServerURL:    "http://localhost:8065",
		WebSocketURL: "ws://localhost:8065",
	})

	tcs := []struct {
		version  string
		flag     string
		expected bool
	}{
		{"10.11.0", "ChannelBookmarks", false},
		{"11.11.0", "ChannelBookmarks", false},
		{"12.0.0", "ChannelBookmarks", true},
		{"12.0.1", "ChannelBookmarks", true},
		{"12.0.0-rc1", "ChannelBookmarks", true},
		{"10.11.0", "CustomProfileAttributes", false},
		{"11.10.0", "CustomProfileAttributes", false},
		{"11.11.0", "CustomProfileAttributes", true},
		{"10.11.0", "WebSocketEventScope", false},
		{"11.9.0", "WebSocketEventScope", false},
		{"11.10.0", "WebSocketEventScope", true},
		{"11.10.0-rc2", "WebSocketEventScope", true},
		{"10.11.0", "UnlistedFlag", true},
	}
	for _, tc := range tcs {
		t.Run(tc.flag+" on "+tc.version, func(t *testing.T) {
			require.NoError(t, store.SetServerVersion(semver.MustParse(tc.version)))
			require.Equal(t, tc.expected, FeatureFlagEnabled(u, tc.flag))
		})
	}

	// A flag that the server sends keeps its value whatever the version.
	require.NoError(t, store.SetServerVersion(semver.MustParse("10.11.0")))
	store.SetClientConfig(map[string]string{"FeatureFlagChannelBookmarks": "true"})
	require.True(t, FeatureFlagEnabled(u, "ChannelBookmarks"))

	// An unknown server version does not disable the features.
	require.NoError(t, store.SetServerVersion(semver.Version{}))
	store.SetClientConfig(map[string]string{})
	require.True(t, FeatureFlagEnabled(u, "ChannelBookmarks"))
}
