// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPickAction(t *testing.T) {
	t.Run("Empty slice", func(t *testing.T) {
		actions := []userAction{}
		action := pickAction(actions)
		require.Nil(t, action)
	})

	t.Run("Zero frequency action", func(t *testing.T) {
		actions := []userAction{
			{
				frequency: 1,
			},
			{
				frequency: 0,
			},
			{
				frequency: 1,
			},
		}
		action := pickAction(actions)
		require.NotNil(t, action)
		require.Condition(t, func() bool {
			switch action {
			case &actions[0], &actions[2]:
				return true
			default:
				return false
			}
		})
	})

	t.Run("Different frequencies", func(t *testing.T) {
		actions := []userAction{
			{
				frequency: 1,
			},
			{
				frequency: 100,
			},
			{
				frequency: 0,
			},
			{
				frequency: 10,
			},
		}

		res := map[int]int{
			0: 0,
			1: 0,
			2: 0,
			3: 0,
		}

		for i := 0; i < 1000; i++ {
			action := pickAction(actions)
			require.NotNil(t, action)

			switch action {
			case &actions[0]:
				res[0]++
			case &actions[1]:
				res[1]++
			case &actions[2]:
				res[2]++
			case &actions[3]:
				res[3]++
			}
		}

		require.Zero(t, res[2])
		require.Greater(t, res[3], res[0])
		require.Greater(t, res[1], res[3])
	})
}
