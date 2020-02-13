package simplecontroller

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomizeUserName(t *testing.T) {

	name := randomizeUserName("test-agent-1-user-4")
	assert.Regexp(t, regexp.MustCompile(`user[[:alpha:]]+-4`), name)

	name = randomizeUserName("lt1-user4")
	assert.True(t, strings.HasPrefix(name, "lt1-user"))

	name = randomizeUserName("testuser")
	assert.Equal(t, name, "testuser")
}
