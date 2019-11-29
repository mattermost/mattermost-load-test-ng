// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package store

import (
	"github.com/mattermost/mattermost-server/model"
)

type UserStore interface {
	Id() string
}

type MutableUserStore interface {
	UserStore
	User() *model.User
	SetUser(user *model.User) error
	Post(postId string) (*model.Post, error)
	SetPost(post *model.Post) error
}
