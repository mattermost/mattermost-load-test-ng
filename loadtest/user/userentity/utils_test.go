// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPostsMapToSlice(t *testing.T) {
	postsMap := make(map[string]*model.Post)

	id1 := model.NewId()
	id2 := model.NewId()
	postsMap[id1] = &model.Post{Id: id1}
	postsMap[id2] = &model.Post{Id: id2}

	assert.Len(t, postsMapToSlice(postsMap), 2)

	postsMap = map[string]*model.Post{}
	assert.Len(t, postsMapToSlice(postsMap), 0)
}

func TestStripIDs(t *testing.T) {
	for _, tc := range []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "empty",
		},
		{
			name:     "no match",
			input:    "/api/v4/users/me",
			expected: "/api/v4/users/me",
		},
		{
			name:     "match",
			input:    "/api/v4/users/w9w9rsucuig4ukzf1tzjzfhy5h/teams/s6kdxh9owfyii8zkq7sianthmh/channels",
			expected: "/api/v4/users/$ID/teams/$ID/channels",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, stripIDs(tc.input))
		})
	}
}
