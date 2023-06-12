// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gencontroller

import (
	"sync"
)

type state struct {
	targets               map[string]int64
	targetsMut            sync.RWMutex
	longRunningThreads    map[string]*ThreadInfo
	longRunningThreadsMut sync.RWMutex
	// This is used to store the global list of channelIDs for the agents
	// to choose from while trying to join a channel. This only contains Open/Private
	// channels.
	channels              []string
	channelsMut           sync.Mutex
	followedThreadsByUser map[string]map[string]bool
	followedThreadsMut    sync.RWMutex
	// Set of all the DMs created. The first key is the user whose ID
	// is lexicographically lower. So if there is DM between users "A"
	// and "B", there will be an entry userDMs["A"]["B"] == true.
	userDMs    map[string]map[string]bool
	userDMsMut sync.RWMutex
}

type ThreadInfo struct {
	Id        string
	ChannelId string
	TeamId    string
}

var st *state

const (
	StateTargetTeams             = "teams"
	StateTargetChannelsDM        = "channelsDM"
	StateTargetChannelsGM        = "channelsGM"
	StateTargetChannelsPublic    = "channelsPublic"
	StateTargetChannelsPrivate   = "channelsPrivate"
	StateTargetPosts             = "posts"
	StateTargetReactions         = "reactions"
	StateTargetPostReminders     = "postreminders"
	StateTargetSidebarCategories = "sidebarcategories"
	StateTargetFollowedThreads   = "followedthreads"
)

func init() {
	st = &state{
		targets: map[string]int64{
			StateTargetTeams:             0,
			StateTargetChannelsDM:        0,
			StateTargetChannelsGM:        0,
			StateTargetChannelsPublic:    0,
			StateTargetChannelsPrivate:   0,
			StateTargetPosts:             0,
			StateTargetReactions:         0,
			StateTargetPostReminders:     0,
			StateTargetSidebarCategories: 0,
			StateTargetFollowedThreads:   0,
		},
		longRunningThreads:    make(map[string]*ThreadInfo),
		channels:              []string{},
		followedThreadsByUser: make(map[string]map[string]bool),
		userDMs:               make(map[string]map[string]bool),
	}
}

func (st *state) inc(targetId string, targetVal int64) bool {
	st.targetsMut.Lock()
	defer st.targetsMut.Unlock()
	if st.targets[targetId] == targetVal {
		return false
	}
	st.targets[targetId]++
	return true
}

func (st *state) dec(targetId string) {
	st.targetsMut.Lock()
	defer st.targetsMut.Unlock()
	st.targets[targetId]--
}

func (st *state) get(targetId string) int64 {
	st.targetsMut.RLock()
	defer st.targetsMut.RUnlock()
	return st.targets[targetId]
}

func (st *state) setLongRunningThread(id, channelId, teamId string) {
	st.longRunningThreadsMut.Lock()
	defer st.longRunningThreadsMut.Unlock()
	st.longRunningThreads[id] = &ThreadInfo{
		Id:        id,
		ChannelId: channelId,
		TeamId:    teamId,
	}
}

func (st *state) getLongRunningThreadsInChannel(channelId string) []*ThreadInfo {
	st.longRunningThreadsMut.Lock()
	defer st.longRunningThreadsMut.Unlock()
	var threadInfos []*ThreadInfo
	for _, ti := range st.longRunningThreads {
		if ti.ChannelId == channelId {
			threadInfos = append(threadInfos, copyThreadInfo(ti))
		}
	}
	return threadInfos
}

func (st *state) storeChannelID(channelID string) {
	st.channelsMut.Lock()
	defer st.channelsMut.Unlock()
	st.channels = append(st.channels, channelID)
}

func (st *state) isThreadFollowedByUser(threadId, userId string) bool {
	st.followedThreadsMut.RLock()
	defer st.followedThreadsMut.RUnlock()
	return st.followedThreadsByUser[userId] != nil && st.followedThreadsByUser[userId][threadId]
}

func (st *state) setThreadFollowedByUser(threadId, userId string) {
	st.followedThreadsMut.Lock()
	defer st.followedThreadsMut.Unlock()
	if _, ok := st.followedThreadsByUser[userId]; !ok {
		st.followedThreadsByUser[userId] = make(map[string]bool)
	}
	st.followedThreadsByUser[userId][threadId] = true
}

func copyThreadInfo(src *ThreadInfo) *ThreadInfo {
	dst := &ThreadInfo{}
	dst.Id = src.Id
	dst.ChannelId = src.ChannelId
	dst.TeamId = src.TeamId
	return dst
}

func orderUsers(user1, user2 string) (string, string) {
	if user1 < user2 {
		return user1, user2
	}

	return user2, user1
}

func (st *state) setDM(user1, user2 string) {
	st.userDMsMut.Lock()
	defer st.userDMsMut.Unlock()

	user1, user2 = orderUsers(user1, user2)

	if _, ok := st.userDMs[user1]; !ok {
		st.userDMs[user1] = make(map[string]bool)
	}

	st.userDMs[user1][user2] = true
}

func (st *state) numDMs(user string) int {
	st.userDMsMut.RLock()
	defer st.userDMsMut.RUnlock()

	return len(st.userDMs[user])
}

func (st *state) dmExists(user1, user2 string) bool {
	st.userDMsMut.RLock()
	defer st.userDMsMut.RUnlock()

	user1, user2 = orderUsers(user1, user2)

	dms, ok := st.userDMs[user1]
	return ok && dms[user2]
}
