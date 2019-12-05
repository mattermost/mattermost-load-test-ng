// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package user

import (
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-server/model"
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

	// user
	GetMe() (string, error)
	GetPreferences() error
	CreateUser(user *model.User) (string, error)

	// posts
	CreatePost(post *model.Post) (string, error)

	// channels
	CreateChannel(channel *model.Channel) (string, error)
	CreateGroupChannel(memberIds []string) (string, error)
	CreateDirectChannel(otherUserId string) (string, error)
	ViewChannel(view *model.ChannelView) (*model.ChannelViewResponse, error)
	GetChannelUnread(channelId string) (*model.ChannelUnread, error)
	GetChannelMembers(channelId string, page, perPage int) error
	GetChannelMember(channelId string, userId string) error
	GetChannelStats(channelId string) error

	// teams
	CreateTeam(team *model.Team) (string, error)
}
