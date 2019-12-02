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

	// connection
	Connect() error
	Disconnect() error
	SignUp(email, username, password string) error
	Login() error
	Logout() (bool, error)

	// posts
	CreatePost(post *model.Post) (string, error)

	// channels

	CreateGroupChannel(memberIds []string) (string, error)
	ViewChannel(view *model.ChannelView) (*model.ChannelViewResponse, error)
	GetChannelUnread(channelId string) (*model.ChannelUnread, error)
	GetChannelMembers(channelId string, page, perPage int) (*model.ChannelMembers, error)

	// teams
}

type UserStatus struct {
	User User
	Code int
	Info string
	Err  error
}
