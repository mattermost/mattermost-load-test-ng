// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package memstore

import (
	"errors"
	"math/rand"
	"reflect"

	"github.com/mattermost/mattermost-server/v5/model"
)

var (
	ErrEmptyMap   = errors.New("memstore: cannot select from an empty map")
	ErrEmptySlice = errors.New("memstore: cannot select from an empty slice")
)

// RandomChannel returns a random channel for a user.
func (s *MemStore) RandomChannel(teamId string) (model.Channel, error) {
	var channels []*model.Channel
	i := 0
	for _, channel := range s.channels {
		if channel.TeamId == teamId {
			channels = append(channels, channel)
		}
		i++
	}
	if len(channels) == 0 {
		return model.Channel{}, ErrEmptySlice
	}
	return *channels[rand.Intn(len(channels))], nil
}

// RandomTeam returns a random team for a user.
func (s *MemStore) RandomTeam() (model.Team, error) {
	key, err := pickRandomKeyFromMap(s.teams)
	if err != nil {
		return model.Team{}, err
	}
	return *s.teams[key.(string)], nil
}

// RandomUser returns a random user from the set of users.
func (s *MemStore) RandomUser() (model.User, error) {
	key, err := pickRandomKeyFromMap(s.users)
	if err != nil {
		return model.User{}, err
	}
	return *s.users[key.(string)], nil
}

// RandomUsers returns N random users from the set of users.
func (s *MemStore) RandomUsers(n int) ([]model.User, error) {
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
	key, err := pickRandomKeyFromMap(s.posts)
	if err != nil {
		return model.Post{}, err
	}
	return *s.posts[key.(string)], nil
}

// RandomEmoji returns a random emoji.
func (s *MemStore) RandomEmoji() (model.Emoji, error) {
	if len(s.emojis) == 0 {
		return model.Emoji{}, ErrEmptySlice
	}
	return *s.emojis[rand.Intn(len(s.emojis))], nil
}

// RandomChannelMember returns a random channel member for a channel.
func (s *MemStore) RandomChannelMember(channelId string) (model.ChannelMember, error) {
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
