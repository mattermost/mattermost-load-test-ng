// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package control

import (
	"testing"

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
