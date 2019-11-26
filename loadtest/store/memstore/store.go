// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package memstore

import (
	"github.com/mattermost/mattermost-server/model"
)

type MemStore struct {
	user     *model.User
	teams    map[string]*model.Team
	channels map[string]*model.Channel
}

func New() *MemStore {
	return &MemStore{}
}

func (s *MemStore) User() *model.User {
	return s.user
}

func (s *MemStore) SetUser(user *model.User) error {
	s.user = user
	return nil
}
