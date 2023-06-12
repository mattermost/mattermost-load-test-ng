// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package gencontroller

import (
	"errors"
	"math/rand"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
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

var errMemberLimitExceeded = errors.New("member limit exceeded")

// chooseChannel will pick a channelID randomly from the range of indexes.
// If the chosen channelID has exceeded the number of channelmembers, it will
// select another one in the range until it has found one.
func chooseChannel(dist []ChannelMemberDistribution, idx int, u user.User) (string, error) {
	minIndexRange := 0.0
	for i := 0; i < idx; i++ {
		minIndexRange += dist[i].PercentChannels
	}
	maxIndexRange := minIndexRange + dist[idx].PercentChannels
	minIndex := int(minIndexRange * float64(len(st.channels)))
	maxIndex := int(maxIndexRange * float64(len(st.channels)))

	if maxIndex-minIndex <= 1 {
		return "", errors.New("not enough channels to select from; either increase range or increase number of channels to create")
	}

	var channelID string
	maxTimes := maxIndex - minIndex
	cnt := 0
	for {
		if cnt == maxTimes {
			return "", errMemberLimitExceeded
		}
		target := rand.Intn(maxIndex-minIndex) + minIndex
		// target is guaranteed to be within bounds of st.channels
		channelID = st.channels[target]

		members, err := u.Store().ChannelMembers(channelID)
		if err != nil {
			return "", err
		}

		// A MemberLimit of 0 means there is no limit
		limit := int(dist[idx].MemberLimit)
		if limit > 0 && len(members) > limit {
			cnt++
			continue
		}

		return channelID, nil
	}
}
