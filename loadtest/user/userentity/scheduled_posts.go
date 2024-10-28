// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"context"
	"github.com/mattermost/mattermost/server/public/model"
)

func (ue *UserEntity) CreateScheduledPost(teamId string, scheduledPost *model.ScheduledPost) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}

	scheduledPost.UserId = user.Id
	createdScheduledPost, _, err := ue.client.CreateScheduledPost(context.Background(), scheduledPost)
	if err != nil {
		return err
	}

	id := scheduledPost.ChannelId

	if createdScheduledPost.RootId != "" {
		id = scheduledPost.RootId
	}

	return ue.store.SetScheduledPost(teamId, id, createdScheduledPost)

}
