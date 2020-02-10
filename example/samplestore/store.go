// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package samplestore

import (
	"errors"

	"github.com/mattermost/mattermost-server/v5/model"
)

type SampleStore struct {
	user           *model.User
	config         *model.Config
	posts          map[string]*model.Post
	preferences    *model.Preferences
	teams          map[string]*model.Team
	channels       map[string]*model.Channel
	channelMembers map[string]*model.ChannelMembers
	roles          map[string]*model.Role
}

func New() *SampleStore {
	return &SampleStore{
		posts:          map[string]*model.Post{},
		teams:          map[string]*model.Team{},
		channels:       map[string]*model.Channel{},
		channelMembers: map[string]*model.ChannelMembers{},
		roles:          map[string]*model.Role{},
	}
}

func (s *SampleStore) Id() string {
	if s.user == nil {
		return ""
	}
	return s.user.Id
}

func (s *SampleStore) Username() string {
	if s.user == nil {
		return ""
	}
	return s.user.Username
}

func (s *SampleStore) Email() string {
	if s.user == nil {
		return ""
	}
	return s.user.Email
}

func (s *SampleStore) Password() string {
	if s.user == nil {
		return ""
	}
	return s.user.Password
}

func (s *SampleStore) Config() model.Config {
	return *s.config
}

func (s *SampleStore) SetConfig(config *model.Config) {
	s.config = config
}

func (s *SampleStore) User() (*model.User, error) {
	return s.user, nil
}

func (s *SampleStore) Preferences() (model.Preferences, error) {
	newPref := make(model.Preferences, len(*s.preferences))
	copy(newPref, *s.preferences)
	return newPref, nil
}

func (s *SampleStore) SetPreferences(preferences *model.Preferences) error {
	s.preferences = preferences
	return nil
}

func (s *SampleStore) Post(postId string) (*model.Post, error) {
	if post, ok := s.posts[postId]; ok {
		return post, nil
	}
	return nil, nil
}

func (s *SampleStore) ChannelPosts(channelId string) ([]*model.Post, error) {
	return nil, nil
}

func (s *SampleStore) ChannelPostsSorted(channelId string, asc bool) ([]*model.Post, error) {
	return nil, nil
}

func (s *SampleStore) PostsSince(ts int64) ([]model.Post, error) {
	return []model.Post{}, nil
}

func (s *SampleStore) SetUser(user *model.User) error {
	s.user = user
	return nil
}

func (s *SampleStore) SetPost(post *model.Post) error {
	s.posts[post.Id] = post
	return nil
}

func (s *SampleStore) SetPosts(posts []*model.Post) error {
	for _, post := range posts {
		s.posts[post.Id] = post
	}
	return nil
}

func (s *SampleStore) Channel(channelId string) (*model.Channel, error) {
	if channel, ok := s.channels[channelId]; ok {
		return channel, nil
	}
	return nil, nil
}

// Channels return all the channels for a team.
func (s *SampleStore) Channels(teamId string) ([]model.Channel, error) {
	var channels []model.Channel
	for _, channel := range s.channels {
		if channel.TeamId == teamId {
			channels = append(channels, *channel)
		}
	}
	return channels, nil
}

func (s *SampleStore) SetChannel(channel *model.Channel) error {
	s.channels[channel.Id] = channel
	return nil
}

func (s *SampleStore) SetChannels(channels []*model.Channel) error {
	for _, channel := range channels {
		if err := s.SetChannel(channel); err != nil {
			return err
		}
	}
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

func (s *SampleStore) Teams() ([]model.Team, error) {
	teams := make([]model.Team, len(s.teams))
	for _, team := range s.teams {
		teams = append(teams, *team)
	}
	return teams, nil
}

func (s *SampleStore) SetTeams(teams []*model.Team) error {
	for _, team := range teams {
		s.teams[team.Id] = team
	}
	return nil
}

// SetRoles stores the given roles.
func (s *SampleStore) SetRoles(roles []*model.Role) error {
	s.roles = make(map[string]*model.Role)
	for _, role := range roles {
		s.roles[role.Id] = role
	}
	return nil
}

// Roles return the roles of the user.
func (s *SampleStore) Roles() ([]model.Role, error) {
	roles := make([]model.Role, len(s.roles))
	i := 0
	for _, role := range s.roles {
		roles[i] = *role
		i++
	}
	return roles, nil
}

// RandomChannel returns a random channel for a user.
func (s *SampleStore) RandomChannel(teamId string) (model.Channel, error) {
	return model.Channel{}, nil
}

// RandomTeam returns a random team for a user.
func (s *SampleStore) RandomTeam() (model.Team, error) {
	return model.Team{}, nil
}

// RandomUser returns a random user from the set of users.
func (s *SampleStore) RandomUser() (model.User, error) {
	return model.User{}, nil
}

// RandomUser returns a random user from the set of users.
func (s *SampleStore) RandomUsers(n int) ([]model.User, error) {
	return []model.User{}, nil
}

// RandomPost returns a random post.
func (s *SampleStore) RandomPost() (model.Post, error) {
	return model.Post{}, nil
}

// RandomEmoji returns a random emoji.
func (s *SampleStore) RandomEmoji() (model.Emoji, error) {
	return model.Emoji{}, nil
}

// RandomChannelMember returns a random channel member for a channel.
func (s *SampleStore) RandomChannelMember(channelId string) (model.ChannelMember, error) {
	return model.ChannelMember{}, nil
}

// RandomTeamMember returns a random team member for a team.
func (s *SampleStore) RandomTeamMember(teamId string) (model.TeamMember, error) {
	return model.TeamMember{}, nil
}

// SetLicense stores the given license in the store.
func (s *SampleStore) SetLicense(license map[string]string) error {
	return nil
}

// SetChannelMembers stores the given channel members in the store.
func (s *SampleStore) SetChannelMembers(channelMembers *model.ChannelMembers) error {
	return errors.New("not implemented")
}

func (s *SampleStore) ChannelMembers(channelId string) (*model.ChannelMembers, error) {
	return nil, errors.New("not implemented")
}

func (s *SampleStore) SetChannelMember(channelId string, channelMember *model.ChannelMember) error {
	return errors.New("not implemented")
}

func (s *SampleStore) ChannelMember(channelId, userId string) (model.ChannelMember, error) {
	return model.ChannelMember{}, errors.New("not implemented")
}

func (s *SampleStore) RemoveChannelMember(channelId string, userId string) error {
	return errors.New("not implemented")
}

func (s *SampleStore) SetTeamMember(teamId string, teamMember *model.TeamMember) error {
	return errors.New("not implemented")
}

func (s *SampleStore) RemoveTeamMember(teamId, userId string) error {
	return errors.New("not implemented")
}

func (s *SampleStore) TeamMember(teamId, userId string) (model.TeamMember, error) {
	return model.TeamMember{}, errors.New("not implemented")
}

func (s *SampleStore) SetTeamMembers(teamId string, teamMembers []*model.TeamMember) error {
	return errors.New("not implemented")
}

func (s *SampleStore) SetEmojis(emoji []*model.Emoji) error {
	return errors.New("not implemented")
}

func (s *SampleStore) SetReactions(postId string, reactions []*model.Reaction) error {
	return errors.New("not implemented")
}

func (s *SampleStore) DeleteReaction(reaction *model.Reaction) (bool, error) {
	return false, errors.New("not implemented")
}

func (s *SampleStore) Reactions(postId string) ([]model.Reaction, error) {
	return nil, errors.New("not implemented")
}

func (s *SampleStore) Users() ([]*model.User, error) {
	return nil, errors.New("not implemented")
}

func (s *SampleStore) SetUsers(users []*model.User) error {
	return errors.New("not implemented")
}

func (s *SampleStore) Clear() {
}
