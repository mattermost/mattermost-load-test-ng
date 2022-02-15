// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"sync"
)

type ThreadInfo struct {
	Id        string
	ChannelId string
	TeamId    string
}

type state struct {
	// longRunningThreads map from ThreadId to ThreadInfo
	longRunningThreads map[string]*ThreadInfo
	mut                sync.RWMutex
}

var st *state

func init() {
	st = &state{
		longRunningThreads: make(map[string]*ThreadInfo),
	}
}

func copyThreadInfo(src *ThreadInfo) *ThreadInfo {
	dst := &ThreadInfo{}
	dst.Id = src.Id
	dst.ChannelId = src.ChannelId
	dst.TeamId = src.TeamId
	return dst
}

func (st *state) setLongRunningThread(id, channelId, teamId string) {
	st.mut.Lock()
	defer st.mut.Unlock()
	st.longRunningThreads[id] = &ThreadInfo{
		Id:        id,
		ChannelId: channelId,
		TeamId:    teamId,
	}
}

func (st *state) getLongRunningThreadsInChannel(channelId string) []*ThreadInfo {
	st.mut.Lock()
	defer st.mut.Unlock()
	threadInfos := make([]*ThreadInfo, 0, len(st.longRunningThreads))
	for _, ti := range st.longRunningThreads {
		if ti.ChannelId == channelId {
			threadInfos = append(threadInfos, copyThreadInfo(ti))
		}
	}
	return threadInfos
}
