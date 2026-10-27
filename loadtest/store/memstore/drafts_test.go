// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func TestRandomDraftForTeamAndDeleteDraft(t *testing.T) {
	s := newStore(t)

	user := &model.User{Id: model.NewId(), Username: "user"}
	require.NoError(t, s.SetUser(user))

	teamId := model.NewId()
	channelDraft := &model.Draft{UserId: user.Id, ChannelId: model.NewId()}
	threadDraft := &model.Draft{UserId: user.Id, ChannelId: model.NewId(), RootId: model.NewId()}
	otherUserDraft := &model.Draft{UserId: model.NewId(), ChannelId: model.NewId()}

	_, err := s.RandomDraftForTeam(teamId)
	require.ErrorIs(t, err, ErrDraftNotFound)

	require.NoError(t, s.SetDrafts(teamId, []*model.Draft{channelDraft, threadDraft, otherUserDraft}))

	// Only the drafts of the current user are returned, whole, so that callers
	// know the channel and the root they belong to.
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		d, err := s.RandomDraftForTeam(teamId)
		require.NoError(t, err)
		require.Equal(t, user.Id, d.UserId)
		seen[d.ChannelId+d.RootId] = true
	}
	require.Equal(t, map[string]bool{
		channelDraft.ChannelId + channelDraft.RootId: true,
		threadDraft.ChannelId + threadDraft.RootId:   true,
	}, seen)

	require.NoError(t, s.DeleteDraft(teamId, channelDraft.ChannelId, ""))
	for i := 0; i < 20; i++ {
		d, err := s.RandomDraftForTeam(teamId)
		require.NoError(t, err)
		require.Equal(t, threadDraft.RootId, d.RootId)
	}

	require.NoError(t, s.DeleteDraft(teamId, threadDraft.ChannelId, threadDraft.RootId))
	_, err = s.RandomDraftForTeam(teamId)
	require.ErrorIs(t, err, ErrDraftNotFound)

	// Deleting something that is not there is not an error.
	require.NoError(t, s.DeleteDraft(model.NewId(), model.NewId(), ""))
}
