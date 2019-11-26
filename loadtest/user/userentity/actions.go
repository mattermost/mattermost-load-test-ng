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
	ue.store.SetUser(newUser)

	return nil
}

func (ue *UserEntity) Login() error {
	user := ue.store.User()

	if user == nil {
		return errors.New("user was not initialized")
	}

	_, resp := ue.client.Login(user.Email, user.Password)

	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (ue *UserEntity) Logout() error {
	user := ue.store.User()

	if user == nil {
		return errors.New("user was not initialized")
	}

	_, resp := ue.client.Logout()

	if resp.Error != nil {
		return resp.Error
	}

	return nil
}
