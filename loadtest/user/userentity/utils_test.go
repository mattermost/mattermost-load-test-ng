// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/stretchr/testify/assert"
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
