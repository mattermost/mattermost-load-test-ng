// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package control

import (
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
	search := "text"

	stop := make(chan bool)

	var result string
	for r := range emulateUserTyping(search, stop) {
		fmt.Print(string(r))
		result += string(r)
	}
	require.Equal(t, search, result)

	result = ""
	stop2 := make(chan bool)

	for r := range emulateUserTyping(search, stop2) {
		fmt.Print(string(r))
		result += string(r)

		if len(result) == 2 {
			stop2 <- true
			break
		}
	}
	require.Equal(t, search[:2], result)
}
