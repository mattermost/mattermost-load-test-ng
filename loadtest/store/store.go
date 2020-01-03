// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package store

import (
	"github.com/mattermost/mattermost-server/v5/model"
)

// UserStore is a read-only interface which provides access to various
// data belonging to a user.
type UserStore interface {
	// Id of the user.
	Id() string
	// TODO: Move all getters to this interface

	// Config return the server configuration settings
	Config() model.Config
	// Channels return the channels for a team.
	Channels(teamId string) ([]model.Channel, error)
	// Teams return the teams a user belong to.
	Teams() ([]model.Team, error)
	// TeamMember returns the TeamMember for the given teamId and userId
	TeamMember(teamdId, userId string) (model.TeamMember, error)
	// Preferences return the preferences of the user.
	Preferences() (model.Preferences, error)
	// Roles return the roles of the user.
	Roles() ([]model.Role, error)

	// Random things
	// RandomChannel returns a random channel for a user.
	RandomChannel(teamId string) (model.Channel, error)
	// RandomTeam returns a random team for a user.
	RandomTeam() (model.Team, error)
	// RandomUser returns a random user from the set of users.
	RandomUser() (model.User, error)
	// RandomPost returns a random post.
	RandomPost() (model.Post, error)
	// RandomEmoji returns a random emoji.
	RandomEmoji() (model.Emoji, error)
	// RandomChannelMember returns a random channel member for a channel.
	RandomChannelMember(channelId string) (model.ChannelMember, error)
	// RandomTeamMember returns a random team member for a team.
	RandomTeamMember(teamId string) (model.TeamMember, error)
}

// MutableUserStore is a super-set of UserStore which, apart from providing
// read access, also allows to edit the data of a user.
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

	// roles
	// SetRoles stores the given roles.
	SetRoles(roles []*model.Role) error

	// emoji
	SetEmojis(emoji []*model.Emoji) error

	// license
	// SetLicense stores the given license in the store.
	SetLicense(license map[string]string) error
}
