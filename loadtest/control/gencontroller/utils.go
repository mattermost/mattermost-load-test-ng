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
