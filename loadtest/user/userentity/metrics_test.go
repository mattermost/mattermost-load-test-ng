package userentity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSimplifyPath(t *testing.T) {
	testCases := map[string]string{
		"":                           "",
		"path":                       "path",
		"idsfx9p4y3do3bd9gyzrwuyyyr": ":id",
		"idsfx9p4y3do3bd9gyzrwuyyyr/gn4jzatxwf84dr9d15mdpyz1so": ":id/:id",
		"/path/idsfx9p4y3do3bd9gyzrwuyyyr/thing":                "/path/:id/thing",
	}

	for path, expectedPath := range testCases {
		assert.Equal(t, expectedPath, simplifyPath(path))
	}
}
