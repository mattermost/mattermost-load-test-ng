// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package store

import (
	"github.com/mattermost/mattermost-server/model"
)

type UserStore interface {
	Id() string
	Post(postId string) (*model.Post, error)
	User() (*model.User, error)
	Channel(channelId string) (*model.Channel, error)
}

type MutableUserStore interface {
	UserStore
	SetUser(user *model.User) error
	SetPost(post *model.Post) error
	SetChannel(channel *model.Channel) error
}
