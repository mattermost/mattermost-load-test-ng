// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"math/rand"
)

// pickAction randomly selects an action from a slice of userAction with
// probability proportional to the action's frequency.
func pickAction(actions []userAction) *userAction {
	var sum int
	if len(actions) == 0 {
		return nil
	}
	for i := range actions {
		sum += actions[i].frequency
	}
	distance := rand.Intn(sum)
	for i := range actions {
		distance -= actions[i].frequency
		if distance < 0 {
			return &actions[i]
		}
	}
	return nil
}
