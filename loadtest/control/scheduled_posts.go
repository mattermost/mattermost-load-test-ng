// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package control

import (
	"math/rand"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost/server/public/model"
)

// RecurringScheduledPostsMinVersion is the minimum server version that supports
// recurring scheduled posts.
var RecurringScheduledPostsMinVersion = UnreleasedVersion

var recurringScheduledPostFallbackTimezones = []string{
	"UTC",
	"America/New_York",
	"Europe/London",
	"Asia/Tokyo",
}

func recurringScheduledPostTimezoneForUser(currentUser *model.User) string {
	timezone := currentUser.GetPreferredTimezone()
	if timezone != "" && timezone != "Local" {
		if _, err := time.LoadLocation(timezone); err == nil {
			return timezone
		}
	}

	return recurringScheduledPostFallbackTimezones[rand.Intn(len(recurringScheduledPostFallbackTimezones))]
}

// RecurringScheduledPostTimezone returns the user's valid preferred timezone,
// or a valid fallback timezone when no preference can be used.
func RecurringScheduledPostTimezone(u user.User) string {
	currentUser, err := u.Store().User()
	if err != nil {
		return recurringScheduledPostFallbackTimezones[rand.Intn(len(recurringScheduledPostFallbackTimezones))]
	}

	return recurringScheduledPostTimezoneForUser(currentUser)
}
