package simplecontroller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandomizeUserName(t *testing.T) {

	name := randomizeUserName("test-agent-1-user-4")
	assert.Contains(t, name, "test-agent-1-user")

	name = randomizeUserName("lt1-user4")
	assert.Contains(t, name, "lt1-user")
}
