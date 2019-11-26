// Copyright (c) 2019 Mattermost, Inc. All Rights Reserved.
// See License.txt for license information

package control

import (
	"github.com/mattermost/mattermost-load-test/loadtest/user"
)

type UserController interface {
	Init(user user.User)
	Run(status chan<- user.UserStatus)
	Stop()
}
