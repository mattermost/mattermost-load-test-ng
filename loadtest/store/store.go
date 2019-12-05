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
	SetUser(user *model.User) error
	SetPost(post *model.Post) error
	SetPreferences(preferences model.Preferences) error
	SetChannel(channel *model.Channel) error
	Post(postId string) (*model.Post, error)
	Preferences() (model.Preferences, error)
	User() (*model.User, error)
	Channel(channelId string) (*model.Channel, error)
	SetChannelMembers(channelId string, channelMembers *model.ChannelMembers) error
	ChannelMembers(channelId string) (*model.ChannelMembers, error)
	SetChannelMember(channelId string, channelMember *model.ChannelMember) error
	ChannelMember(channelId, userId string) (*model.ChannelMember, error)
	SetTeamMember(teamId string, teamMember *model.TeamMember) error
	TeamMember(teamdId, userId string) (*model.TeamMember, error)
}
