// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package memstore

import (
	"github.com/mattermost/mattermost-server/model"
)

type MemStore struct {
	user           *model.User
	preferences    model.Preferences
	posts          map[string]*model.Post
	teams          map[string]*model.Team
	channels       map[string]*model.Channel
	channelMembers map[string]*model.ChannelMembers
}

func New() *MemStore {
	return &MemStore{
		posts:          map[string]*model.Post{},
		teams:          map[string]*model.Team{},
		channels:       map[string]*model.Channel{},
		channelMembers: map[string]*model.ChannelMembers{},
	}
}

func (s *MemStore) Id() string {
	if s.user == nil {
		return ""
	}
	return s.user.Id
}

func (s *MemStore) User() (*model.User, error) {
	return s.user, nil
}

func (s *MemStore) SetUser(user *model.User) error {
	s.user = user
	return nil
}

func (s *MemStore) Preferences() (model.Preferences, error) {
	return s.preferences, nil
}

func (s *MemStore) SetPreferences(preferences model.Preferences) error {
	s.preferences = preferences
	return nil
}

func (s *MemStore) Post(postId string) (*model.Post, error) {
	if post, ok := s.posts[postId]; ok {
		return post, nil
	}
	return nil, nil
}

func (s *MemStore) SetPost(post *model.Post) error {
	s.posts[post.Id] = post
	return nil
}

func (s *MemStore) Channel(channelId string) (*model.Channel, error) {
	if channel, ok := s.channels[channelId]; ok {
		return channel, nil
	}
	return nil, nil
}

func (s *MemStore) SetChannel(channel *model.Channel) error {
	s.channels[channel.Id] = channel
	return nil
}

func (s *MemStore) SetChannelMembers(channelId string, channelMembers *model.ChannelMembers) error {
	s.channelMembers[channelId] = channelMembers
	return nil
}
