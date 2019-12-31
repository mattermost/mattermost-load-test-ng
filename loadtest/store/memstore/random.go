// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package memstore

import (
	"math/rand"

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
	genericMap := make(map[string]interface{})
	for k, v := range s.teams {
		genericMap[k] = v
	}
	team := selectRandomFromMap(genericMap)
	return *(team.(*model.Team)), nil
}

// RandomUser returns a random user from the set of users.
func (s *MemStore) RandomUser() (model.User, error) {
	genericMap := make(map[string]interface{})
	for k, v := range s.users {
		genericMap[k] = v
	}
	user := selectRandomFromMap(genericMap)
	return *(user.(*model.User)), nil
}

// RandomPost returns a random post.
func (s *MemStore) RandomPost() (model.Post, error) {
	genericMap := make(map[string]interface{})
	for k, v := range s.posts {
		genericMap[k] = v
	}
	post := selectRandomFromMap(genericMap)
	return *(post.(*model.Post)), nil
}

// RandomEmoji returns a random emoji.
func (s *MemStore) RandomEmoji() (model.Emoji, error) {
	return *s.emojis[rand.Intn(len(s.emojis))], nil
}

// RandomChannelMember returns a random channel member for a channel.
func (s *MemStore) RandomChannelMember(channelId string) (model.ChannelMember, error) {
	genericMap := make(map[string]interface{})
	for k, v := range s.channelMembers {
		if k == channelId {
			for k1, v1 := range v {
				genericMap[k1] = v1
			}
			break
		}
	}
	cm := selectRandomFromMap(genericMap)
	return *(cm.(*model.ChannelMember)), nil
}

// RandomTeamMember returns a random team member for a team.
func (s *MemStore) RandomTeamMember(teamId string) (model.TeamMember, error) {
	genericMap := make(map[string]interface{})
	for k, v := range s.teamMembers {
		if k == teamId {
			for k1, v1 := range v {
				genericMap[k1] = v1
			}
			break
		}
	}
	tm := selectRandomFromMap(genericMap)
	return *(tm.(*model.TeamMember)), nil
}

// selectRandomFromMap flattens a map to a slice
// and then returns a random index from the slice.
// We do this, because even though the map iteration order
// is "undefined" in the spec, practically it returns the
// same set of values always.
func selectRandomFromMap(m map[string]interface{}) interface{} {
	items := make([]interface{}, len(m))
	i := 0
	for _, val := range m {
		items[i] = val
		i++
	}
	return items[rand.Intn(len(items))]
}
