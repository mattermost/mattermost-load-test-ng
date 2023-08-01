// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"math"
	"math/rand"
	"regexp"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost/server/public/model"
)

var errNoMatch = errors.New("could not match username")
var userMentionRe = regexp.MustCompile(`@[a-z0-9_.-]+`)

// pickAction randomly selects an action from a slice of userAction with
// probability proportional to the action's frequency.
func pickAction(actions []userAction) (*userAction, error) {
	if len(actions) == 0 {
		return nil, errors.New("failed to pick action: slice is empty")
	}

	weights := make([]int, len(actions))

	// finding the minimum, non-zero frequency.
	var minFreq float64
	for _, action := range actions {
		if minFreq == 0 && action.frequency > 0 {
			minFreq = action.frequency
		} else if action.frequency < minFreq && action.frequency > 0 {
			minFreq = action.frequency
		}
	}

	if minFreq == 0 {
		return nil, errors.New("all actions have zero frequency")
	}

	for i := range actions {
		weights[i] = int(math.Round(actions[i].frequency / minFreq))
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

func emulateMention(u user.User, teamId, channelId, name string, auto func(teamId, channelId, username string, limit int) (map[string]bool, error)) error {
	found := errors.New("found") // will be used to halt emulate typing function

	prefix, typed := splitName(name)
	cutoff := getCutoff(prefix, typed, nil)
	resp := control.EmulateUserTyping(typed, func(term string) control.UserActionResponse {
		term = prefix + term
		users, err := auto(teamId, channelId, term, 100)
		if err != nil {
			return control.UserActionResponse{Err: err}
		}

		if err := getProfileImageForUsers(u, keys(users)); err != nil {
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

func pickIds(input []string, n int) []string {
	var ids []string
	l := len(input)
	if l < n {
		return ids
	}

	ids = make([]string, n)
	for i := 0; i < n; i++ {
		idx := rand.Intn(l)
		ids[i] = input[idx]

		// remove picked element
		input[l-1], input[idx] = input[idx], input[l-1]
		input = input[:l-1]
		l--
	}

	return ids
}

func extractMentionFromMessage(msg string) string {
	mention := userMentionRe.FindString(msg)
	if mention == "" {
		return mention
	}
	return mention[1:]
}

// findIndex returns the index of needle in a haystack.
func findIndex(haystack []string, needle string) int {
	for i := range haystack {
		if haystack[i] == needle {
			return i
		}
	}
	return -1
}

func keys[K comparable, V any](in map[K]V) []K {
	keys := make([]K, 0, len(in))
	for k := range in {
		keys = append(keys, k)
	}
	return keys
}

func getPermalinkPostIDFromMessage(m string) string {
	index := strings.Index(m, "/pl/")
	if index == -1 {
		return ""
	}
	// If there are multiple permalinks found, we will click on the last one
	// in the channel. This naturally leads to an effect of having clicked all
	// permalinks which emulates organic behavior.
	start := index + len("/pl/")
	idLen := 26 // All IDs are always 26-char long
	postID := m[start : start+idLen]
	return postID
}

// usersForPosts returns a map of ids containing
// the unique set of user ids who are the creators
// for the list of posts passed.
func usersForPosts(posts []*model.Post) map[string]bool {
	userIds := make(map[string]bool)
	for _, p := range posts {
		userIds[p.UserId] = true
	}
	return userIds
}
