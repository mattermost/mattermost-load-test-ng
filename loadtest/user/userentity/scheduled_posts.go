package userentity

import (
	"context"
	"fmt"
	"github.com/mattermost/mattermost/server/public/model"
)

func (ue *UserEntity) CreateScheduledPost(teamId string, scheduledPost *model.ScheduledPost) error {
	fmt.Println("CreateScheduledPost: start")

	user, err := ue.getUserFromStore()
	if err != nil {
		fmt.Println("CreateScheduledPost: getUserFromStore error", err)
		return err
	}

	scheduledPost.UserId = user.Id
	createdScheduledPost, _, err := ue.client.CreateScheduledPost(context.Background(), scheduledPost)
	if err != nil {
		fmt.Println("CreateScheduledPost: CreateScheduledPost error", err)
		return err
	}

	id := scheduledPost.ChannelId
	if createdScheduledPost.RootId != "" {
		id = scheduledPost.RootId
	}

	err = ue.store.SetScheduledPost(teamId, id, createdScheduledPost)
	if err != nil {
		fmt.Println("CreateScheduledPost: SetScheduledPost error", err)
		return err
	}

	fmt.Println("CreateScheduledPost: end")
	return nil
}

func (ue *UserEntity) UpdateScheduledPost(teamId string, scheduledPost *model.ScheduledPost) error {
	fmt.Println("UpdateScheduledPost: start")

	user, err := ue.getUserFromStore()
	if err != nil {
		fmt.Println("UpdateScheduledPost: getUserFromStore error", err)
		return err
	}

	scheduledPost.UserId = user.Id
	updatedScheduledPost, _, err := ue.client.UpdateScheduledPost(context.Background(), scheduledPost)
	if err != nil {
		fmt.Println("UpdateScheduledPost: UpdateScheduledPost error", err)
		return err
	}

	id := scheduledPost.ChannelId
	if updatedScheduledPost.RootId != "" {
		id = updatedScheduledPost.RootId
	}

	err = ue.store.SetScheduledPost(teamId, id, updatedScheduledPost)
	if err != nil {
		fmt.Println("UpdateScheduledPost: SetScheduledPost error", err)
		return err
	}

	ue.Store().UpdateScheduledPost(teamId, updatedScheduledPost)

	fmt.Println("UpdateScheduledPost: end")
	return nil
}

func (ue *UserEntity) DeleteScheduledPost(scheduledPostId string) error {
	fmt.Println("DeleteScheduledPost: start")

	_, _, err := ue.client.DeleteScheduledPost(context.Background(), scheduledPostId)
	if err != nil {
		fmt.Println("DeleteScheduledPost: DeleteScheduledPost error", err)
		return err
	}

	fmt.Println("DeleteScheduledPost: end")
	return nil
}

func (ue *UserEntity) GetTeamScheduledPosts(teamID string) error {
	fmt.Println("GetTeamScheduledPosts: start")

	scheduledPostsByTeam, _, err := ue.client.GetUserScheduledPosts(context.Background(), teamID, true)
	if err != nil {
		fmt.Println("GetTeamScheduledPosts: GetUserScheduledPosts error", err)
		return err
	}

	for teamIdInResponse := range scheduledPostsByTeam {
		for _, scheduledPost := range scheduledPostsByTeam[teamIdInResponse] {
			err := ue.store.SetScheduledPost(teamID, scheduledPost.Id, scheduledPost)
			if err != nil {
				fmt.Println("GetTeamScheduledPosts: SetScheduledPost error", err)
				return err
			}
		}
	}

	fmt.Println("GetTeamScheduledPosts: end")
	return nil
}
