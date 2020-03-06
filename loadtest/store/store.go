// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package store

import (
	"github.com/mattermost/mattermost-server/v5/model"
)

// UserStore is a read-only interface which provides access to various
// data belonging to a user.
type UserStore interface {
	// Id of the user.
	Id() string
	// Username of the user.
	Username() string
	// Email of the user.
	Email() string
	// Password of the user.
	Password() string

	// TODO: Move all getters to this interface

	// Config returns the server configuration settings.
	Config() model.Config
	// Channel returns the channel for a given channelId.
	Channel(channelId string) (*model.Channel, error)
	// Channels returns the channels for a team.
	Channels(teamId string) ([]model.Channel, error)
	// ChannelMember returns the ChannelMember for the given channelId and userId.
	ChannelMember(channelId, userId string) (model.ChannelMember, error)
	// ChannelPosts returns all posts for given channelId
	ChannelPosts(channelId string) ([]*model.Post, error)
	// ChannelPostsSorted returns all posts for given channelId, sorted by CreateAt
	ChannelPostsSorted(channelId string, asc bool) ([]*model.Post, error)
	// Teams returns the teams a user belong to.
	Teams() ([]model.Team, error)
	// TeamMember returns the TeamMember for the given teamId and userId.
	TeamMember(teamdId, userId string) (model.TeamMember, error)
	// Preferences returns the preferences of the user.
	Preferences() (model.Preferences, error)
	// Roles returns the roles of the user.
	Roles() ([]model.Role, error)

	// PostsSince returns posts created after a specified timestamp in milliseconds.
	PostsSince(ts int64) ([]model.Post, error)

	// Reactions returns reactions for a given postId.
	Reactions(postId string) ([]model.Reaction, error)

	// Random things
	// RandomChannel returns a random channel for a user.
	RandomChannel(teamId string) (model.Channel, error)
	// RandomChannelJoined returns a random channel for the given teamId that the
	// current user is a member of.
	RandomChannelJoined(teamId string) (model.Channel, error)
	// RandomTeam returns a random team for a user.
	RandomTeam() (model.Team, error)
	// RandomTeamJoined returns a random team the current user is a member of.
	RandomTeamJoined() (model.Team, error)
	// RandomUser returns a random user from the set of users.
	RandomUser() (model.User, error)
	// RandomUsers returns N random users from the set of users.
	RandomUsers(n int) ([]model.User, error)
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

	// Clear resets the store and removes all entries
	Clear()

	// server
	SetConfig(*model.Config)

	// users
	SetUser(user *model.User) error
	User() (*model.User, error)
	SetUsers(users []*model.User) error
	Users() ([]*model.User, error)

	// posts
	SetPost(post *model.Post) error
	DeletePost(postId string) error
	SetPosts(posts []*model.Post) error
	Post(postId string) (*model.Post, error)

	// reactions
	SetReactions(postId string, reactions []*model.Reaction) error
	SetReaction(reaction *model.Reaction) error
	DeleteReaction(reaction *model.Reaction) (bool, error)

	// preferences
	SetPreferences(preferences *model.Preferences) error

	// channels
	SetChannel(channel *model.Channel) error
	SetChannels(channels []*model.Channel) error
	// SetChannelMembers stores the given channel members in the store.
	SetChannelMembers(channelMembers *model.ChannelMembers) error
	ChannelMembers(channelId string) (*model.ChannelMembers, error)
	SetChannelMember(channelId string, channelMember *model.ChannelMember) error
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
