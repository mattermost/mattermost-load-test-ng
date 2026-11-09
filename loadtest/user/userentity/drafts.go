// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"context"
	"fmt"

	"github.com/mattermost/mattermost/server/public/model"
)

// GetDrafts fetches drafts for the given user in a specified team.
func (ue *UserEntity) GetDrafts(teamId string) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}

	drafts, _, err := ue.client.GetDrafts(context.Background(), user.Id, teamId)
	if err != nil {
		return err
	}

	return ue.store.SetDrafts(teamId, drafts)
}

// UpsertDraft creates and stores a new draft made by the user.
func (ue *UserEntity) UpsertDraft(teamId string, draft *model.Draft) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}

	draft.UserId = user.Id
	upsertedDraft, _, err := ue.client.UpsertDraft(context.Background(), draft)
	if err != nil {
		return err
	}

	if upsertedDraft.RootId == "" {
		return ue.store.SetDraft(teamId, upsertedDraft.ChannelId, upsertedDraft)
	}

	return ue.store.SetDraft(teamId, upsertedDraft.RootId, upsertedDraft)
}

func (ue *UserEntity) DeleteDraft(teamId, channelId, rootId string) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}

	if rootId == "" {
		if _, _, err := ue.client.DeleteDraft(context.Background(), user.Id, channelId, rootId); err != nil {
			return err
		}
	} else {
		// Client4.DeleteDraft ignores the root id and would delete the draft of
		// the channel instead, so the thread route is requested directly, as
		// the webapp does.
		resp, err := ue.client.DoAPIDelete(context.Background(), fmt.Sprintf("/users/%s/channels/%s/drafts/%s", user.Id, channelId, rootId))
		if err != nil {
			return err
		}
		resp.Body.Close()
	}

	return ue.store.DeleteDraft(teamId, channelId, rootId)
}
