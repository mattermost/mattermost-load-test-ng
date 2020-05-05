// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"math"
	"math/rand"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

// pickAction randomly selects an action from a slice of userAction with
// probability proportional to the action's frequency.
func pickAction(actions []userAction) (*userAction, error) {
	var sum int
	if len(actions) == 0 {
		return nil, errors.New("actions cannot be empty")
	}
	for i := range actions {
		sum += actions[i].frequency
	}
	if sum == 0 {
		return nil, errors.New("actions frequency sum cannot be zero")
	}
	distance := rand.Intn(sum)
	for i := range actions {
		distance -= actions[i].frequency
		if distance < 0 {
			return &actions[i], nil
		}
	}
	return nil, errors.New("should not be able to reach this point")
}

func genMessage(u user.User, isReply bool) (string, error) {
	// This is an estimate that comes from stats on community servers.
	// The average length (in words) for a reply.
	// TODO: should be part of some advanced configuration.
	avgWordCount := 35
	minWordCount := 1

	if isReply {
		avgWordCount = 24
	}

	// TODO: make a util function out of this behaviour.
	wordCount := rand.Intn(avgWordCount*2-minWordCount*2) + minWordCount

	message := control.GenerateRandomSentences(wordCount)

	// 2% of the times someone is mentioned.
	if rand.Float64() < 0.02 {
		if resp := control.AutoCompleteUsers(u); resp.Err != nil && resp.Info != "" {
			message += " @" + resp.Info
		} else if resp.Err != nil {
			return "", resp.Err
		}
	}
	return message, nil
}

func pickIdleTimeMs(minIdleTimeMs, avgIdleTimeMs int, rate float64) time.Duration {
	// Randomly selecting a value in the interval
	// [minIdleTimeMs, avgIdleTimeMs*2 - minIdleTimeMs).
	// This will give us an expected value equal to avgIdleTimeMs.
	// TODO: consider if it makes more sense to select this value using
	// a truncated normal distribution.
	idleMs := rand.Intn(avgIdleTimeMs*2-minIdleTimeMs*2) + minIdleTimeMs
	idleTimeMs := time.Duration(math.Round(float64(idleMs) * rate))

	return idleTimeMs * time.Millisecond
}
