// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package samplestore

import (
	"github.com/mattermost/mattermost-server/model"
)

type SampleStore struct {
	user *model.User
}

func New() *SampleStore {
	return &SampleStore{}
}

func (s *SampleStore) Id() string {
	if s.user == nil {
		return ""
	}
	return s.user.Id
}

func (s *SampleStore) User() *model.User {
	return s.user
}

func (s *SampleStore) SetUser(user *model.User) error {
	s.user = user
	return nil
}
