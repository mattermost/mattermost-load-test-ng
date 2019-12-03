// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package control

import (
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

type UserController interface {
	Init(user user.User)
	Run(status chan<- UserStatus)
	SetRate(rate float64) error
	Stop()
}
