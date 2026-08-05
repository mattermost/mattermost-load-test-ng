// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"context"

	"github.com/mattermost/mattermost/server/public/model"
)

func (ue *UserEntity) CreateRecap(request *model.CreateRecapRequest) (*model.Recap, error) {
	recap, _, err := ue.client.CreateRecap(context.Background(), request)
	return recap, err
}

func (ue *UserEntity) GetRecap(recapID string) (*model.Recap, error) {
	recap, _, err := ue.client.GetRecap(context.Background(), recapID)
	return recap, err
}

func (ue *UserEntity) GetRecaps(page, perPage int) ([]*model.Recap, error) {
	recaps, _, err := ue.client.GetRecaps(context.Background(), page, perPage)
	return recaps, err
}

func (ue *UserEntity) GetRecapLimitStatus() (*model.RecapLimitStatus, error) {
	status, _, err := ue.client.GetRecapLimitStatus(context.Background())
	return status, err
}

func (ue *UserEntity) MarkRecapsAsViewed() (*model.MarkRecapsViewedResponse, error) {
	response, _, err := ue.client.MarkRecapsAsViewed(context.Background())
	return response, err
}

func (ue *UserEntity) MarkRecapAsRead(recapID string) (*model.Recap, error) {
	recap, _, err := ue.client.MarkRecapAsRead(context.Background(), recapID)
	return recap, err
}

func (ue *UserEntity) DeleteRecap(recapID string) error {
	_, err := ue.client.DeleteRecap(context.Background(), recapID)
	return err
}

func (ue *UserEntity) CreateScheduledRecap(scheduledRecap *model.ScheduledRecap) (*model.ScheduledRecap, error) {
	created, _, err := ue.client.CreateScheduledRecap(context.Background(), scheduledRecap)
	return created, err
}

func (ue *UserEntity) GetScheduledRecaps(page, perPage int) ([]*model.ScheduledRecap, error) {
	scheduledRecaps, _, err := ue.client.GetScheduledRecaps(context.Background(), page, perPage)
	return scheduledRecaps, err
}

func (ue *UserEntity) UpdateScheduledRecap(scheduledRecap *model.ScheduledRecap) (*model.ScheduledRecap, error) {
	updated, _, err := ue.client.UpdateScheduledRecap(context.Background(), scheduledRecap)
	return updated, err
}

func (ue *UserEntity) DeleteScheduledRecap(scheduledRecapID string) error {
	_, err := ue.client.DeleteScheduledRecap(context.Background(), scheduledRecapID)
	return err
}

func (ue *UserEntity) PauseScheduledRecap(scheduledRecapID string) (*model.ScheduledRecap, error) {
	paused, _, err := ue.client.PauseScheduledRecap(context.Background(), scheduledRecapID)
	return paused, err
}

func (ue *UserEntity) ResumeScheduledRecap(scheduledRecapID string) (*model.ScheduledRecap, error) {
	resumed, _, err := ue.client.ResumeScheduledRecap(context.Background(), scheduledRecapID)
	return resumed, err
}
