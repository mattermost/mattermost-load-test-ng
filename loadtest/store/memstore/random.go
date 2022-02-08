// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"errors"
	"math/rand"
	"reflect"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-server/v6/model"
)

var (
	ErrEmptyMap          = errors.New("memstore: cannot select from an empty map")
	ErrEmptySlice        = errors.New("memstore: cannot select from an empty slice")
	ErrLenMismatch       = errors.New("memstore: cannot select from a map, not enough elements")
	ErrTeamNotFound      = errors.New("memstore: team not found")
	ErrUserNotSet        = errors.New("memstore: user is not set")
	ErrTeamStoreEmpty    = errors.New("memstore: team store is empty")
	ErrChannelStoreEmpty = errors.New("memstore: channel store is empty")
	ErrChannelNotFound   = errors.New("memstore: channel not found")
	ErrPostNotFound      = errors.New("memstore: post not found")
	ErrInvalidData       = errors.New("memstore: invalid data found")
	ErrThreadNotFound    = errors.New("memstore: thread not found")
)

func isSelectionType(st, t store.SelectionType) bool {
	return (st & t) == t
}

// RandomTeam returns a random team for the current user.
func (s *MemStore) RandomTeam(st store.SelectionType) (model.Team, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.user == nil {
		return model.Team{}, ErrUserNotSet
	}

	userId := s.user.Id

	var currTeamId string
	if s.currentTeam != nil {
		currTeamId = s.currentTeam.Id
	}

	var teams []*model.Team
	for teamId, team := range s.teams {
		if (currTeamId == teamId) && isSelectionType(st, store.SelectNotCurrent) {
			continue
		}
		_, isMember := s.teamMembers[teamId][userId]
		if isMember && isSelectionType(st, store.SelectMemberOf) {
			teams = append(teams, team)
		}
		if !isMember && isSelectionType(st, store.SelectNotMemberOf) {
			teams = append(teams, team)
		}
	}

	if len(teams) == 0 {
		return model.Team{}, ErrTeamStoreEmpty
	}

	idx := rand.Intn(len(teams))

	return *teams[idx], nil
}

func excludeChannelType(st store.SelectionType, channelType model.ChannelType) bool {
	m := map[store.SelectionType]model.ChannelType{
		store.SelectNotPublic:  model.ChannelTypeOpen,
		store.SelectNotPrivate: model.ChannelTypePrivate,
		store.SelectNotDirect:  model.ChannelTypeDirect,
		store.SelectNotGroup:   model.ChannelTypeGroup,
	}

	for s, t := range m {
		if isSelectionType(st, s) && channelType == t {
			return true
		}
	}

	return false
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

	var currChanId string
	if s.currentChannel != nil {
		currChanId = s.currentChannel.Id
	}

	var channels []*model.Channel
	for channelId, channel := range s.channels {
		if (currChanId == channelId) && isSelectionType(st, store.SelectNotCurrent) {
			continue
		}
		if excludeChannelType(st, channel.Type) {
			continue
		}
		_, isMember := s.channelMembers[channelId][userId]
		if (channel.Type == model.ChannelTypeOpen || channel.Type == model.ChannelTypePrivate) && channel.TeamId != teamId {
			continue
		}
		if isMember && isSelectionType(st, store.SelectMemberOf) {
			channels = append(channels, channel)
		}
		if !isMember && isSelectionType(st, store.SelectNotMemberOf) {
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

	return s.randomUser()
}

func (s *MemStore) randomUser() (model.User, error) {
	// We check if the current user is present in the stored map of users.
	// If so we increment by one minLen since we purposely skip the current user on selection.
	// This is done to avoid spinning indefinitely in case the store holds only one
	// user and that being the current one.
	minLen := 1
	if _, ok := s.users[s.user.Id]; ok {
		minLen++
	}
	if len(s.users) < minLen {
		return model.User{}, ErrLenMismatch
	}

	for {
		key, err := pickRandomKeyFromMap(s.users)
		if err != nil {
			return model.User{}, err
		}
		user := s.users[key.(string)]
		if user == nil || user.Id == "" {
			return model.User{}, ErrInvalidData
		}
		// We don't want to pick ourselves.
		if user.Id == s.user.Id {
			continue
		}
		return *user, nil
	}
}

// RandomUsers returns N random users from the set of users.
func (s *MemStore) RandomUsers(n int) ([]model.User, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	// We check if the current user is present in the stored map of users.
	// If so we decrement by one the maximum number of selectable users (numUsers)
	// since RandomUser() will never return the current one.
	// This is done to avoid spinning indefinitely when trying to pick N users in
	// a store of exactly N users and one of them being the current one.
	numUsers := len(s.users)
	if _, ok := s.users[s.user.Id]; ok {
		numUsers--
	}
	if n > numUsers {
		return nil, ErrLenMismatch
	}

	users := make([]model.User, 0, n)
	for len(users) < n {
		u, err := s.randomUser()
		if err != nil {
			return nil, err
		}
		var found bool
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

	var postIds []string
	for _, p := range s.posts {
		if p.Type == "" {
			postIds = append(postIds, p.Id)
		}
	}

	if len(postIds) == 0 {
		return model.Post{}, ErrPostNotFound
	}

	return *s.posts[postIds[rand.Intn(len(postIds))]].Clone(), nil
}

// RandomPostForChannel returns a random post for the given channel.
func (s *MemStore) RandomPostForChannel(channelId string) (model.Post, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var postIds []string
	for _, p := range s.posts {
		if p.ChannelId == channelId && p.Type == "" {
			postIds = append(postIds, p.Id)
		}
	}

	if len(postIds) == 0 {
		return model.Post{}, ErrPostNotFound
	}

	return *s.posts[postIds[rand.Intn(len(postIds))]].Clone(), nil
}

// RandomPostForChannelForUser returns a random post for the given channel made
// by the given user.
func (s *MemStore) RandomPostForChannelByUser(channelId, userId string) (model.Post, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var postIds []string
	for _, p := range s.posts {
		if p.ChannelId == channelId && p.UserId == userId && p.Type == "" {
			postIds = append(postIds, p.Id)
		}
	}

	if len(postIds) == 0 {
		return model.Post{}, ErrPostNotFound
	}

	return *s.posts[postIds[rand.Intn(len(postIds))]].Clone(), nil
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

// RandomThread returns a random post.
func (s *MemStore) RandomThread() (model.ThreadResponse, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if len(s.threads) == 0 {
		return model.ThreadResponse{}, ErrThreadNotFound
	}

	threadIds := make([]string, 0, len(s.threads))
	for _, t := range s.threads {
		threadIds = append(threadIds, t.PostId)
	}

	randomThread := s.threads[threadIds[rand.Intn(len(threadIds))]]
	return *cloneThreadResponse(randomThread, &model.ThreadResponse{}), nil
}
