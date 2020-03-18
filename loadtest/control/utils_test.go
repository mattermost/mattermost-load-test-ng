// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package control

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRandomizeUserName(t *testing.T) {
	name := RandomizeUserName("test-agent-1-user-4")
	assert.Regexp(t, regexp.MustCompile(`user[[:alpha:]]+-4`), name)

	name = RandomizeUserName("lt1-user4")
	assert.True(t, strings.HasPrefix(name, "lt1-user"))

	name = RandomizeUserName("testuser")
	assert.Equal(t, name, "testuser")
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
	search := "Lorem ipsum dolor sit amet, consectetur adipiscing elit."
	res := emulateUserTyping(search, func(term string) UserActionResponse {
		return UserActionResponse{Info: term}
	})
	require.Nil(t, res.Err)
	require.Equal(t, search, res.Info)
	text := ""
	i := 0
	res = emulateUserTyping(search, func(term string) UserActionResponse {
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
