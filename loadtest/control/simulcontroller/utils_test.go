// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"fmt"
	"math/rand"
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
		prefix, typed := splitName(tc.input)
		require.Equal(t, tc.prefix, prefix)
		require.Equal(t, tc.typed, typed)
	}
}

func TestGetCutoff(t *testing.T) {
	testCases := []struct {
		prefix, typed string
		cutoff        int
	}{
		{
			prefix: "testuser-",
			typed:  "1",
			cutoff: 11,
		},
		{
			prefix: "testuser",
			typed:  "999",
			cutoff: 10,
		},
		{
			prefix: "téstüser",
			typed:  "999",
			cutoff: 12,
		},
		{
			prefix: "",
			typed:  "testuser",
			cutoff: 5,
		},
		{
			prefix: "",
			typed:  "testuser-100a",
			cutoff: 7,
		},
	}
	// custom rand with fixed source for deterministic values
	// without polluting global rand
	newRand := rand.New(rand.NewSource(1))
	for _, tc := range testCases {
		require.Equal(t, tc.cutoff, getCutoff(tc.prefix, tc.typed, newRand))
	}
}

func TestPickIds(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		ids := pickIds([]string{}, 1)
		require.Empty(t, ids)
	})

	t.Run("not enough elements", func(t *testing.T) {
		ids := pickIds([]string{"id0"}, 2)
		require.Empty(t, ids)
	})

	t.Run("one element", func(t *testing.T) {
		ids := pickIds([]string{"id0"}, 1)
		require.Len(t, ids, 1)
		require.Equal(t, "id0", ids[0])
	})

	t.Run("two elements", func(t *testing.T) {
		input := []string{"id0", "id1"}
		ids := pickIds(input, 1)
		require.Len(t, ids, 1)
		require.Contains(t, input, ids[0])

		ids = pickIds(input, 2)
		require.Len(t, ids, 2)
		require.Contains(t, ids, "id0")
		require.Contains(t, ids, "id1")
	})
}

func TestExtractMentionFromMessage(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "",
			expected: "",
		},
		{
			input:    "@",
			expected: "",
		},
		{
			input:    "@ ",
			expected: "",
		},
		{
			input:    "@@",
			expected: "",
		},
		{
			input:    "@;/",
			expected: "",
		},
		{
			input:    "@user",
			expected: "user",
		},
		{
			input:    "@user ",
			expected: "user",
		},
		{
			input:    "@user1",
			expected: "user1",
		},
		{
			input:    "@user1-0",
			expected: "user1-0",
		},
		{
			input:    "@1user1-0",
			expected: "1user1-0",
		},
		{
			input:    "@user_test",
			expected: "user_test",
		},
		{
			input:    "@user.test",
			expected: "user.test",
		},
		{
			input:    "someone mentioned @user",
			expected: "user",
		},
	}

	for _, tc := range testCases {
		require.Equal(t, tc.expected, extractMentionFromMessage(tc.input))
	}
}
