// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package memstore

import (
	"github.com/mattermost/mattermost-server/model"
)

type MemStore struct {
	user           *model.User
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

func (s *MemStore) Team(teamId string) (*model.Team, error) {
	if team, ok := s.teams[teamId]; ok {
		return team, nil
	}
	return nil, nil
}

func (s *MemStore) SetTeam(team *model.Team) error {
	s.teams[team.Id] = team
	return nil
}

func (s *MemStore) Teams() ([]*model.Team, error) {
	teams := []*model.Team{}
	for _, team := range s.teams {
		teams = append(teams, team)
	}
	return teams, nil
}

func (s *MemStore) SetTeams(teams []*model.Team) error {
	s.teams = make(map[string]*model.Team)
	for _, team := range teams {
		s.teams[team.Id] = team
	}
	return nil
}

func (s *MemStore) SetChannelMembers(channelId string, channelMembers *model.ChannelMembers) error {
	s.channelMembers[channelId] = channelMembers
	return nil
}
