// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package store

import (
	"github.com/mattermost/mattermost-server/v5/model"
)

type UserStore interface {
	Id() string
	// TODO: Move all getters to this interface

	// Config return the server configuration settings
	Config() model.Config
	// Channels return the channels for a team.
	Channels(teamId string) ([]model.Channel, error)
	// Teams return the teams a user belong to.
	Teams() ([]model.Team, error)
	// Preferences return the preferences of the user.
	Preferences() (model.Preferences, error)
	// Roles return the roles of the user.
	Roles() ([]model.Role, error)
}

type MutableUserStore interface {
	UserStore

	// server
	SetConfig(*model.Config)

	// users
	SetUser(user *model.User) error
	User() (*model.User, error)
	SetUsers(users []*model.User) error
	Users() ([]*model.User, error)

	// posts
	SetPost(post *model.Post) error
	SetPosts(posts []*model.Post) error
	Post(postId string) (*model.Post, error)
	ChannelPosts(channelId string) ([]*model.Post, error)
	SetReactions(postId string, reactions []*model.Reaction) error
	Reactions(postId string) ([]*model.Reaction, error)

	// preferences
	SetPreferences(preferences *model.Preferences) error

	// channels
	SetChannel(channel *model.Channel) error
	SetChannels(channels []*model.Channel) error
	Channel(channelId string) (*model.Channel, error)
	// SetChannelMembers stores the given channel members in the store.
	SetChannelMembers(channelMembers *model.ChannelMembers) error
	ChannelMembers(channelId string) (*model.ChannelMembers, error)
	SetChannelMember(channelId string, channelMember *model.ChannelMember) error
	ChannelMember(channelId, userId string) (*model.ChannelMember, error)
	RemoveChannelMember(channelId string, userId string) error

	// teams
	SetTeam(team *model.Team) error
	Team(teamId string) (*model.Team, error)
	SetTeams(teams []*model.Team) error
	SetTeamMember(teamId string, teamMember *model.TeamMember) error
	RemoveTeamMember(teamId, memberId string) error
	SetTeamMembers(teamId string, teamMember []*model.TeamMember) error
	TeamMember(teamdId, userId string) (*model.TeamMember, error)

	// roles
	// SetRoles stores the given roles.
	SetRoles(roles []*model.Role) error

	// emoji
	SetEmojis(emoji []*model.Emoji) error
}
