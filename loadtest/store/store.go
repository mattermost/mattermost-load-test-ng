// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package store

import (
	"github.com/mattermost/mattermost-server/v5/model"
)

// SelectionType is the selection parameter for a store entity.
type SelectionType uint8

// Defines the membership rules for a store entity.
const (
	SelectMemberOf    SelectionType = 1 << iota // Select all cases where the user is a member.
	SelectNotMemberOf                           // Select cases where the user is not a member.
	SelectNotCurrent                            // When set the selection won't return current team/channel.

	SelectAny = SelectMemberOf | SelectNotMemberOf
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
	// CurrentChannel gets the channel the user is currently viewing.
	CurrentChannel() (*model.Channel, error)
	// ChannelMember returns the ChannelMember for the given channelId and userId.
	ChannelMember(channelId, userId string) (model.ChannelMember, error)
	// ChannelPosts returns all posts for given channelId
	ChannelPosts(channelId string) ([]*model.Post, error)
	// ChannelPostsSorted returns all posts for given channelId, sorted by CreateAt
	ChannelPostsSorted(channelId string, asc bool) ([]*model.Post, error)
	// ChannelView return the timestamp of the last view for the given channelId.
	ChannelView(channelId string) (int64, error)

	// Teams returns the teams a user belong to.
	Teams() ([]model.Team, error)
	// CurrentTeam gets the currently selected team for the user.
	CurrentTeam() (*model.Team, error)
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

	// RandomChannel returns a random channel for the given teamId for the
	// current user.
	RandomChannel(teamId string, st SelectionType) (model.Channel, error)
	// RandomTeam returns a random team for the current user.
	RandomTeam(st SelectionType) (model.Team, error)
	// RandomUser returns a random user from the set of users.
	RandomUser() (model.User, error)
	// RandomUsers returns N random users from the set of users.
	RandomUsers(n int) ([]model.User, error)
	// RandomPost returns a random post.
	RandomPost() (model.Post, error)
	// RandomPostForChannel returns a random post for the given channel.
	RandomPostForChannel(channelId string) (model.Post, error)
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
	// SetCurrentChannel sets the channel the user is currently viewing.
	SetCurrentChannel(channel *model.Channel) error
	// SetChannelView marks a channel as viewed and updates the store with the
	// current timestamp.
	SetChannelView(channelId string) error
	// SetChannelMembers stores the given channel members in the store.
	SetChannelMembers(channelMembers *model.ChannelMembers) error
	ChannelMembers(channelId string) (*model.ChannelMembers, error)
	SetChannelMember(channelId string, channelMember *model.ChannelMember) error
	RemoveChannelMember(channelId string, userId string) error

	// teams
	SetTeam(team *model.Team) error
	Team(teamId string) (*model.Team, error)
	SetTeams(teams []*model.Team) error
	// SetCurrentTeam sets the currently selected team for the user.
	SetCurrentTeam(team *model.Team) error
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
