// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gencontroller

import (
	"sync"
)

type state struct {
	targets map[string]int64
	mut     sync.Mutex
}

var st *state

func init() {
	st = &state{
		targets: map[string]int64{
			"teams":     0,
			"channels":  0,
			"posts":     0,
			"reactions": 0,
		},
	}
}

func (st *state) inc(targetId string, targetVal int64) bool {
	st.mut.Lock()
	defer st.mut.Unlock()
	if st.targets[targetId] == targetVal {
		return false
	}
	st.targets[targetId]++
	return true
}

func (st *state) dec(targetId string) {
	st.mut.Lock()
	defer st.mut.Unlock()
	st.targets[targetId]--
}

func (st *state) get(targetId string) int64 {
	st.mut.Lock()
	defer st.mut.Unlock()
	return st.targets[targetId]
}
