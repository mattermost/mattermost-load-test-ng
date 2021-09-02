// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import "github.com/mattermost/mattermost-server/v6/model"

func postsMapToSlice(postsMap map[string]*model.Post) []*model.Post {
	posts := make([]*model.Post, len(postsMap))
	i := 0
	for _, v := range postsMap {
		posts[i] = v
		i++
	}
	return posts
}

func postListToSlice(list *model.PostList) []*model.Post {
	posts := make([]*model.Post, len(list.Order))
	for i, id := range list.Order {
		posts[i] = list.Posts[id]
	}
	return posts
}
