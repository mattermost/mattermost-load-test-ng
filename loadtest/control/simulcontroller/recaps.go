// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost/server/public/model"
)

const (
	recapListPageSize   = 10
	minutesPerDay       = 24 * 60
	recapTitle          = "Load test recap"
	scheduledRecapTitle = "Load test scheduled recap"
)

func (c *SimulController) createRecap(u user.User) control.UserActionResponse {
	channelIDs, err := pickRecapChannelIDs(u, c.config.RecapsConfiguration.MaxChannelsPerRecap)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("select recap channels: %w", err))}
	}
	if len(channelIDs) == 0 {
		return control.UserActionResponse{Info: "no eligible channels for recap"}
	}

	agentID, err := c.resolveRecapAgentID(u)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("resolve recap agent: %w", err))}
	}

	recap, err := u.CreateRecap(&model.CreateRecapRequest{
		Title:      recapTitle,
		ChannelIds: channelIDs,
		AgentID:    agentID,
	})
	if isRecapsNotImplemented(err) {
		return control.UserActionResponse{Info: "AI Recaps feature is disabled"}
	}
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("create recap: %w", err))}
	}

	return c.pollRecap(u, recap)
}

func (c *SimulController) pollRecap(u user.User, recap *model.Recap) control.UserActionResponse {
	if response, done := recapTerminalResponse(recap); done {
		return response
	}

	pollInterval := time.Duration(c.config.RecapsConfiguration.PollIntervalMs) * time.Millisecond
	timeout := time.NewTimer(time.Duration(c.config.RecapsConfiguration.PollTimeoutMs) * time.Millisecond)
	ticker := time.NewTicker(pollInterval)
	defer timeout.Stop()
	defer ticker.Stop()

	for {
		select {
		case <-c.stopChan:
			return control.UserActionResponse{Info: fmt.Sprintf("recap %s polling canceled", recap.Id)}
		case <-timeout.C:
			return control.UserActionResponse{Info: fmt.Sprintf("recap %s did not reach a terminal status before the polling timeout", recap.Id)}
		case <-ticker.C:
			updated, err := u.GetRecap(recap.Id)
			if isRecapsNotImplemented(err) {
				return control.UserActionResponse{Info: "AI Recaps feature is disabled"}
			}
			if err != nil {
				return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("poll recap %s: %w", recap.Id, err))}
			}
			if response, done := recapTerminalResponse(updated); done {
				return response
			}
		}
	}
}

func recapTerminalResponse(recap *model.Recap) (control.UserActionResponse, bool) {
	switch recap.Status {
	case model.RecapStatusCompleted:
		return control.UserActionResponse{Info: fmt.Sprintf("recap %s completed", recap.Id)}, true
	case model.RecapStatusFailed:
		return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("recap %s failed", recap.Id))}, true
	case model.RecapStatusSkipped:
		info := fmt.Sprintf("recap %s was skipped", recap.Id)
		if recap.SkipReason != "" {
			info += ": " + recap.SkipReason
		}
		return control.UserActionResponse{Info: info}, true
	default:
		return control.UserActionResponse{}, false
	}
}

func (c *SimulController) viewRecaps(u user.User) control.UserActionResponse {
	recaps, err := u.GetRecaps(0, recapListPageSize)
	if isRecapsNotImplemented(err) {
		return control.UserActionResponse{Info: "AI Recaps feature is disabled"}
	}
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("get recaps: %w", err))}
	}

	if _, err := u.MarkRecapsAsViewed(); isRecapsNotImplemented(err) {
		return control.UserActionResponse{Info: "AI Recaps feature is disabled"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("mark recaps as viewed: %w", err))}
	}

	completed := make([]*model.Recap, 0, len(recaps))
	for _, recap := range recaps {
		if recap.Status == model.RecapStatusCompleted {
			completed = append(completed, recap)
		}
	}
	if len(completed) == 0 {
		return control.UserActionResponse{Info: "viewed recaps; no completed recap to read"}
	}

	selected := completed[rand.Intn(len(completed))]
	if _, err := u.GetRecap(selected.Id); isRecapsNotImplemented(err) {
		return control.UserActionResponse{Info: "AI Recaps feature is disabled"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("get recap %s: %w", selected.Id, err))}
	}
	if _, err := u.MarkRecapAsRead(selected.Id); isRecapsNotImplemented(err) {
		return control.UserActionResponse{Info: "AI Recaps feature is disabled"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("mark recap %s as read: %w", selected.Id, err))}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("viewed recaps and read recap %s", selected.Id)}
}

func (c *SimulController) createScheduledRecap(u user.User) control.UserActionResponse {
	config := c.config.RecapsConfiguration
	if config.MaxScheduledRecapsPerUser == 0 {
		return control.UserActionResponse{Info: "scheduled recap creation is disabled by the per-user cap"}
	}

	existing, err := u.GetScheduledRecaps(0, config.MaxScheduledRecapsPerUser)
	if isRecapsNotImplemented(err) {
		return control.UserActionResponse{Info: "AI Recaps feature is disabled"}
	}
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("get scheduled recaps: %w", err))}
	}
	if len(existing) >= config.MaxScheduledRecapsPerUser {
		return control.UserActionResponse{Info: fmt.Sprintf("scheduled recap per-user cap of %d reached", config.MaxScheduledRecapsPerUser)}
	}

	var channelIDs []string
	if config.ScheduledRecapChannelMode == model.ChannelModeSpecific {
		channelIDs, err = pickRecapChannelIDs(u, config.MaxChannelsPerRecap)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("select scheduled recap channels: %w", err))}
		}
		if len(channelIDs) == 0 {
			return control.UserActionResponse{Info: "no eligible channels for scheduled recap"}
		}
	}

	agentID, err := c.resolveRecapAgentID(u)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("resolve recap agent: %w", err))}
	}

	scheduledRecap, err := u.CreateScheduledRecap(&model.ScheduledRecap{
		Title:       scheduledRecapTitle,
		DaysOfWeek:  model.EveryDay,
		TimeOfDay:   scheduledRecapDueTime(config.ScheduledRecapDueTime, rand.Intn(minutesPerDay)),
		Timezone:    "UTC",
		TimePeriod:  model.TimePeriodSinceLastRead,
		ChannelMode: config.ScheduledRecapChannelMode,
		ChannelIds:  model.StringArray(channelIDs),
		AgentId:     agentID,
		IsRecurring: true,
	})
	if isRecapsNotImplemented(err) {
		return control.UserActionResponse{Info: "AI Recaps feature is disabled"}
	}
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(fmt.Errorf("create scheduled recap: %w", err))}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("created scheduled recap %s due at %s UTC", scheduledRecap.Id, scheduledRecap.TimeOfDay)}
}

func (c *SimulController) resolveRecapAgentID(u user.User) (string, error) {
	if c.recapAgentID != "" {
		return c.recapAgentID, nil
	}

	username := strings.TrimSpace(c.config.RecapsConfiguration.AgentUsername)
	ids, err := u.GetUsersByUsernames([]string{username})
	if err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("no user found for agent username %q", username)
	}

	c.recapAgentID = ids[0]
	return c.recapAgentID, nil
}

func pickRecapChannelIDs(u user.User, maxChannels int) ([]string, error) {
	userStore := u.Store()
	userID := userStore.Id()
	teams, err := userStore.Teams()
	if err != nil {
		return nil, err
	}

	var eligible []string
	for _, team := range teams {
		channels, err := userStore.Channels(team.Id)
		if err != nil {
			return nil, err
		}
		for _, channel := range channels {
			if channel.DeleteAt != 0 || (channel.Type != model.ChannelTypeOpen && channel.Type != model.ChannelTypePrivate) {
				continue
			}
			member, err := userStore.ChannelMember(channel.Id, userID)
			if err != nil {
				return nil, err
			}
			if member.UserId != "" {
				eligible = append(eligible, channel.Id)
			}
		}
	}
	if len(eligible) == 0 {
		return nil, nil
	}

	count := min(maxChannels, len(eligible))
	count = 1 + rand.Intn(count)
	return pickIds(eligible, count), nil
}

func scheduledRecapDueTime(configured string, randomMinute int) string {
	if configured != "" {
		return configured
	}
	return fmt.Sprintf("%02d:%02d", randomMinute/60, randomMinute%60)
}

func isRecapsNotImplemented(err error) bool {
	var appErr *model.AppError
	return errors.As(err, &appErr) && appErr.StatusCode == http.StatusNotImplemented
}
