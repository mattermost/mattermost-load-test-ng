// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"math/rand"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
)

var errNoMatch = errors.New("could not match username")

// pickAction randomly selects an action from a slice of userAction with
// probability proportional to the action's frequency.
func pickAction(actions []userAction) (*userAction, error) {
	weights := make([]int, len(actions))
	for i := range actions {
		weights[i] = actions[i].frequency
	}

	idx, err := control.SelectWeighted(weights)
	if err != nil {
		return nil, err
	}

	return &actions[idx], nil
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

	message := control.GenerateRandomSentences(wordCount)

	return message
}

func splitName(name string) (string, string) {
	typed := user.TestUserSuffixRegexp.FindString(name)
	var prefix string
	if typed == "" {
		typed = name
	} else {
		prefix = strings.TrimSuffix(name, typed)
	}
	return prefix, typed
}

func getCutoff(prefix, typed string, altRand *rand.Rand) int {
	cutoff := len(prefix) + 2
	switch {
	case len(typed)/2 > 0 && altRand != nil:
		return cutoff + altRand.Intn(len(typed)/2)
	case len(typed)/2 > 0:
		return cutoff + rand.Intn(len(typed)/2)
	default:
		return cutoff
	}
}

func emulateMention(teamId, channelId, name string, auto func(teamId, channelId, username string, limit int) (map[string]bool, error)) error {
	found := errors.New("found") // will be used to halt emulate typing function

	prefix, typed := splitName(name)
	cutoff := getCutoff(prefix, typed, nil)
	resp := control.EmulateUserTyping(typed, func(term string) control.UserActionResponse {
		term = prefix + term
		users, err := auto(teamId, channelId, term, 100)
		if err != nil {
			return control.UserActionResponse{Err: err}
		}

		if len(users) == 1 {
			return control.UserActionResponse{Err: found, Info: name}
		}

		if len(term) == cutoff {
			return control.UserActionResponse{Err: found, Info: name}
		}

		return control.UserActionResponse{Info: "user not found"}
	})

	if errors.Is(resp.Err, found) {
		return nil
	} else if resp.Err != nil {
		return resp.Err
	}

	return errNoMatch
}
