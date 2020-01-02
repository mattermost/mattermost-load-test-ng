// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package memstore

import (
	"math/rand"
	"reflect"

	"github.com/mattermost/mattermost-server/v5/model"
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
	return *channels[rand.Intn(len(channels))], nil
}

// RandomTeam returns a random team for a user.
func (s *MemStore) RandomTeam() (model.Team, error) {
	key := pickRandomKeyFromMap(s.teams).(string)
	return *s.teams[key], nil
}

// RandomUser returns a random user from the set of users.
func (s *MemStore) RandomUser() (model.User, error) {
	key := pickRandomKeyFromMap(s.users).(string)
	return *s.users[key], nil
}

// RandomPost returns a random post.
func (s *MemStore) RandomPost() (model.Post, error) {
	key := pickRandomKeyFromMap(s.posts).(string)
	return *s.posts[key], nil
}

// RandomEmoji returns a random emoji.
func (s *MemStore) RandomEmoji() (model.Emoji, error) {
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
	key := pickRandomKeyFromMap(chanMemberMap).(string)
	return *chanMemberMap[key], nil
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
	key := pickRandomKeyFromMap(teamMemberMap).(string)
	return *teamMemberMap[key], nil
}

func pickRandomKeyFromMap(m interface{}) interface{} {
	keys := reflect.ValueOf(m).MapKeys()
	idx := rand.Intn(len(keys))
	return keys[idx].Interface()
}
