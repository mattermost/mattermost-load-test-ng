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

	err = ue.store.SetScheduledPost(teamId, createdScheduledPost)
	if err != nil {
		return err
	}

	return nil
}

func (ue *UserEntity) UpdateScheduledPost(teamId string, scheduledPost *model.ScheduledPost) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}

	scheduledPost.UserId = user.Id
	updatedScheduledPost, _, err := ue.client.UpdateScheduledPost(context.Background(), scheduledPost)
	if err != nil {
		return err
	}

	ue.Store().UpdateScheduledPost(teamId, updatedScheduledPost)

	return nil
}

func (ue *UserEntity) DeleteScheduledPost(scheduledPost *model.ScheduledPost) error {
	_, _, err := ue.client.DeleteScheduledPost(context.Background(), scheduledPost.Id)
	if err != nil {
		return err
	}

	ue.Store().DeleteScheduledPost(scheduledPost)
	return nil
}

func (ue *UserEntity) GetTeamScheduledPosts(teamID string) error {
	scheduledPostsByTeam, _, err := ue.client.GetUserScheduledPosts(context.Background(), teamID, true)
	if err != nil {
		return err
	}

	for _, scheduledPostByTeamId := range scheduledPostsByTeam {
		for _, scheduledPost := range scheduledPostByTeamId {
			err := ue.store.SetScheduledPost(teamID, scheduledPost)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
