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
	user := ue.store.User()

	if user == nil {
		return errors.New("user was not initialized")
	}

	_, resp := ue.client.Login(user.Email, user.Password)

	return resp.Error
}

func (ue *UserEntity) Logout() (bool, error) {
	user := ue.store.User()

	if user == nil {
		return false, errors.New("user was not initialized")
	}

	ok, resp := ue.client.Logout()

	return ok, resp.Error
}


func (ue *UserEntity) CreatePost(post *model.Post) error {
	user := ue.store.User()
	if user == nil {
		return errors.New("user was not initialized")
	}
	
	post.PendingPostId = model.NewId()
	post.UserId = user.Id

	_, resp := ue.client.CreatePost(post)

	return resp.Error
}