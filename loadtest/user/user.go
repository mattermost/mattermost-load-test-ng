// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package user

import (
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-server/v5/model"
)

// User provides a wrapper interface to interact with the Mattermost server
// through its client APIs. It persists the data to its UserStore for later use.
type User interface {
	Store() store.UserStore

	// Cleanup is a one time method used to close any open resources
	// that the user might have kept open throughout its lifetime.
	Cleanup()

	// connection
	Connect() <-chan error
	Disconnect() error
	// Events returns the WebSocket event chan for the controller
	// to listen and react to events.
	Events() <-chan *model.WebSocketEvent
	SignUp(email, username, password string) error
	Login() error
	Logout() (bool, error)

	//server
	GetConfig() error

	// user
	GetMe() (string, error)
	GetPreferences() error
	CreateUser(user *model.User) (string, error)
	UpdateUser(user *model.User) error
	PatchUser(userId string, patch *model.UserPatch) error
	GetUsersByIds(userIds []string) ([]string, error)
	GetUsersByUsernames(usernames []string) ([]string, error)
	GetUserStatus() error
	GetUsersStatusesByIds(userIds []string) error
	SetProfileImage(data []byte) error
	GetProfileImage() error
	GetProfileImageForUser(userId string) error
	SearchUsers(search *model.UserSearch) ([]*model.User, error)

	// posts
	CreatePost(post *model.Post) (string, error)
	SearchPosts(teamId, terms string, isOrSearch bool) (*model.PostList, error)
	GetPostsForChannel(channelId string, page, perPage int) error
	GetPostsBefore(channelId, postId string, page, perPage int) error
	GetPostsAfter(channelId, postId string, page, perPage int) error
	GetPostsSince(channelId string, time int64) error
	// GetPostsAroundLastUnread returns the list of posts around last unread post by the current user in a channel.
	GetPostsAroundLastUnread(channelId string, limitBefore, limitAfter int) error
	SaveReaction(reaction *model.Reaction) error
	DeleteReaction(reaction *model.Reaction) error
	GetReactions(postId string) error

	// files
	UploadFile(data []byte, channelId, filename string) (*model.FileUploadResponse, error)
	GetFileInfosForPost(postId string) ([]*model.FileInfo, error)
	GetFileThumbnail(fileId string) error
	GetFilePreview(fileId string) error

	// channels
	CreateChannel(channel *model.Channel) (string, error)
	CreateGroupChannel(memberIds []string) (string, error)
	CreateDirectChannel(otherUserId string) (string, error)
	GetChannel(channelId string) error
	GetChannelsForTeam(teamId string) error
	SearchChannels(teamId string, search *model.ChannelSearch) ([]*model.Channel, error)
	RemoveUserFromChannel(channelId, userId string) (bool, error)
	ViewChannel(view *model.ChannelView) (*model.ChannelViewResponse, error)
	GetChannelUnread(channelId string) (*model.ChannelUnread, error)
	GetChannelMembers(channelId string, page, perPage int) error
	// GetChannelMembersForUser gets all the channel members for a user on a team.
	GetChannelMembersForUser(userId, teamId string) error
	GetChannelMember(channelId string, userId string) error
	GetChannelStats(channelId string) error
	AddChannelMember(channelId, userId string) error
	GetChannelsForTeamForUser(teamId, userId string) ([]*model.Channel, error)
	// AutocompleteChannelsForTeam returns an ordered list of channels for a given name.
	AutocompleteChannelsForTeam(teamId, name string) error

	// teams
	GetTeams() ([]string, error)
	// GetAllTeams returns all teams based on permissions.
	GetAllTeams(page, perPage int) ([]string, error)
	CreateTeam(team *model.Team) (string, error)
	GetTeam(teamId string) error
	GetTeamsForUser(userId string) ([]string, error)
	AddTeamMember(teamId, userId string) error
	RemoveTeamMember(teamId, userId string) error
	GetTeamMembers(teamId string, page, perPage int) error
	GetTeamMembersForUser(userId string) error
	GetTeamStats(teamId string) error
	GetTeamsUnread(teamIdToExclude string) ([]*model.TeamUnread, error)
	AddTeamMemberFromInvite(token, inviteId string) error

	// roles
	// GetRolesByNames returns a list of role ids based on the provided role names.
	GetRolesByNames(roleNames []string) ([]string, error)

	// emoji
	GetEmojiList(page, perPage int) error
	GetEmojiImage(emojiId string) error

	// plugins
	GetWebappPlugins() error

	// license
	// GetClientLicense returns the client license in the old format.
	GetClientLicense() error

	// utils
	// IsSysAdmin will return true if the user is a SystemAdmin, false otherwise.
	IsSysAdmin() (bool, error)
}
