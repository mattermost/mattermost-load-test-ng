// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost/server/public/model"

	"github.com/stretchr/testify/require"
)

type scheduledPostMutation struct {
	operation string
	teamID    string
	post      *model.ScheduledPost
}

type scheduledPostRecordingStore struct {
	*memstore.MemStore
	mutations      []scheduledPostMutation
	channelErr     error
	currentTeamErr error
}

func (s *scheduledPostRecordingStore) Channel(channelID string) (*model.Channel, error) {
	if s.channelErr != nil {
		return nil, s.channelErr
	}
	return s.MemStore.Channel(channelID)
}

func (s *scheduledPostRecordingStore) CurrentTeam() (*model.Team, error) {
	if s.currentTeamErr != nil {
		return nil, s.currentTeamErr
	}
	return s.MemStore.CurrentTeam()
}

func (s *scheduledPostRecordingStore) SetScheduledPost(teamID string, scheduledPost *model.ScheduledPost) error {
	s.mutations = append(s.mutations, scheduledPostMutation{
		operation: "set",
		teamID:    teamID,
		post:      scheduledPost,
	})
	return s.MemStore.SetScheduledPost(teamID, scheduledPost)
}

func (s *scheduledPostRecordingStore) UpdateScheduledPost(teamID string, scheduledPost *model.ScheduledPost) {
	s.mutations = append(s.mutations, scheduledPostMutation{
		operation: "update",
		teamID:    teamID,
		post:      scheduledPost,
	})
	s.MemStore.UpdateScheduledPost(teamID, scheduledPost)
}

func (s *scheduledPostRecordingStore) DeleteScheduledPost(scheduledPost *model.ScheduledPost) {
	s.mutations = append(s.mutations, scheduledPostMutation{
		operation: "delete",
		post:      scheduledPost,
	})
	s.MemStore.DeleteScheduledPost(scheduledPost)
}

func newScheduledPostRecordingStore(t *testing.T) *scheduledPostRecordingStore {
	t.Helper()

	store, err := memstore.New(nil)
	require.NoError(t, err)
	return &scheduledPostRecordingStore{MemStore: store}
}

func newScheduledPostEvent(t *testing.T, eventType model.WebsocketEventType, scheduledPost *model.ScheduledPost) *model.WebSocketEvent {
	t.Helper()

	payload, err := json.Marshal(scheduledPost)
	require.NoError(t, err)

	event := model.NewWebSocketEvent(eventType, "", "", "", nil, "")
	event.Add("scheduledPost", string(payload))
	return event
}

func scheduledPost(id, channelID string, scheduledAt int64) *model.ScheduledPost {
	return &model.ScheduledPost{
		Draft: model.Draft{
			ChannelId: channelID,
			Message:   "scheduled message",
		},
		Id:          id,
		ScheduledAt: scheduledAt,
	}
}

func TestScheduledPostWebSocketEvents(t *testing.T) {
	const (
		teamID    = "team1"
		channelID = "channel1"
		postID    = "scheduled-post1"
	)

	testCases := []struct {
		name              string
		eventType         model.WebsocketEventType
		eventPost         *model.ScheduledPost
		setup             func(t *testing.T, store *scheduledPostRecordingStore)
		applyCount        int
		expectedOperation string
		expectedTeamID    string
		expectedEmpty     bool
		expectedError     string
	}{
		{
			name:      "created in team channel",
			eventType: model.WebsocketScheduledPostCreated,
			eventPost: scheduledPost(postID, channelID, 1000),
			setup: func(t *testing.T, store *scheduledPostRecordingStore) {
				require.NoError(t, store.SetChannel(&model.Channel{Id: channelID, TeamId: teamID}))
			},
			expectedOperation: "set",
			expectedTeamID:    teamID,
		},
		{
			name:      "created after HTTP write is idempotent",
			eventType: model.WebsocketScheduledPostCreated,
			eventPost: scheduledPost(postID, channelID, 2000),
			setup: func(t *testing.T, store *scheduledPostRecordingStore) {
				require.NoError(t, store.SetChannel(&model.Channel{Id: channelID, TeamId: teamID}))
				require.NoError(t, store.MemStore.SetScheduledPost(teamID, scheduledPost(postID, channelID, 1000)))
			},
			applyCount:        2,
			expectedOperation: "set",
			expectedTeamID:    teamID,
		},
		{
			name:      "updated existing recurring post advances schedule and records failure",
			eventType: model.WebsocketScheduledPostUpdated,
			eventPost: &model.ScheduledPost{
				Draft:          model.Draft{ChannelId: channelID, Message: "scheduled message"},
				Id:             postID,
				ScheduledAt:    604801000,
				ErrorCode:      "unable_to_send",
				RepeatType:     model.ScheduledPostRepeatTypeWeekly,
				RepeatTimezone: "UTC",
			},
			setup: func(t *testing.T, store *scheduledPostRecordingStore) {
				require.NoError(t, store.SetChannel(&model.Channel{Id: channelID, TeamId: teamID}))
				require.NoError(t, store.MemStore.SetScheduledPost(teamID, scheduledPost(postID, channelID, 1000)))
			},
			expectedOperation: "update",
			expectedTeamID:    teamID,
		},
		{
			name:      "updated absent post is inserted",
			eventType: model.WebsocketScheduledPostUpdated,
			eventPost: scheduledPost(postID, channelID, 2000),
			setup: func(t *testing.T, store *scheduledPostRecordingStore) {
				require.NoError(t, store.SetChannel(&model.Channel{Id: channelID, TeamId: teamID}))
			},
			expectedOperation: "update",
			expectedTeamID:    teamID,
		},
		{
			name:      "deleted existing post",
			eventType: model.WebsocketScheduledPostDeleted,
			eventPost: scheduledPost(postID, channelID, 1000),
			setup: func(t *testing.T, store *scheduledPostRecordingStore) {
				require.NoError(t, store.MemStore.SetScheduledPost(teamID, scheduledPost(postID, channelID, 1000)))
			},
			expectedOperation: "delete",
			expectedEmpty:     true,
		},
		{
			name:      "deleted post with unknown channel",
			eventType: model.WebsocketScheduledPostDeleted,
			eventPost: scheduledPost(postID, "unknown-channel", 1000),
			setup: func(t *testing.T, store *scheduledPostRecordingStore) {
				require.NoError(t, store.MemStore.SetScheduledPost(teamID, scheduledPost(postID, channelID, 1000)))
			},
			expectedOperation: "delete",
			expectedEmpty:     true,
		},
		{
			name:      "created in direct channel uses current team",
			eventType: model.WebsocketScheduledPostCreated,
			eventPost: scheduledPost(postID, "direct-channel", 1000),
			setup: func(t *testing.T, store *scheduledPostRecordingStore) {
				require.NoError(t, store.SetChannel(&model.Channel{Id: "direct-channel", Type: model.ChannelTypeDirect}))
				require.NoError(t, store.SetCurrentTeam(&model.Team{Id: teamID}))
			},
			expectedOperation: "set",
			expectedTeamID:    teamID,
		},
		{
			name:      "created in group channel uses current team",
			eventType: model.WebsocketScheduledPostCreated,
			eventPost: scheduledPost(postID, "group-channel", 1000),
			setup: func(t *testing.T, store *scheduledPostRecordingStore) {
				require.NoError(t, store.SetChannel(&model.Channel{Id: "group-channel", Type: model.ChannelTypeGroup}))
				require.NoError(t, store.SetCurrentTeam(&model.Team{Id: teamID}))
			},
			expectedOperation: "set",
			expectedTeamID:    teamID,
		},
		{
			name:          "unknown channel is ignored",
			eventType:     model.WebsocketScheduledPostCreated,
			eventPost:     scheduledPost(postID, "unknown-channel", 1000),
			expectedEmpty: true,
		},
		{
			name:      "direct channel without current team is ignored",
			eventType: model.WebsocketScheduledPostUpdated,
			eventPost: scheduledPost(postID, "direct-channel", 1000),
			setup: func(t *testing.T, store *scheduledPostRecordingStore) {
				require.NoError(t, store.SetChannel(&model.Channel{Id: "direct-channel", Type: model.ChannelTypeDirect}))
			},
			expectedEmpty: true,
		},
		{
			name:      "channel lookup error is returned",
			eventType: model.WebsocketScheduledPostCreated,
			eventPost: scheduledPost(postID, channelID, 1000),
			setup: func(t *testing.T, store *scheduledPostRecordingStore) {
				store.channelErr = errors.New("channel lookup failed")
			},
			expectedEmpty: true,
			expectedError: "failed to get scheduled post channel from store",
		},
		{
			name:      "current team lookup error is returned",
			eventType: model.WebsocketScheduledPostUpdated,
			eventPost: scheduledPost(postID, "direct-channel", 1000),
			setup: func(t *testing.T, store *scheduledPostRecordingStore) {
				require.NoError(t, store.SetChannel(&model.Channel{Id: "direct-channel", Type: model.ChannelTypeDirect}))
				store.currentTeamErr = errors.New("current team lookup failed")
			},
			expectedEmpty: true,
			expectedError: "failed to get current team from store",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			store := newScheduledPostRecordingStore(t)
			if testCase.setup != nil {
				testCase.setup(t, store)
			}
			entity := &UserEntity{store: store}
			event := newScheduledPostEvent(t, testCase.eventType, testCase.eventPost)

			applyCount := testCase.applyCount
			if applyCount == 0 {
				applyCount = 1
			}
			for i := range applyCount {
				event = event.SetSequence(int64(i))
				err := entity.wsEventHandler(event)
				if testCase.expectedError != "" {
					require.ErrorContains(t, err, testCase.expectedError)
				} else {
					require.NoError(t, err)
				}
			}

			if testCase.expectedOperation == "" {
				require.Empty(t, store.mutations)
			} else {
				require.Len(t, store.mutations, applyCount)
				for _, mutation := range store.mutations {
					require.Equal(t, testCase.expectedOperation, mutation.operation)
					require.Equal(t, testCase.expectedTeamID, mutation.teamID)
					require.Equal(t, testCase.eventPost.Id, mutation.post.Id)
				}
			}

			storedPost, err := store.GetRandomScheduledPost()
			if testCase.expectedEmpty {
				require.ErrorIs(t, err, memstore.ErrScheduledPostStoreEmpty)
				return
			}
			require.NoError(t, err)
			require.Equal(t, testCase.eventPost, storedPost)
		})
	}
}

func TestScheduledPostWebSocketEventMalformedPayload(t *testing.T) {
	testCases := []struct {
		name        string
		addPayload  bool
		payload     any
		expectedErr string
	}{
		{
			name:        "missing scheduled post",
			expectedErr: "scheduled post data is missing",
		},
		{
			name:        "non-string scheduled post",
			addPayload:  true,
			payload:     map[string]any{"id": "scheduled-post1"},
			expectedErr: "type of the scheduled post data should be a string",
		},
		{
			name:       "invalid scheduled post JSON",
			addPayload: true,
			payload:    "{",
		},
		{
			name:        "null scheduled post JSON",
			addPayload:  true,
			payload:     "null",
			expectedErr: "scheduled post data is null",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			store := newScheduledPostRecordingStore(t)
			entity := &UserEntity{store: store}
			event := model.NewWebSocketEvent(model.WebsocketScheduledPostCreated, "", "", "", nil, "")
			if testCase.addPayload {
				event.Add("scheduledPost", testCase.payload)
			}

			err := entity.wsEventHandler(event)
			require.Error(t, err)
			if testCase.expectedErr != "" {
				require.ErrorContains(t, err, testCase.expectedErr)
			}
			require.Empty(t, store.mutations)
			_, err = store.GetRandomScheduledPost()
			require.ErrorIs(t, err, memstore.ErrScheduledPostStoreEmpty)
		})
	}
}
