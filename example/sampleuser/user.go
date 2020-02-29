// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package sampleuser

import (
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-server/v5/model"
)

type SampleUser struct {
	store  store.MutableUserStore
	client *model.Client4
}

func (u *SampleUser) Store() store.UserStore {
	return u.store
}

func New(store store.MutableUserStore, serverURL string) *SampleUser {
	client := model.NewAPIv4Client(serverURL)
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          1000,
		MaxIdleConnsPerHost:   1000,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	client.HttpClient = &http.Client{Transport: transport}
	return &SampleUser{
		client: client,
		store:  store,
	}
}

func (u *SampleUser) Connect() <-chan error {
	return nil
}

func (u *SampleUser) Disconnect() error {
	return nil
}

func (u *SampleUser) Events() <-chan *model.WebSocketEvent {
	return nil
}

func (u *SampleUser) Cleanup() {
}

func (u *SampleUser) CreatePost(post *model.Post) (string, error) {
	return "", nil
}

func (u *SampleUser) SearchPosts(teamId, terms string, isOrSearch bool) (*model.PostList, error) {
	return nil, nil
}

func (u *SampleUser) GetPostsForChannel(channelId string, page, perPage int) error {
	return nil
}

func (u *SampleUser) GetPostsBefore(channelId, postId string, page, perPage int) error {
	return nil
}

func (u *SampleUser) GetPostsAfter(channelId, postId string, page, perPage int) error {
	return nil
}

func (u *SampleUser) GetPostsSince(channelId string, time int64) error {
	return nil
}

// GetPostsAroundLastUnread returns the list of posts around last unread post by the current user in a channel.
func (u *SampleUser) GetPostsAroundLastUnread(channelId string, limitBefore, limitAfter int) error {
	return nil
}

func (u *SampleUser) UploadFile(data []byte, channelId, filename string) (*model.FileUploadResponse, error) {
	return nil, nil
}

func (u *SampleUser) GetFileInfosForPost(postId string) ([]*model.FileInfo, error) {
	return nil, nil
}

func (u *SampleUser) GetFileThumbnail(fileId string) error {
	return nil
}

func (u *SampleUser) GetFilePreview(fileId string) error {
	return nil
}

func (u *SampleUser) CreateChannel(channel *model.Channel) (string, error) {
	return "", nil
}

func (u *SampleUser) CreateGroupChannel(memberIds []string) (string, error) {
	return "", nil
}

func (u *SampleUser) CreateDirectChannel(otherUserId string) (string, error) {
	return "", nil
}

func (u *SampleUser) RemoveUserFromChannel(channelId, userId string) (bool, error) {
	return true, nil
}

func (u *SampleUser) AddChannelMember(channelId, userId string) error {
	return nil
}

func (u *SampleUser) ViewChannel(view *model.ChannelView) (*model.ChannelViewResponse, error) {
	return nil, nil
}

func (u *SampleUser) GetChannel(channelId string) error {
	return nil
}

func (u *SampleUser) GetChannelsForTeam(teamId string) error {
	return nil
}

func (u *SampleUser) SearchChannels(teamId string, search *model.ChannelSearch) ([]*model.Channel, error) {
	return nil, nil
}

func (u *SampleUser) GetChannelUnread(channelId string) (*model.ChannelUnread, error) {
	return nil, nil
}

func (u *SampleUser) GetChannelMembers(channelId string, page, perPage int) error {
	return nil
}

// GetChannelMembersForUser gets all the channel members for a user on a team.
func (u *SampleUser) GetChannelMembersForUser(userId, teamId string) error {
	return nil
}

func (u *SampleUser) GetChannelMember(channelId, userId string) error {
	return nil
}

func (u *SampleUser) GetChannelStats(channelId string) error {
	return nil
}

func (u *SampleUser) GetChannelsForTeamForUser(teamId, userId string) ([]*model.Channel, error) {
	return nil, nil
}

// AutocompleteChannelsForTeam returns an ordered list of channels for a given name.
func (u *SampleUser) AutocompleteChannelsForTeam(teamId, name string) error {
	return nil
}

func (u *SampleUser) GetUserStatus() error {
	return nil
}

func (u *SampleUser) GetUsersStatusesByIds(userIds []string) error {
	return nil
}

func (u *SampleUser) SignUp(email, username, password string) error {
	user := model.User{
		Email:    email,
		Username: username,
		Password: password,
	}

	newUser, resp := u.client.CreateUser(&user)

	if resp.Error != nil {
		return resp.Error
	}

	newUser.Password = password

	return u.store.SetUser(newUser)
}

func (u *SampleUser) Login() error {
	user, err := u.store.User()

	if user == nil || err != nil {
		return errors.New("user was not initialized")
	}

	if _, resp := u.client.Login(user.Email, user.Password); resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (u *SampleUser) Logout() (bool, error) {
	user, err := u.store.User()

	if user == nil || err != nil {
		return false, errors.New("user was not initialized")
	}

	ok, resp := u.client.Logout()

	if resp.Error != nil {
		return false, resp.Error
	}

	return ok, nil
}

func (u *SampleUser) GetConfig() error {
	return nil
}

func (u *SampleUser) GetMe() (string, error) {
	user, resp := u.client.GetMe("")
	if resp.Error != nil {
		return "", resp.Error
	}

	if err := u.store.SetUser(user); err != nil {
		return "", err
	}

	return user.Id, nil
}

func (u *SampleUser) GetPreferences() error {
	user, err := u.store.User()
	if user == nil || err != nil {
		return errors.New("user was not initialized")
	}

	preferences, resp := u.client.GetPreferences(user.Id)
	if resp.Error != nil {
		return resp.Error
	}

	if err := u.store.SetPreferences(&preferences); err != nil {
		return err
	}
	return nil
}

func (u *SampleUser) CreateUser(user *model.User) (string, error) {
	return "", nil
}

func (u *SampleUser) UpdateUser(user *model.User) error {
	return nil
}

func (u *SampleUser) PatchUser(userId string, patch *model.UserPatch) error {
	return nil
}

func (u *SampleUser) GetTeams() ([]string, error) {
	user, err := u.store.User()
	if user == nil || err != nil {
		return nil, errors.New("user was not initialized")
	}

	return u.GetTeamsForUser(user.Id)
}

// GetAllTeams returns all teams based on permissions.
func (u *SampleUser) GetAllTeams(page, perPage int) ([]string, error) {
	return nil, nil
}

func (u *SampleUser) CreateTeam(team *model.Team) (string, error) {
	return "", nil
}

func (u *SampleUser) GetTeam(teamId string) error {
	return nil
}

func (u *SampleUser) GetTeamsForUser(userId string) ([]string, error) {
	teams, resp := u.client.GetTeamsForUser(userId, "")
	if resp.Error != nil {
		return nil, resp.Error
	}

	if err := u.store.SetTeams(teams); err != nil {
		return nil, err
	}

	teamIds := make([]string, len(teams))
	for i, team := range teams {
		teamIds[i] = team.Id
	}
	return teamIds, nil
}

func (u *SampleUser) AddTeamMember(teamId, userId string) error {
	return nil
}

func (u *SampleUser) RemoveTeamMember(teamId, userId string) error {
	return nil
}

func (u *SampleUser) GetTeamMembers(teamId string, page, perPage int) error {
	return nil
}

func (u *SampleUser) GetTeamMembersForUser(userId string) error {
	return nil
}

func (u *SampleUser) GetUsersByIds(userIds []string) ([]string, error) {
	return nil, nil
}

func (u *SampleUser) GetUsersByUsernames(usernames []string) ([]string, error) {
	return nil, nil
}

func (u *SampleUser) GetTeamStats(teamId string) error {
	return nil
}

func (u *SampleUser) GetTeamsUnread(teamIdToExclude string) ([]*model.TeamUnread, error) {
	return []*model.TeamUnread{}, nil
}

func (u *SampleUser) AddTeamMemberFromInvite(token, inviteId string) error {
	return nil
}

func (u *SampleUser) SetProfileImage(data []byte) error {
	return nil
}

func (u *SampleUser) GetProfileImage() error {
	return nil
}

func (u *SampleUser) GetProfileImageForUser(userId string) error {
	return nil
}

func (u *SampleUser) SearchUsers(search *model.UserSearch) ([]*model.User, error) {
	return nil, nil
}

func (u *SampleUser) GetEmojiList(page, perPage int) error {
	return nil
}

func (u *SampleUser) GetEmojiImage(emojiId string) error {
	return nil
}

func (u *SampleUser) SaveReaction(reaction *model.Reaction) error {
	return nil
}

func (u *SampleUser) DeleteReaction(reaction *model.Reaction) error {
	return nil
}

func (u *SampleUser) GetReactions(postId string) error {
	return nil
}

func (u *SampleUser) GetRolesByNames(roleNames []string) ([]string, error) {
	return nil, nil
}

func (u *SampleUser) GetWebappPlugins() error {
	return nil
}

// GetClientLicense returns the client license in the old format.
func (u *SampleUser) GetClientLicense() error {
	return nil
}

func (u *SampleUser) IsSysAdmin() (bool, error) {
	return false, nil
}
