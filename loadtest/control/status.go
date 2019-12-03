// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package control

import (
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

const (
	USER_STATUS_UNKNOWN int = iota
	USER_STATUS_STARTED
	USER_STATUS_STOPPED
	USER_STATUS_DONE
	USER_STATUS_ERROR
	USER_STATUS_FAILED
	USER_STATUS_INFO
)

type UserStatus struct {
	User user.User
	Code int
	Info string
	Err  error
}
