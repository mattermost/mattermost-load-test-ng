// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package user

import (
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-server/v5/model"
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
	UpdateUser(user *model.User) error
	PatchUser(userId string, patch *model.UserPatch) error
	GetUsersStatusesByIds(userIds []string) error
	SetProfileImage(data []byte) error

	// posts
	CreatePost(post *model.Post) (string, error)

	// files
	UploadFile(data []byte, channelId, filename string) (*model.FileUploadResponse, error)
	GetFileInfosForPost(postId string) ([]*model.FileInfo, error)
	GetFileThumbnail(fileId string) ([]byte, error)

	// channels
	CreateChannel(channel *model.Channel) (string, error)
	CreateGroupChannel(memberIds []string) (string, error)
	CreateDirectChannel(otherUserId string) (string, error)
	GetChannel(channelId string) error
	SearchChannels(teamId string, search *model.ChannelSearch) ([]*model.Channel, error)
	RemoveUserFromChannel(channelId, userId string) (bool, error)
	ViewChannel(view *model.ChannelView) (*model.ChannelViewResponse, error)
	GetChannelUnread(channelId string) (*model.ChannelUnread, error)
	GetChannelMembers(channelId string, page, perPage int) error
	GetChannelMember(channelId string, userId string) error
	GetChannelStats(channelId string) error
	AddChannelMember(channelId, userId string) error

	// teams
	GetTeams() ([]string, error)
	CreateTeam(team *model.Team) (string, error)
	AddTeamMember(teamId, userId string) error
	GetTeamMembers(teamId string, page, perPage int) error
	GetTeamStats(teamId string) error
	GetTeamsUnread(teamIdToExclude string) ([]*model.TeamUnread, error)
	AddTeamMemberFromInvite(token, inviteId string) error

	// emoji
	GetEmojiList(page, perPage int) error
}
