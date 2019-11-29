// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package user

import (
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-server/model"
)

const (
	STATUS_UNKNOWN int = iota
	STATUS_STARTED
	STATUS_STOPPED
	STATUS_DONE
	STATUS_ERROR
	STATUS_FAILED
)

type User interface {
	Id() int
	Store() store.UserStore

	Connect() error
	Disconnect() error
	SignUp(email, username, password string) error
	Login() error
	Logout() (bool, error)
	CreatePost(post *model.Post) error
}

type UserStatus struct {
	User User
	Code int
	Info string
	Err  error
}
