// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package sampleuser

import (
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-server/model"
)

type SampleUser struct {
	id     int
	store  store.MutableUserStore
	client *model.Client4
}

func (u *SampleUser) Id() int {
	return u.id
}

func (u *SampleUser) Store() store.UserStore {
	return u.store
}

func New(store store.MutableUserStore, id int, serverURL string) *SampleUser {
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
		id:     id,
		client: client,
		store:  store,
	}
}

func (u *SampleUser) Connect() error {
	return nil
}

func (u *SampleUser) Disconnect() error {
	return nil
}

func (u *SampleUser) CreatePost(post *model.Post) (string, error) {
	return "", nil
}

func (u *SampleUser) UploadFile(data []byte, channelId, filename string) (*model.FileUploadResponse, error) {
	return nil, nil
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

func (ue *SampleUser) RemoveUserFromChannel(channelId, userId string) (bool, error) {
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

func (u *SampleUser) GetChannelUnread(channelId string) (*model.ChannelUnread, error) {
	return nil, nil
}

func (u *SampleUser) GetChannelMembers(channelId string, page, perPage int) error {
	return nil
}

func (u *SampleUser) GetChannelMember(channelId, userId string) error {
	return nil
}

func (u *SampleUser) GetChannelStats(channelId string) error {
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

	_, resp := u.client.Login(user.Email, user.Password)

	return resp.Error
}

func (u *SampleUser) Logout() (bool, error) {
	user, err := u.store.User()

	if user == nil || err != nil {
		return false, errors.New("user was not initialized")
	}

	ok, resp := u.client.Logout()

	return ok, resp.Error
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

func (u *SampleUser) CreateTeam(team *model.Team) (string, error) {
	return "", nil
}

func (u *SampleUser) AddTeamMember(teamId, userId string) error {
	return nil
}

func (u *SampleUser) GetTeamStats(teamId string) error {
	return nil
}

func (ue *SampleUser) GetTeamsUnread(teamIdToExclude string) ([]*model.TeamUnread, error) {
	return []*model.TeamUnread{}, nil
}