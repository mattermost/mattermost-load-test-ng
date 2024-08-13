// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package control

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	seed := memstore.SetRandomSeed()
	fmt.Printf("Seed value is: %d\n", seed)
	os.Exit(m.Run())
}

func TestRandomizeUserName(t *testing.T) {
	name := RandomizeUserName("test-agent-1-user-4")
	assert.Regexp(t, regexp.MustCompile(`user[[:alpha:]]+-4`), name)

	name = RandomizeUserName("lt1-user4")
	assert.True(t, strings.HasPrefix(name, "lt1-user"))

	name = RandomizeUserName("testuser")
	assert.Equal(t, name, "testuser")
}

func TestRandomizeTeamDisplayName(t *testing.T) {
	name := RandomizeTeamDisplayName("badname")
	assert.Equal(t, "badname", name)

	name = RandomizeTeamDisplayName("team9")
	assert.True(t, strings.HasPrefix(name, "team9-"))

	name = RandomizeTeamDisplayName("team9-k")
	assert.True(t, strings.HasPrefix(name, "team9-"))
}

func TestGetErrOrigin(t *testing.T) {
	var origin string
	test := func() {
		origin = getErrOrigin()
	}
	test()
	fmt.Println(origin)
	require.True(t, strings.HasPrefix(origin, "control.TestGetErrOrigin"))
}

func TestEmulateUserTyping(t *testing.T) {
	search := "this is long enough"
	res := EmulateUserTyping(search, func(term string) UserActionResponse {
		return UserActionResponse{Info: term}
	})
	require.Nil(t, res.Err)
	require.Equal(t, search, res.Info)
	text := ""
	i := 0
	res = EmulateUserTyping(search, func(term string) UserActionResponse {
		text = term
		if i == 2 {
			return UserActionResponse{Err: errors.New("an error")}
		}
		i++
		return UserActionResponse{Info: text}
	})
	require.NotNil(t, res.Err)
	require.Equal(t, "an error", res.Err.Error())
}

func TestGenerateRandomSentences(t *testing.T) {
	randomize := GenerateRandomSentences(8)
	s := strings.Split(randomize, " ")
	require.Len(t, s, 8)

	randomize = GenerateRandomSentences(0)
	s = strings.Split(randomize, " ")
	require.Len(t, s, 1)
	require.Equal(t, s[0], "🙂")
}

func TestAddLink(t *testing.T) {
	msg := "hello world"
	out := AddLink(msg)
	words := strings.Split(out, " ")
	require.Len(t, words, 4)
	assert.Contains(t, links, words[2])
}

func TestSelectWeighted(t *testing.T) {
	t.Run("empty weights", func(t *testing.T) {
		idx, err := SelectWeighted([]int{})
		require.Error(t, err)
		require.Equal(t, -1, idx)
	})

	t.Run("zero sum", func(t *testing.T) {
		weights := []int{
			0,
			0,
			0,
		}
		idx, err := SelectWeighted(weights)
		require.Error(t, err)
		require.Equal(t, -1, idx)
	})

	t.Run("weighted selection", func(t *testing.T) {
		weights := []int{
			1000,
			100,
			10,
		}

		distribution := make(map[int]int, len(weights))

		n := 10000
		for i := 0; i < n; i++ {
			idx, err := SelectWeighted(weights)
			require.NoError(t, err)
			distribution[idx]++
		}

		require.Greater(t, distribution[0], distribution[1])
		require.Greater(t, distribution[1], distribution[2])
	})
}

func TestGeneratePostsSearchTerm(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		var opts PostsSearchOpts
		words := []string{}
		term := GeneratePostsSearchTerm(words, opts)
		require.Empty(t, term)
	})

	t.Run("simple", func(t *testing.T) {
		var opts PostsSearchOpts
		words := []string{"one", "two"}
		term := GeneratePostsSearchTerm(words, opts)
		require.Equal(t, "one two", term)
	})

	t.Run("from modifier", func(t *testing.T) {
		opts := PostsSearchOpts{
			From: "user1",
		}
		words := []string{"one"}
		term := GeneratePostsSearchTerm(words, opts)
		require.Equal(t, "from:user1 one", term)
	})

	t.Run("in modifier", func(t *testing.T) {
		opts := PostsSearchOpts{
			In: "town-square",
		}
		words := []string{"one"}
		term := GeneratePostsSearchTerm(words, opts)
		require.Equal(t, "in:town-square one", term)
	})

	t.Run("on modifier", func(t *testing.T) {
		now := time.Now()
		opts := PostsSearchOpts{
			On: now,
		}
		words := []string{"one"}
		term := GeneratePostsSearchTerm(words, opts)
		require.Equal(t, fmt.Sprintf("on:%s one", now.Format("2006-01-02")), term)
	})

	t.Run("before modifier", func(t *testing.T) {
		now := time.Now()
		opts := PostsSearchOpts{
			Before: now,
		}
		words := []string{"one"}
		term := GeneratePostsSearchTerm(words, opts)
		require.Equal(t, fmt.Sprintf("before:%s one", now.Format("2006-01-02")), term)
	})

	t.Run("after modifier", func(t *testing.T) {
		now := time.Now()
		opts := PostsSearchOpts{
			After: now,
		}
		words := []string{"one"}
		term := GeneratePostsSearchTerm(words, opts)
		require.Equal(t, fmt.Sprintf("after:%s one", now.Format("2006-01-02")), term)
	})

	t.Run("excluded words", func(t *testing.T) {
		opts := PostsSearchOpts{
			Excluded: []string{"two", "three"},
		}
		words := []string{"one"}
		term := GeneratePostsSearchTerm(words, opts)
		require.Equal(t, "-two -three one", term)
	})

	t.Run("phrase", func(t *testing.T) {
		opts := PostsSearchOpts{
			IsPhrase: true,
		}
		words := []string{"one", "two"}
		term := GeneratePostsSearchTerm(words, opts)
		require.Equal(t, "\"one two\"", term)
	})

	t.Run("mixed", func(t *testing.T) {
		now := time.Now()
		opts := PostsSearchOpts{
			From: "user1",
			In:   "town-square",
			On:   now,
		}
		words := []string{"one"}
		term := GeneratePostsSearchTerm(words, opts)
		require.Equal(t, fmt.Sprintf("from:user1 in:town-square on:%s one", now.Format("2006-01-02")), term)
	})
}

func TestParseServerVersion(t *testing.T) {
	testCases := []struct {
		name            string
		version         string
		expectedVersion semver.Version
		expectedErr     string
	}{
		{
			name:        "Empty string",
			version:     "",
			expectedErr: "Version string empty",
		},
		{
			name:        "Non-empty but invalid string",
			version:     "invalid",
			expectedErr: "Version string empty",
		},
		{
			name:            "Valid string",
			version:         "5.30.1",
			expectedVersion: semver.MustParse("5.30.1"),
			expectedErr:     "",
		},
		{
			name:            "Valid string with pre-version",
			version:         "5.30.0.dev.d74e4887bd588dbe342a45c77d6dc52a.false",
			expectedVersion: semver.MustParse("5.30.0"),
			expectedErr:     "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sv, err := ParseServerVersion(tc.version)
			if tc.expectedErr == "" {
				require.Nil(t, err)
			} else {
				require.Equal(t, tc.expectedErr, err.Error())
			}
			require.Equal(t, tc.expectedVersion, sv)
		})
	}
}
