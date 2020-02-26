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
		origin = GetErrOrigin()
	}
	test()
	fmt.Println(origin)
	require.True(t, strings.HasPrefix(origin, "control.TestGetErrOrigin"))
}
