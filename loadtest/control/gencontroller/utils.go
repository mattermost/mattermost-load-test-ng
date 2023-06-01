// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package gencontroller

import (
	"errors"
	"math/rand"
)

// pickAction randomly selects an action from a map of userAction with
// probability proportional to the action's frequency.
func pickAction(actions map[string]userAction) (*userAction, error) {
	var sum int
	if len(actions) == 0 {
		return nil, errors.New("actions cannot be empty")
	}
	for id := range actions {
		sum += actions[id].frequency
	}
	if sum == 0 {
		return nil, errors.New("actions frequency sum cannot be zero")
	}
	distance := rand.Intn(sum)
	for id := range actions {
		distance -= actions[id].frequency
		if distance < 0 {
			action := actions[id]
			return &action, nil
		}
	}
	return nil, errors.New("should not be able to reach this point")
}

const MaxLongRunningThreadsPerChannel = 2

// shouldMakeLongRunningThreads returns if a long thread should be created
// TODO: The rates and logic in this function should be made configurable
func shouldMakeLongRunningThread(channelId string) bool {
	// 2% of the the time we check if we should make a long running thread
	// this way we don't make all long running threads near the start
	if rand.Float64() > 0.02 {
		return false
	}
	// limit the maximum number of long running threads in any channel
	if len(st.getLongRunningThreadsInChannel(channelId)) >= MaxLongRunningThreadsPerChannel {
		return false
	}
	return true
}
