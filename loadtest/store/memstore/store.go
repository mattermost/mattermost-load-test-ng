// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package memstore

import (
	"fmt"

	"github.com/mattermost/mattermost-server/model"
)

type MemStore struct {
	user     *model.User
	posts    map[string]*model.Post
	teams    map[string]*model.Team
	channels map[string]*model.Channel
}

func New() *MemStore {
	return &MemStore{		
		posts: map[string]*model.Post{},
		teams: map[string]*model.Team{},
		channels: map[string]*model.Channel{},
	}
}

func (s *MemStore) Id() string {
	if s.user == nil {
		return ""
	}
	return s.user.Id
}

func (s *MemStore) User() *model.User {
	return s.user
}

func (s *MemStore) SetUser(user *model.User) error {
	s.user = user
	return nil
}

func (s *MemStore) Post(postId string) (*model.Post, error) {
	if post, ok := s.posts[postId]; ok {
		return post, nil
	}
	return nil, fmt.Errorf("post with id %v not found", postId)
}

func (s *MemStore) SetPost(post *model.Post) error {
	s.posts[post.Id] = post
	return nil
}
