// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package memstore

import (
	"errors"

	"github.com/mattermost/mattermost-server/model"
)

type MemStore struct {
	user           *model.User
	preferences    model.Preferences
	posts          map[string]*model.Post
	teams          map[string]*model.Team
	channels       map[string]*model.Channel
	channelMembers map[string]map[string]*model.ChannelMember
}

func New() *MemStore {
	return &MemStore{
		posts:          map[string]*model.Post{},
		teams:          map[string]*model.Team{},
		channels:       map[string]*model.Channel{},
		channelMembers: map[string]map[string]*model.ChannelMember{},
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
	membersMap := make(map[string]*model.ChannelMember)
	members := *channelMembers
	for _, m := range members {
		membersMap[m.UserId] = &m
	}
	s.channelMembers[channelId] = membersMap
	return nil
}

func (s *MemStore) ChannelMembers(channelId string) (*model.ChannelMembers, error) {
	channelMembers := model.ChannelMembers{}
	for key := range s.channelMembers[channelId] {
		channelMembers = append(channelMembers, *s.channelMembers[channelId][key])
	}
	return &channelMembers, nil
}

func (s *MemStore) SetChannelMember(channelId string, channelMember *model.ChannelMember) error {
	if s.channelMembers[channelId] == nil {
		s.channelMembers[channelId] = map[string]*model.ChannelMember{}
	}
	s.channelMembers[channelId][channelMember.UserId] = channelMember
	return nil
}

func (s *MemStore) ChannelMember(channelId, userId string) (*model.ChannelMember, error) {
	return s.channelMembers[channelId][userId], nil
}

func (s *MemStore) RemoveChannelMember(channelId string, channelMember *model.ChannelMember) error {
	members := *s.channelMembers[channelId]
	for index, member := range *s.channelMembers[channelId] {
		if member.UserId == channelMember.UserId {
			copy(members[index:], members[index+1:])
			members := members[:len(members)-1]
			s.channelMembers[channelId] = &members
			return nil
		}
	}

	return errors.New("User is not a channel member for the passed channel")
}
