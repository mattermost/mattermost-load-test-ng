// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"errors"
	"math/rand"
	"reflect"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-server/v5/model"
)

var (
	ErrEmptyMap          = errors.New("memstore: cannot select from an empty map")
	ErrEmptySlice        = errors.New("memstore: cannot select from an empty slice")
	ErrLenMismatch       = errors.New("memstore: cannot select from a map, not enough elements")
	ErrTeamNotFound      = errors.New("memstore: team not found")
	ErrUserNotSet        = errors.New("memstore: user is not set")
	ErrTeamStoreEmpty    = errors.New("memstore: team store is empty")
	ErrChannelStoreEmpty = errors.New("memstore: channel store is empty")
	ErrPostNotFound      = errors.New("memstore: post not found")
)

// RandomTeam returns a random team for the current user.
func (s *MemStore) RandomTeam(st store.SelectionType) (model.Team, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.user == nil {
		return model.Team{}, ErrUserNotSet
	}

	userId := s.user.Id

	var teams []*model.Team
	for teamId, team := range s.teams {
		_, isMember := s.teamMembers[teamId][userId]
		if isMember && ((st & store.SelectMemberOf) == store.SelectMemberOf) {
			teams = append(teams, team)
		}
		if !isMember && ((st & store.SelectNotMemberOf) == store.SelectNotMemberOf) {
			teams = append(teams, team)
		}
	}

	if len(teams) == 0 {
		return model.Team{}, ErrTeamStoreEmpty
	}

	idx := rand.Intn(len(teams))

	return *teams[idx], nil
}

// RandomChannel returns a random channel for the given teamId for the current
// user.
func (s *MemStore) RandomChannel(teamId string, st store.SelectionType) (model.Channel, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.user == nil {
		return model.Channel{}, ErrUserNotSet
	}

	if s.teams[teamId] == nil {
		return model.Channel{}, ErrTeamNotFound
	}

	userId := s.user.Id

	var channels []*model.Channel
	for channelId, channel := range s.channels {
		_, isMember := s.channelMembers[channelId][userId]
		if channel.TeamId != teamId {
			continue
		}
		if isMember && ((st & store.SelectMemberOf) == store.SelectMemberOf) {
			channels = append(channels, channel)
		}
		if !isMember && ((st & store.SelectNotMemberOf) == store.SelectNotMemberOf) {
			channels = append(channels, channel)
		}
	}

	if len(channels) == 0 {
		return model.Channel{}, ErrChannelStoreEmpty
	}

	idx := rand.Intn(len(channels))

	return *channels[idx], nil
}

// RandomUser returns a random user from the set of users.
func (s *MemStore) RandomUser() (model.User, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	key, err := pickRandomKeyFromMap(s.users)
	if err != nil {
		return model.User{}, err
	}
	return *s.users[key.(string)], nil
}

// RandomUsers returns N random users from the set of users.
func (s *MemStore) RandomUsers(n int) ([]model.User, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if n > len(s.users) {
		return nil, ErrLenMismatch
	}
	var users []model.User
	for len(users) < n {
		u, err := s.RandomUser()
		if err != nil {
			return nil, err
		}
		found := false
		for _, ou := range users {
			if ou.Id == u.Id {
				found = true
				break
			}
		}
		if found {
			continue
		}
		users = append(users, u)
	}
	return users, nil
}

// RandomPost returns a random post.
func (s *MemStore) RandomPost() (model.Post, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	key, err := pickRandomKeyFromMap(s.posts)
	if err != nil {
		return model.Post{}, err
	}
	return *s.posts[key.(string)], nil
}

// RandomPostForChannel returns a random post for the given channel.
func (s *MemStore) RandomPostForChannel(channelId string) (model.Post, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var postIds []string
	for _, p := range s.posts {
		if p.ChannelId == channelId {
			postIds = append(postIds, p.Id)
		}
	}

	if len(postIds) == 0 {
		return model.Post{}, ErrPostNotFound
	}

	return *s.posts[postIds[rand.Intn(len(postIds))]], nil
}

// RandomEmoji returns a random emoji.
func (s *MemStore) RandomEmoji() (model.Emoji, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if len(s.emojis) == 0 {
		return model.Emoji{}, ErrEmptySlice
	}
	return *s.emojis[rand.Intn(len(s.emojis))], nil
}

// RandomChannelMember returns a random channel member for a channel.
func (s *MemStore) RandomChannelMember(channelId string) (model.ChannelMember, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var chanMemberMap map[string]*model.ChannelMember
	for k, v := range s.channelMembers {
		if k == channelId {
			chanMemberMap = v
			break
		}
	}
	key, err := pickRandomKeyFromMap(chanMemberMap)
	if err != nil {
		return model.ChannelMember{}, err
	}
	return *chanMemberMap[key.(string)], nil
}

// RandomTeamMember returns a random team member for a team.
func (s *MemStore) RandomTeamMember(teamId string) (model.TeamMember, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var teamMemberMap map[string]*model.TeamMember
	for k, v := range s.teamMembers {
		if k == teamId {
			teamMemberMap = v
			break
		}
	}
	key, err := pickRandomKeyFromMap(teamMemberMap)
	if err != nil {
		return model.TeamMember{}, err
	}
	return *teamMemberMap[key.(string)], nil
}

func pickRandomKeyFromMap(m interface{}) (interface{}, error) {
	val := reflect.ValueOf(m)
	if val.Kind() != reflect.Map {
		return nil, errors.New("memstore: not a map")
	}
	keys := val.MapKeys()
	if len(keys) == 0 {
		return nil, ErrEmptyMap
	}
	idx := rand.Intn(len(keys))
	return keys[idx].Interface(), nil
}
