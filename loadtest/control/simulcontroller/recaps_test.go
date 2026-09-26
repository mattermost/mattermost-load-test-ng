// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

type recapTestUser struct {
	user.User
	getRecap            func(string) (*model.Recap, error)
	getUsersByUsernames func([]string) ([]string, error)
	userStore           store.UserStore
}

func (u *recapTestUser) GetRecap(recapID string) (*model.Recap, error) {
	return u.getRecap(recapID)
}

func (u *recapTestUser) GetUsersByUsernames(usernames []string) ([]string, error) {
	return u.getUsersByUsernames(usernames)
}

func (u *recapTestUser) Store() store.UserStore {
	return u.userStore
}

func defaultSimulControllerConfig(t *testing.T) Config {
	t.Helper()

	var config Config
	require.NoError(t, defaults.Set(&config))
	return config
}

func TestRecapsConfigurationDefaults(t *testing.T) {
	config := defaultSimulControllerConfig(t)
	require.NoError(t, defaults.Validate(&config))

	require.Equal(t, RecapsConfiguration{
		Enabled:                   false,
		AgentUsername:             "ai",
		MaxChannelsPerRecap:       3,
		PollIntervalMs:            5000,
		PollTimeoutMs:             120000,
		MaxScheduledRecapsPerUser: 1,
		ScheduledRecapChannelMode: "specific",
		ScheduledRecapDueTime:     "",
	}, config.RecapsConfiguration)
}

func TestRecapsConfigurationValidation(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*RecapsConfiguration)
		wantError bool
	}{
		{
			name: "valid configured due time",
			mutate: func(config *RecapsConfiguration) {
				config.Enabled = true
				config.ScheduledRecapDueTime = "09:00"
			},
		},
		{
			name: "disabled zero value",
			mutate: func(config *RecapsConfiguration) {
				*config = RecapsConfiguration{}
			},
		},
		{
			name: "empty agent username",
			mutate: func(config *RecapsConfiguration) {
				config.Enabled = true
				config.AgentUsername = " "
			},
			wantError: true,
		},
		{
			name: "zero max channels",
			mutate: func(config *RecapsConfiguration) {
				config.Enabled = true
				config.MaxChannelsPerRecap = 0
			},
			wantError: true,
		},
		{
			name: "invalid channel mode",
			mutate: func(config *RecapsConfiguration) {
				config.Enabled = true
				config.ScheduledRecapChannelMode = "invalid"
			},
			wantError: true,
		},
		{
			name: "invalid due time",
			mutate: func(config *RecapsConfiguration) {
				config.Enabled = true
				config.ScheduledRecapDueTime = "9:00"
			},
			wantError: true,
		},
		{
			name: "zero poll interval",
			mutate: func(config *RecapsConfiguration) {
				config.Enabled = true
				config.PollIntervalMs = 0
			},
			wantError: true,
		},
		{
			name: "zero poll timeout",
			mutate: func(config *RecapsConfiguration) {
				config.Enabled = true
				config.PollTimeoutMs = 0
			},
			wantError: true,
		},
		{
			name: "poll interval exceeds timeout",
			mutate: func(config *RecapsConfiguration) {
				config.Enabled = true
				config.PollIntervalMs = config.PollTimeoutMs + 1
			},
			wantError: true,
		},
		{
			name: "negative scheduled recap cap",
			mutate: func(config *RecapsConfiguration) {
				config.Enabled = true
				config.MaxScheduledRecapsPerUser = -1
			},
			wantError: true,
		},
		{
			name: "scheduled recap cap exceeds page size",
			mutate: func(config *RecapsConfiguration) {
				config.Enabled = true
				config.MaxScheduledRecapsPerUser = maxScheduledRecapsPageSize + 1
			},
			wantError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := defaultSimulControllerConfig(t)
			test.mutate(&config.RecapsConfiguration)
			if test.wantError {
				require.Error(t, defaults.Validate(&config))
			} else {
				require.NoError(t, defaults.Validate(&config))
			}
		})
	}
}

func TestRecapActionsGatedByConfiguration(t *testing.T) {
	tests := []struct {
		name    string
		enabled bool
		present bool
	}{
		{name: "disabled", enabled: false, present: false},
		{name: "enabled", enabled: true, present: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := defaultSimulControllerConfig(t)
			config.RecapsConfiguration.Enabled = test.enabled
			controller := &SimulController{config: &config}

			actionMap := getActionMap(getActionList(controller))
			for _, actionName := range []string{"CreateRecap", "ViewRecaps", "CreateScheduledRecap"} {
				action, found := actionMap[actionName]
				require.Equal(t, test.present, found, actionName)
				if found {
					require.Equal(t, "11.2.0", action.minServerVersion.String())
				}
			}
		})
	}
}

func TestScheduledRecapDueTime(t *testing.T) {
	tests := []struct {
		name         string
		configured   string
		randomMinute int
		expected     string
	}{
		{name: "configured", configured: "08:30", randomMinute: 0, expected: "08:30"},
		{name: "start of day", randomMinute: 0, expected: "00:00"},
		{name: "end of day", randomMinute: minutesPerDay - 1, expected: "23:59"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.expected, scheduledRecapDueTime(test.configured, test.randomMinute))
		})
	}
}

func TestPickRecapChannelIDs(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*testing.T, *memstore.MemStore, string) []string
	}{
		{
			name: "empty store",
			setup: func(_ *testing.T, _ *memstore.MemStore, _ string) []string {
				return nil
			},
		},
		{
			name: "only active member public or private channels",
			setup: func(t *testing.T, userStore *memstore.MemStore, userID string) []string {
				teamID := model.NewId()
				eligibleID := model.NewId()
				nonMemberID := model.NewId()
				deletedID := model.NewId()
				directID := model.NewId()
				require.NoError(t, userStore.SetTeams([]*model.Team{{Id: teamID}}))
				require.NoError(t, userStore.SetChannels([]*model.Channel{
					{Id: eligibleID, TeamId: teamID, Type: model.ChannelTypePrivate},
					{Id: nonMemberID, TeamId: teamID, Type: model.ChannelTypeOpen},
					{Id: deletedID, TeamId: teamID, Type: model.ChannelTypeOpen, DeleteAt: 1},
					{Id: directID, TeamId: teamID, Type: model.ChannelTypeDirect},
				}))
				require.NoError(t, userStore.SetChannelMembers(model.ChannelMembers{
					{ChannelId: eligibleID, UserId: userID},
					{ChannelId: deletedID, UserId: userID},
					{ChannelId: directID, UserId: userID},
				}))
				return []string{eligibleID}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			userStore, err := memstore.New(nil)
			require.NoError(t, err)
			userID := model.NewId()
			require.NoError(t, userStore.SetUser(&model.User{Id: userID}))
			expected := test.setup(t, userStore, userID)

			channelIDs, err := pickRecapChannelIDs(&recapTestUser{userStore: userStore}, 3)

			require.NoError(t, err)
			require.ElementsMatch(t, expected, channelIDs)
		})
	}
}

func TestPollRecapTimeoutIsNotFatal(t *testing.T) {
	config := defaultSimulControllerConfig(t)
	config.RecapsConfiguration.PollIntervalMs = 1
	config.RecapsConfiguration.PollTimeoutMs = 10
	controller := &SimulController{
		config:   &config,
		stopChan: make(chan struct{}),
	}
	testUser := &recapTestUser{
		getRecap: func(recapID string) (*model.Recap, error) {
			return &model.Recap{Id: recapID, Status: model.RecapStatusProcessing}, nil
		},
	}

	response := controller.pollRecap(testUser, &model.Recap{Id: "recap-id", Status: model.RecapStatusPending})

	require.NoError(t, response.Err)
	require.Contains(t, response.Info, "polling timeout")
}

func TestRecapTerminalResponse(t *testing.T) {
	tests := []struct {
		name      string
		recap     *model.Recap
		wantDone  bool
		wantError bool
		wantInfo  string
	}{
		{
			name:     "completed",
			recap:    &model.Recap{Id: "completed-id", Status: model.RecapStatusCompleted},
			wantDone: true,
			wantInfo: "completed",
		},
		{
			name:      "failed",
			recap:     &model.Recap{Id: "failed-id", Status: model.RecapStatusFailed},
			wantDone:  true,
			wantError: true,
		},
		{
			name:     "skipped with reason",
			recap:    &model.Recap{Id: "skipped-id", Status: model.RecapStatusSkipped, SkipReason: model.SkipReasonDailyLimit},
			wantDone: true,
			wantInfo: model.SkipReasonDailyLimit,
		},
		{
			name:  "processing",
			recap: &model.Recap{Id: "processing-id", Status: model.RecapStatusProcessing},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response, done := recapTerminalResponse(test.recap)
			require.Equal(t, test.wantDone, done)
			if test.wantError {
				require.Error(t, response.Err)
			} else {
				require.NoError(t, response.Err)
			}
			if test.wantInfo != "" {
				require.Contains(t, response.Info, test.wantInfo)
			}
		})
	}
}

func TestResolveRecapAgentIDCachesResult(t *testing.T) {
	config := defaultSimulControllerConfig(t)
	controller := &SimulController{config: &config}
	callCount := 0
	testUser := &recapTestUser{
		getUsersByUsernames: func(usernames []string) ([]string, error) {
			callCount++
			require.Equal(t, []string{"ai"}, usernames)
			return []string{"agent-id"}, nil
		},
	}

	first, err := controller.resolveRecapAgentID(testUser)
	require.NoError(t, err)
	second, err := controller.resolveRecapAgentID(testUser)
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, "agent-id", first)
	require.Equal(t, 1, callCount)
}
