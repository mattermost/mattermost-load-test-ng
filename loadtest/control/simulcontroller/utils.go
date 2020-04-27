// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"math/rand"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
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

func genMessage(isReply bool) string {
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

	return control.GenerateRandomSentences(wordCount)
}
