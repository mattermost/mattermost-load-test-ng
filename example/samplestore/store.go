// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package samplestore

import (
	"errors"

	"github.com/mattermost/mattermost-server/model"
)

type SampleStore struct {
	user        *model.User
	preferences model.Preferences
	posts       map[string]*model.Post
	channels    map[string]*model.Channel
}

func New() *SampleStore {
	return &SampleStore{
		posts:    map[string]*model.Post{},
		channels: map[string]*model.Channel{},
	}
}

func (s *SampleStore) Id() string {
	if s.user == nil {
		return ""
	}
	return s.user.Id
}

func (s *SampleStore) User() (*model.User, error) {
	return s.user, nil
}

func (s *SampleStore) Preferences() (model.Preferences, error) {
	return s.preferences, nil
}

func (s *SampleStore) SetPreferences(preferences model.Preferences) error {
	s.preferences = preferences
	return nil
}

func (s *SampleStore) Post(postId string) (*model.Post, error) {
	if post, ok := s.posts[postId]; ok {
		return post, nil
	}
	return nil, nil
}

func (s *SampleStore) SetUser(user *model.User) error {
	s.user = user
	return nil
}

func (s *SampleStore) SetPost(post *model.Post) error {
	s.posts[post.Id] = post
	return nil
}

func (s *SampleStore) Channel(channelId string) (*model.Channel, error) {
	if channel, ok := s.channels[channelId]; ok {
		return channel, nil
	}
	return nil, nil
}

func (s *SampleStore) SetChannel(channel *model.Channel) error {
	s.channels[channel.Id] = channel
	return nil
}

func (s *SampleStore) SetChannelMembers(channelId string, channelMembers *model.ChannelMembers) error {
	return errors.New("not implemented")
}

func (s *SampleStore) ChannelMembers(channelId string) (*model.ChannelMembers, error) {
	return nil, errors.New("not implemented")
}

func (s *SampleStore) SetChannelMember(channelId string, channelMember *model.ChannelMember) error {
	return errors.New("not implemented")
}

func (s *SampleStore) ChannelMember(channelId, userId string) (*model.ChannelMember, error) {
	return nil, errors.New("not implemented")
}

func (s *SampleStore) RemoveChannelMember(channelId string, userId string) error {
	return errors.New("not implemented")
}
