package userentity

import "github.com/mattermost/mattermost-server/v5/model"

func postsMapToSlice(postsMap map[string]*model.Post) []*model.Post {
	posts := make([]*model.Post, 0, len(postsMap))
	i := 0
	for _, v := range postsMap {
		posts[i] = v
		i++
	}
	return posts
}
