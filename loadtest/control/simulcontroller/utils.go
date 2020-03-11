// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"math/rand"
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
