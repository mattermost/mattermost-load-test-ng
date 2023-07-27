// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package store

import (
	"github.com/mattermost/mattermost/server/public/model"
)

// SelectionType is the selection parameter for a store entity.
type SelectionType uint8

// Defines the membership rules for a store entity.
const (
	SelectMemberOf    SelectionType = 1 << iota // Select all cases where the user is a member.
	SelectNotMemberOf                           // Select cases where the user is not a member.
	SelectNotCurrent                            // When set the selection won't return current team/channel.
	SelectNotPublic                             // Don't include public channels in selection.
	SelectNotPrivate                            // Don't include private channels in selection.
	SelectNotDirect                             // Don't include direct channels in selection.
	SelectNotGroup                              // Don't include group channels in selection.

	SelectAny = SelectMemberOf | SelectNotMemberOf
)

// UserStore is a read-only interface which provides access to various
// data belonging to a user.
type UserStore interface {
	// Id returns the id for the stored user.
	Id() string
	// Username returns the username for the stored user.
	Username() string
	// Email returns the email for the stored user.
	Email() string
	// Password returns the password for the stored user.
	Password() string

	// Config returns the server configuration settings.
	Config() model.Config
	// ClientConfig returns the partial server configuration settings for logged in user.
	ClientConfig() map[string]string
	// Channel returns the channel for the given channelId.
	Channel(channelId string) (*model.Channel, error)
	// Channels returns the channels for a team.
	Channels(teamId string) ([]model.Channel, error)
	// CurrentChannel gets the channel the user is currently viewing.
	CurrentChannel() (*model.Channel, error)
	// ChannelMember returns the ChannelMember for the given channelId and userId.
	ChannelMember(channelId, userId string) (model.ChannelMember, error)
	// ChannelMembers returns a list of members for the specified channel.
	ChannelMembers(channelId string) (model.ChannelMembers, error)
	// ChannelPosts returns all posts for the specified channel.
	ChannelPosts(channelId string) ([]*model.Post, error)
	// ChannelPostsSorted returns all posts for specified channel, sorted by CreateAt.
	ChannelPostsSorted(channelId string, asc bool) ([]*model.Post, error)
	// ChannelView returns the timestamp of the last view for the given channelId.
	ChannelView(channelId string) (int64, error)
	// ChannelStats returns statistics for the given channelId.
	ChannelStats(channelId string) (*model.ChannelStats, error)

	// GetUser returns the user for the given userId.
	GetUser(userId string) (model.User, error)
	// Users returns all users in the store.
	Users() ([]model.User, error)

	// Status returns the status for the given userId.
	Status(userId string) (model.Status, error)

	// teams
	// Teams returns the teams a user belong to.
	Teams() ([]model.Team, error)
	// CurrentTeam gets the currently selected team for the user.
	CurrentTeam() (*model.Team, error)
	// TeamMember returns the TeamMember for the given teamId and userId.
	TeamMember(teamdId, userId string) (model.TeamMember, error)
	// IsTeamMember returns if the user is part of the team.
	IsTeamMember(teamdId, userId string) bool

	// Preferences returns the preferences for the stored user.
	Preferences() (model.Preferences, error)
	// Roles returns the roles of the user.
	Roles() ([]model.Role, error)

	// Reactions returns the reactions for the specified post.
	Reactions(postId string) ([]model.Reaction, error)

	// random utils
	// RandomChannel returns a random channel for the given teamId
	// for the current user.
	RandomChannel(teamId string, st SelectionType) (model.Channel, error)
	// RandomTeam returns a random team for the current user.
	RandomTeam(st SelectionType) (model.Team, error)
	// RandomUser returns a random user from the set of users.
	RandomUser() (model.User, error)
	// RandomUsers returns N random users from the set of users.
	RandomUsers(n int) ([]model.User, error)
	// RandomPost returns a random post, whose channel will satisfy
	// the constraints provided by st
	RandomPost(st SelectionType) (model.Post, error)
	// RandomPostForChannel returns a random post for the given channel.
	RandomPostForChannel(channelId string) (model.Post, error)
	// RandomReplyPostForChannel returns a random reply post for the given channel.
	RandomReplyPostForChannel(channelId string) (model.Post, error)
	// RandomPostForChanneByUser returns a random post for the given channel made
	// by the given user.
	RandomPostForChannelByUser(channelId, userId string) (model.Post, error)
	// RandomEmoji returns a random emoji.
	RandomEmoji() (model.Emoji, error)
	// RandomChannelMember returns a random channel member for a channel.
	RandomChannelMember(channelId string) (model.ChannelMember, error)
	// RandomTeamMember returns a random team member for a team.
	RandomTeamMember(teamId string) (model.TeamMember, error)
	// RandomThread returns a random thread.
	RandomThread() (model.ThreadResponse, error)
	// RandomCategory returns a random category from a team
	RandomCategory(teamID string) (model.SidebarCategoryWithChannels, error)

	// profile
	// ProfileImage returns whether the profile image for the given user has been
	// stored.
	ProfileImage(userId string) (bool, error)

	// posts
	// Post returns the post for the given postId.
	Post(postId string) (*model.Post, error)
	// UserForPost returns the userId for the user who created the specified post.
	UserForPost(postId string) (string, error)
	// FileInfoForPost returns the FileInfo for the specified post, if any.
	FileInfoForPost(postId string) ([]*model.FileInfo, error)
	// PostsIdsSince returns a list of post ids for posts created
	// after a specified timestamp in milliseconds.
	PostsIdsSince(ts int64) ([]string, error)

	// ServerVersion returns the server version string.
	ServerVersion() (string, error)

	// Threads
	Thread(threadId string) (*model.ThreadResponse, error)
	// ThreadsSorted returns all threads, sorted by LastReplyAt
	ThreadsSorted(unreadOnly, asc bool) ([]*model.ThreadResponse, error)

	// PostsWithAckRequests returns IDs of the posts that asked for acknowledgment.
	PostsWithAckRequests() ([]string, error)
}

// MutableUserStore is a super-set of UserStore which, apart from providing
// read access, also allows to edit the data of a user.
type MutableUserStore interface {
	UserStore

	// Clear resets the store and removes all entries with the exception of the
	// user object and state information (current team/channel) which are preserved.
	Clear()

	// server
	// SetConfig stores the given configuration settings.
	SetConfig(*model.Config)
	// SetClientConfig stores the given client configuration settings.
	SetClientConfig(map[string]string)

	// users
	// SetUser stores the given user.
	SetUser(user *model.User) error
	// User returns the stored user.
	User() (*model.User, error)
	// SetUsers stores the given users.
	SetUsers(users []*model.User) error

	// statuses
	// SetStatus stores the status for the given userId.
	SetStatus(userId string, status *model.Status) error

	// posts
	// SetPost stores the given post.
	SetPost(post *model.Post) error
	// DeletePost deletes the specified post.
	DeletePost(postId string) error
	// SetPosts stores the given posts.
	SetPosts(posts []*model.Post) error

	// reactions
	// SetReaction stores the given reaction.
	SetReaction(reaction *model.Reaction) error
	// DeleteReaction deletes the given reaction.
	// It returns whether or not the reaction was deleted.
	DeleteReaction(reaction *model.Reaction) (bool, error)

	// preferences
	// Preferences stores the preferences for the stored user.
	SetPreferences(preferences model.Preferences) error

	// channels
	SetChannel(channel *model.Channel) error
	// SetChannels adds the given channels to the store.
	SetChannels(channels []*model.Channel) error
	// SetCurrentChannel stores the channel the user is currently viewing.
	SetCurrentChannel(channel *model.Channel) error
	// SetChannelView marks the given channel as viewed and updates the store with the
	// current timestamp.
	SetChannelView(channelId string) error
	// SetChannelMembers stores the given channel members in the store.
	SetChannelMembers(channelMembers model.ChannelMembers) error
	// SetChannelMember stores the given channel member.
	SetChannelMember(channelId string, channelMember *model.ChannelMember) error
	// RemoveChannelMember removes the channel member for the specified channel and user.
	RemoveChannelMember(channelId string, userId string) error
	// SetChannelStats stores statistics for the given channelId.
	SetChannelStats(channelId string, stats *model.ChannelStats) error

	// teams
	SetTeam(team *model.Team) error
	// Team returns the team for the given teamId.
	Team(teamId string) (*model.Team, error)
	// SetTeams stores the given teams.
	SetTeams(teams []*model.Team) error
	// SetCurrentTeam sets the currently selected team for the user.
	SetCurrentTeam(team *model.Team) error
	// SetTeamMember stores the given team member.
	SetTeamMember(teamId string, teamMember *model.TeamMember) error
	// RemoveTeamMember removes the team member for the specified team and user..
	RemoveTeamMember(teamId, memberId string) error
	// SetTeamMembers stores the given team members.
	SetTeamMembers(teamId string, teamMember []*model.TeamMember) error

	// roles
	// SetRoles stores the given roles.
	SetRoles(roles []*model.Role) error

	// emoji
	// SetEmojis stores the given emojis.
	SetEmojis(emoji []*model.Emoji) error

	// license
	// SetLicense stores the given license in the store.
	SetLicense(license map[string]string) error

	// profile
	// SetProfileImage sets as stored the profile image for the given user.
	SetProfileImage(userId string) error

	// SetServerVersion sets the server version string.
	SetServerVersion(version string) error

	// Threads
	// SetThreads stores the given posts.
	SetThreads(threads []*model.ThreadResponse) error
	// MarkAllThreadsInTeamAsRead marks all threads in the given team as read
	MarkAllThreadsInTeamAsRead(teamId string) error

	// SidebarCategories
	SetCategories(teamID string, sidebarCategories *model.OrderedSidebarCategories) error
}
