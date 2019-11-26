// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package user

import (
	"github.com/mattermost/mattermost-load-test/loadtest/store"
)

const (
	STATUS_STARTED int = iota
	STATUS_STOPPED int = iota
	STATUS_DONE    int = iota
	STATUS_ERROR   int = iota
	STATUS_FAILED  int = iota
)

type User interface {
	Id() int
	Store() store.UserStore

	SignUp(email, username, password string) error
	Login() error
	Logout() error
}

type UserStatus struct {
	User User
	Code int
	Info string
	Err  error
}
