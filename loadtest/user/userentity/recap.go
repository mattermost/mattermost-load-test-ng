// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"context"

	"github.com/mattermost/mattermost/server/public/model"
)

// CreateRecap creates an on-demand recap.
func (ue *UserEntity) CreateRecap(request *model.CreateRecapRequest) (*model.Recap, error) {
	recap, _, err := ue.client.CreateRecap(context.Background(), request)
	return recap, err
}

// GetRecap fetches a recap by ID.
func (ue *UserEntity) GetRecap(recapID string) (*model.Recap, error) {
	recap, _, err := ue.client.GetRecap(context.Background(), recapID)
	return recap, err
}

// GetRecaps fetches a page of the current user's recaps.
func (ue *UserEntity) GetRecaps(page, perPage int) ([]*model.Recap, error) {
	recaps, _, err := ue.client.GetRecaps(context.Background(), page, perPage)
	return recaps, err
}

// MarkRecapsAsViewed marks all unviewed recaps as viewed.
func (ue *UserEntity) MarkRecapsAsViewed() (*model.MarkRecapsViewedResponse, error) {
	response, _, err := ue.client.MarkRecapsAsViewed(context.Background())
	return response, err
}

// MarkRecapAsRead marks a recap as read.
func (ue *UserEntity) MarkRecapAsRead(recapID string) (*model.Recap, error) {
	recap, _, err := ue.client.MarkRecapAsRead(context.Background(), recapID)
	return recap, err
}

// CreateScheduledRecap creates a scheduled recap.
func (ue *UserEntity) CreateScheduledRecap(scheduledRecap *model.ScheduledRecap) (*model.ScheduledRecap, error) {
	created, _, err := ue.client.CreateScheduledRecap(context.Background(), scheduledRecap)
	return created, err
}

// GetScheduledRecaps fetches a page of the current user's scheduled recaps.
func (ue *UserEntity) GetScheduledRecaps(page, perPage int) ([]*model.ScheduledRecap, error) {
	scheduledRecaps, _, err := ue.client.GetScheduledRecaps(context.Background(), page, perPage)
	return scheduledRecaps, err
}
