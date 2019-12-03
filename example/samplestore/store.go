// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package samplestore

import (
	"github.com/mattermost/mattermost-server/model"
)

type SampleStore struct {
	user           *model.User
	posts          map[string]*model.Post
	teams          map[string]*model.Team
	channels       map[string]*model.Channel
	channelMembers map[string]*model.ChannelMembers
}

func New() *SampleStore {
	return &SampleStore{
		posts:          map[string]*model.Post{},
		teams:          map[string]*model.Team{},
		channels:       map[string]*model.Channel{},
		channelMembers: map[string]*model.ChannelMembers{},
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

func (s *SampleStore) Team(teamId string) (*model.Team, error) {
	if team, ok := s.teams[teamId]; ok {
		return team, nil
	}
	return nil, nil
}

func (s *SampleStore) SetTeam(team *model.Team) error {
	s.teams[team.Id] = team
	return nil
}

func (s *SampleStore) Teams() ([]*model.Team, error) {
	teams := make([]*model.Team, len(s.teams))
	for _, team := range s.teams {
		teams = append(teams, team)
	}
	return teams, nil
}

func (s *SampleStore) SetTeams(teams []*model.Team) error {
	for _, team := range teams {
		s.teams[team.Id] = team
	}
	return nil
}

func (s *SampleStore) SetChannelMembers(channelId string, channelMembers *model.ChannelMembers) error {
	s.channelMembers[channelId] = channelMembers
	return nil
}
