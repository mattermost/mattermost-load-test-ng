// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package control

import (
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

// User status types.
const (
	USER_STATUS_UNKNOWN int = iota
	USER_STATUS_STARTED
	USER_STATUS_STOPPED
	USER_STATUS_DONE
	USER_STATUS_ERROR
	USER_STATUS_FAILED
	USER_STATUS_INFO
)

// UserStatus contains the status of an action performed by a user.
type UserStatus struct {
	// ControllerId is the id of the controller.
	ControllerId int
	// User is the user who is performing the action.
	User user.User
	// Code is an integer code of the status returned.
	Code int
	// Info contains any extra information attached with the status.
	Info string
	// Custom error containing the error encountered and location information.
	Err error
}
