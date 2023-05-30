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
}

type ThreadInfo struct {
	Id        string
	ChannelId string
	TeamId    string
}

var st *state

const (
	StateTargetTeams         = "teams"
	StateTargetChannels      = "channels"
	StateTargetPosts         = "posts"
	StateTargetReactions     = "reactions"
	StateTargetPostReminders = "postreminders"
)

func init() {
	st = &state{
		targets: map[string]int64{
			StateTargetTeams:         0,
			StateTargetChannels:      0,
			StateTargetPosts:         0,
			StateTargetReactions:     0,
			StateTargetPostReminders: 0,
		},
		longRunningThreads: make(map[string]*ThreadInfo),
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

func copyThreadInfo(src *ThreadInfo) *ThreadInfo {
	dst := &ThreadInfo{}
	dst.Id = src.Id
	dst.ChannelId = src.ChannelId
	dst.TeamId = src.TeamId
	return dst
}
