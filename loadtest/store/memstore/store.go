// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package memstore

import (
	"errors"

	"github.com/mattermost/mattermost-server/v5/model"
)

type MemStore struct {
	user           *model.User
	preferences    *model.Preferences
	emojis         []*model.Emoji
	posts          map[string]*model.Post
	teams          map[string]*model.Team
	channels       map[string]*model.Channel
	channelMembers map[string]map[string]*model.ChannelMember
	teamMembers    map[string]map[string]*model.TeamMember
	users          map[string]*model.User
	reactions      map[string][]*model.Reaction
}

func New() *MemStore {
	return &MemStore{
		posts:          map[string]*model.Post{},
		teams:          map[string]*model.Team{},
		channels:       map[string]*model.Channel{},
		channelMembers: map[string]map[string]*model.ChannelMember{},
		teamMembers:    map[string]map[string]*model.TeamMember{},
		users:          map[string]*model.User{},
		reactions:      map[string][]*model.Reaction{},
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
	if user == nil {
		return errors.New("user should not be nil")
	}
	s.user = user
	return nil
}

func (s *MemStore) Preferences() (*model.Preferences, error) {
	return s.preferences, nil
}

func (s *MemStore) SetPreferences(preferences *model.Preferences) error {
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
	if post == nil {
		return errors.New("post should not be nil")
	}
	s.posts[post.Id] = post
	return nil
}

func (s *MemStore) SetPosts(posts []*model.Post) error {
	if posts == nil || len(posts) == 0 {
		return errors.New("posts should not be nil or empty")
	}

	for _, post := range posts {
		s.SetPost(post)
	}
	return nil
}

func (s *MemStore) Channel(channelId string) (*model.Channel, error) {
	if channel, ok := s.channels[channelId]; ok {
		return channel, nil
	}
	return nil, nil
}

func (s *MemStore) SetChannel(channel *model.Channel) error {
	if channel == nil {
		return errors.New("channel should not be nil")
	}
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
	teams := make([]*model.Team, len(s.teams))
	i := 0
	for _, team := range s.teams {
		teams[i] = team
		i++
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
	if channelMembers == nil {
		return errors.New("channelMembers should not be nil")
	}
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
	if channelMember == nil {
		return errors.New("channelMember should not be nil")
	}
	if s.channelMembers[channelId] == nil {
		s.channelMembers[channelId] = map[string]*model.ChannelMember{}
	}
	s.channelMembers[channelId][channelMember.UserId] = channelMember
	return nil
}

func (s *MemStore) ChannelMember(channelId, userId string) (*model.ChannelMember, error) {
	return s.channelMembers[channelId][userId], nil
}

func (s *MemStore) RemoveChannelMember(channelId string, userId string) error {
	if s.channelMembers[channelId] == nil {
		return nil
	}
	delete(s.channelMembers[channelId], userId)
	return nil
}

func (s *MemStore) SetTeamMember(teamId string, teamMember *model.TeamMember) error {
	if teamMember == nil {
		return errors.New("teamMember should not be nil")
	}
	if s.teamMembers[teamId] == nil {
		s.teamMembers[teamId] = map[string]*model.TeamMember{}
	}
	s.teamMembers[teamId][teamMember.UserId] = teamMember
	return nil
}

func (s *MemStore) SetTeamMembers(teamId string, teamMembers []*model.TeamMember) error {
	s.teamMembers[teamId] = map[string]*model.TeamMember{}
	for _, m := range teamMembers {
		s.teamMembers[teamId][m.UserId] = m
	}

	return nil
}

func (s *MemStore) TeamMember(teamId, userId string) (*model.TeamMember, error) {
	return s.teamMembers[teamId][userId], nil
}

func (s *MemStore) SetEmojis(emoji []*model.Emoji) error {
	s.emojis = emoji
	return nil
}

func (s *MemStore) SetReactions(postId string, reactions []*model.Reaction) error {
	s.reactions[postId] = reactions
	return nil
}

func (s *MemStore) Reactions(postId string) ([]*model.Reaction, error) {
	return s.reactions[postId], nil
}

func (s *MemStore) Users() ([]*model.User, error) {
	users := make([]*model.User, len(s.users))
	i := 0
	for _, user := range s.users {
		users[i] = user
		i++
	}
	return users, nil
}

func (s *MemStore) SetUsers(users []*model.User) error {
	s.users = make(map[string]*model.User)
	for _, user := range users {
		s.users[user.Id] = user
	}
	return nil
}
