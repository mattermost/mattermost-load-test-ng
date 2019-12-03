// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package userentity

import (
	"errors"

	"github.com/mattermost/mattermost-server/model"
)

func (ue *UserEntity) SignUp(email, username, password string) error {
	user := model.User{
		Email:    email,
		Username: username,
		Password: password,
	}

	newUser, resp := ue.client.CreateUser(&user)

	if resp.Error != nil {
		return resp.Error
	}

	newUser.Password = password
	return ue.store.SetUser(newUser)
}

func (ue *UserEntity) Login() error {
	user, err := ue.store.User()

	if user == nil || err != nil {
		return errors.New("user was not initialized")
	}

	_, resp := ue.client.Login(user.Email, user.Password)

	return resp.Error
}

func (ue *UserEntity) Logout() (bool, error) {
	user, err := ue.store.User()

	if user == nil || err != nil {
		return false, errors.New("user was not initialized")
	}

	ok, resp := ue.client.Logout()

	return ok, resp.Error
}

func (ue *UserEntity) CreatePost(post *model.Post) (string, error) {
	user, err := ue.store.User()
	if user == nil || err != nil {
		return "", errors.New("user was not initialized")
	}

	post.PendingPostId = model.NewId()
	post.UserId = user.Id

	post, resp := ue.client.CreatePost(post)
	if resp.Error != nil {
		return "", resp.Error
	}

	err = ue.store.SetPost(post)

	return post.Id, err
}

func (ue *UserEntity) CreateChannel(channel *model.Channel) (string, error) {
	user, err := ue.store.User()
	if user == nil || err != nil {
		return "", errors.New("user was not initialized")
	}

	channel, resp := ue.client.CreateChannel(channel)
	if resp.Error != nil {
		return "", resp.Error
	}
	err = ue.store.SetChannel(channel)
	return channel.Id, err
}

func (ue *UserEntity) CreateGroupChannel(memberIds []string) (string, error) {
	user, err := ue.store.User()
	if user == nil || err != nil {
		return "", errors.New("user was not initialized")
	}
	channel, resp := ue.client.CreateGroupChannel(memberIds)
	if resp.Error != nil {
		return "", resp.Error
	}
	err = ue.store.SetChannel(channel)
	return channel.Id, err
}

func (ue *UserEntity) CreateDirectChannel(otherUserId string) (string, error) {
	user, err := ue.store.User()
	if user == nil || err != nil {
		return "", errors.New("user was not initialized")
	}
	channel, resp := ue.client.CreateDirectChannel(user.Id, otherUserId)
	if resp.Error != nil {
		return "", resp.Error
	}
	err = ue.store.SetChannel(channel)
	return channel.Id, err
}

func (ue *UserEntity) ViewChannel(view *model.ChannelView) (*model.ChannelViewResponse, error) {
	user, err := ue.store.User()
	if user == nil || err != nil {
		return nil, errors.New("user was not initialized")
	}
	channelViewResponse, resp := ue.client.ViewChannel(user.Id, view)
	return channelViewResponse, resp.Error
}

func (ue *UserEntity) GetChannelUnread(channelId string) (*model.ChannelUnread, error) {
	user, err := ue.store.User()
	if user == nil || err != nil {
		return nil, errors.New("user was not initialized")
	}
	channelUnreadResponse, resp := ue.client.GetChannelUnread(channelId, user.Id)
	return channelUnreadResponse, resp.Error

}
