// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"errors"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost/server/public/model"
	"math/rand"
)

var (
	ErrEmptyMap                = errors.New("memstore: cannot select from an empty map")
	ErrEmptySlice              = errors.New("memstore: cannot select from an empty slice")
	ErrLenMismatch             = errors.New("memstore: cannot select from a map, not enough elements")
	ErrTeamNotFound            = errors.New("memstore: team not found")
	ErrUserNotSet              = errors.New("memstore: user is not set")
	ErrTeamStoreEmpty          = errors.New("memstore: team store is empty")
	ErrChannelStoreEmpty       = errors.New("memstore: channel store is empty")
	ErrChannelNotFound         = errors.New("memstore: channel not found")
	ErrPostNotFound            = errors.New("memstore: post not found")
	ErrInvalidData             = errors.New("memstore: invalid data found")
	ErrThreadNotFound          = errors.New("memstore: thread not found")
	ErrMaxAttempts             = errors.New("memstore: maximum number of attempts tried")
	ErrDraftNotFound           = errors.New("memstore: draft not found")
	ErrScheduledPostStoreEmpty = errors.New("memstore: scheduled post store is empty")
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
		user := s.users[key]
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
func (s *MemStore) RandomPost(st store.SelectionType) (model.Post, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var postIds []string

	// Simplest case: just get a random user post from s.posts
	if isSelectionType(st, store.SelectAny) {
		for _, p := range s.posts {
			if p.Type == "" {
				postIds = append(postIds, p.Id)
			}
		}
	} else {
		// Preflight checks: make sure that both user and current channel are set
		if s.user == nil {
			return model.Post{}, ErrUserNotSet
		}
		var currChanId string
		if s.currentChannel != nil {
			currChanId = s.currentChannel.Id
		}

		for _, p := range s.posts {
			if p.Type != "" {
				continue
			}

			channel := s.channels[p.ChannelId]
			if channel == nil {
				continue
			}
			if (currChanId == channel.Id) && isSelectionType(st, store.SelectNotCurrent) {
				continue
			}
			if excludeChannelType(st, channel.Type) {
				continue
			}
			if !isSelectionType(st, store.SelectMemberOf) && !isSelectionType(st, store.SelectNotMemberOf) {
				postIds = append(postIds, p.Id)
			} else {
				_, isMember := s.channelMembers[channel.Id][s.user.Id]
				if isMember && isSelectionType(st, store.SelectMemberOf) {
					postIds = append(postIds, p.Id)
				}
				if !isMember && isSelectionType(st, store.SelectNotMemberOf) {
					postIds = append(postIds, p.Id)
				}
			}
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

// RandomReplyPostForChannel returns a random reply post for the given channel.
func (s *MemStore) RandomReplyPostForChannel(channelId string) (model.Post, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var postIds []string
	for _, p := range s.posts {
		if p.ChannelId == channelId && p.Type == "" && p.RootId != "" {
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
	return *chanMemberMap[key], nil
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
	return *teamMemberMap[key], nil
}

func (s *MemStore) RandomCategory(teamID string) (model.SidebarCategoryWithChannels, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	teamCat := s.sidebarCategories[teamID]

	key, err := pickRandomKeyFromMap(teamCat)
	if err != nil {
		return model.SidebarCategoryWithChannels{}, err
	}

	category := *teamCat[key]
	tmp := make([]string, len(category.Channels))
	copy(tmp, category.Channels)
	category.Channels = tmp
	return category, nil
}

func pickRandomKeyFromMap[K comparable, V any](m map[K]V) (K, error) {
	var def K
	if len(m) == 0 {
		return def, ErrEmptyMap
	}
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	idx := rand.Intn(len(m))
	return keys[idx], nil
}

// RandomThread returns a random post.
func (s *MemStore) RandomThread() (model.ThreadResponse, error) {
	s.lock.RLock()
	threads, err := s.getThreads(false)
	s.lock.RUnlock()
	if err != nil {
		return model.ThreadResponse{}, err
	}
	if len(threads) == 0 {
		return model.ThreadResponse{}, ErrThreadNotFound
	}
	return *threads[rand.Intn(len(threads))], nil
}

// RandomDraftForTeam returns a random draft id for the given team
func (s *MemStore) RandomDraftForTeam(teamId string) (string, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var draftIDs []string
	for _, draft := range s.drafts[teamId] {
		if draft.UserId == s.user.Id {
			rootID := draft.RootId
			if rootID == "" {
				rootID = draft.ChannelId
			}
			draftIDs = append(draftIDs, rootID)
		}
	}

	if len(draftIDs) == 0 {
		return "", ErrDraftNotFound
	}

	return draftIDs[rand.Intn(len(draftIDs))], nil
}

func (s *MemStore) GetRandomScheduledPost() (*model.ScheduledPost, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	// Check if scheduledPosts is empty
	if len(s.scheduledPosts) == 0 {
		return &model.ScheduledPost{}, ErrScheduledPostStoreEmpty
	}

	var keys []string
	for key, innerMap := range s.scheduledPosts {
		if len(innerMap) > 0 {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		return &model.ScheduledPost{}, ErrScheduledPostStoreEmpty
	}

	selectedInnerMap := s.scheduledPosts[keys[rand.Intn(len(keys))]]

	// Pick a random index for the inner map
	randomInnerIndex := rand.Intn(len(selectedInnerMap))
	var selectedPost *model.ScheduledPost
	innerIndex := 0
	for _, post := range selectedInnerMap {
		if innerIndex == randomInnerIndex {
			selectedPost = post[rand.Intn(len(post))]
			break
		}
		innerIndex++
	}

	return selectedPost, nil
}
