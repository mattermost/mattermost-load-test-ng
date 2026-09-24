// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"maps"
	"slices"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
)

// defaultFeatureFlags are the feature flags always set on the servers.
var defaultFeatureFlags = map[string]string{
	"PostPriority":            "true",
	"WebSocketEventScope":     "true",
	"ChannelBookmarks":        "true",
	"CustomProfileAttributes": "true",
}

// accessControlFeatureFlags are the feature flags set on the servers when
// AccessControlSettings.Enable is true. SessionAttributes needs to be set at
// startup, since the server registers the properties API routes only if it's
// enabled.
var accessControlFeatureFlags = map[string]string{
	"PermissionPolicies": "true",
	"SessionAttributes":  "true",
}

// featureFlagsEnv returns the environment variable assignments setting the
// feature flags on the servers, sorted by name.
func featureFlagsEnv(cfg *deployment.Config) []string {
	flags := maps.Clone(defaultFeatureFlags)
	if cfg.AccessControlSettings.Enable {
		maps.Copy(flags, accessControlFeatureFlags)
	}
	maps.Copy(flags, cfg.MattermostFeatureFlags)

	env := make([]string, 0, len(flags))
	for _, name := range slices.Sorted(maps.Keys(flags)) {
		env = append(env, "MM_FEATUREFLAGS_"+strings.ToUpper(name)+"="+flags[name])
	}
	return env
}
