// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"fmt"
	"os"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	seed := memstore.SetRandomSeed()
	fmt.Printf("Seed value is: %d\n", seed)
	os.Exit(m.Run())
}

func TestPickAction(t *testing.T) {
	t.Run("Empty slice", func(t *testing.T) {
		actions := []userAction{}
		action, err := pickAction(actions)
		require.Nil(t, action)
		require.Error(t, err)
	})

	t.Run("Zero frequency sum", func(t *testing.T) {
		actions := []userAction{
			{
				frequency: 0,
			},
			{
				frequency: 0,
			},
		}
		action, err := pickAction(actions)
		require.Nil(t, action)
		require.Error(t, err)
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
		action, err := pickAction(actions)
		require.NotNil(t, action)
		require.NoError(t, err)
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
			action, err := pickAction(actions)
			require.NotNil(t, action)
			require.NoError(t, err)

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

func TestSplitName(t *testing.T) {
	testCases := []struct {
		input, prefix, typed string
	}{
		{
			input:  "testuser-1",
			prefix: "testuser-",
			typed:  "1",
		},
		{
			input:  "testuser999",
			prefix: "testuser",
			typed:  "999",
		},
		{
			input:  "téstüser999",
			prefix: "téstüser",
			typed:  "999",
		},
		{
			input:  "testuser",
			prefix: "",
			typed:  "testuser",
		},
		{
			input:  "testuser-100a",
			prefix: "",
			typed:  "testuser-100a",
		},
	}
	for _, tc := range testCases {
		ePrefix, eTyped := splitName(tc.input)
		require.Equal(t, tc.prefix, ePrefix)
		require.Equal(t, tc.typed, eTyped)
	}
}
