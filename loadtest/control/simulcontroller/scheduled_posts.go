// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost/server/public/model"
)

const (
	probabilityToggleScheduledPostRecurrence = 0.10
	scheduledPostFutureTimeDeltaStart        = 48 * time.Hour
	scheduledPostFutureTimeMaxUntil          = 240 * time.Hour
)

func nextScheduledPostBatchBoundary(now time.Time, interval, minLead time.Duration) int64 {
	eligible := now.UTC().Add(minLead)
	boundary := eligible.Truncate(interval)
	if boundary.Before(eligible) {
		boundary = boundary.Add(interval)
	}

	return boundary.UnixMilli()
}

func (c *SimulController) scheduledPostTime(now time.Time) int64 {
	if rand.Float64() < c.config.PercentBatchAlignedScheduledPosts {
		interval := time.Duration(c.config.ScheduledPostBatchIntervalMinutes) * time.Minute
		minLead := time.Duration(c.config.ScheduledPostBatchMinLeadMinutes) * time.Minute
		return nextScheduledPostBatchBoundary(now, interval, minLead)
	}

	return loadtest.RandomFutureTime(scheduledPostFutureTimeDeltaStart, scheduledPostFutureTimeMaxUntil)
}

func (c *SimulController) createScheduledPost(u user.User) control.UserActionResponse {
	return c.createScheduledPostWithRecurrence(u, false)
}

func (c *SimulController) createRecurringScheduledPost(u user.User) control.UserActionResponse {
	return c.createScheduledPostWithRecurrence(u, true)
}

func (c *SimulController) createScheduledPostWithRecurrence(u user.User, recurring bool) control.UserActionResponse {
	if ok, resp := control.ScheduledPostsEnabled(u); resp.Err != nil {
		return resp
	} else if !ok {
		return control.UserActionResponse{Info: "scheduled posts not enabled"}
	}

	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	var rootId = ""
	if rand.Float64() < 0.25 {
		post, err := u.Store().RandomPostForChannel(channel.Id)
		if errors.Is(err, memstore.ErrPostNotFound) {
			return control.UserActionResponse{Info: fmt.Sprintf("no posts found in channel %v", channel.Id)}
		} else if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}

		if post.RootId != "" {
			rootId = post.RootId
		} else {
			rootId = post.Id
		}
	}

	if err := sendTypingEventIfEnabled(u, channel.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	message, err := createMessage(u, channel, rootId != "")
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	scheduledPost := &model.ScheduledPost{
		Draft: model.Draft{
			Message:   message,
			ChannelId: channel.Id,
			RootId:    rootId,
			CreateAt:  model.GetMillis(),
		},
		ScheduledAt: c.scheduledPostTime(time.Now()),
	}

	if rand.Float64() < probabilityAttachFileToPost {
		if err := control.AttachFilesToDraft(u, &scheduledPost.Draft); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	if recurring {
		scheduledPost.RepeatType = model.ScheduledPostRepeatTypeWeekly
		scheduledPost.RepeatTimezone = control.RecurringScheduledPostTimezone(u)
	}

	if err := u.CreateScheduledPost(channel.TeamId, scheduledPost); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if recurring {
		return control.UserActionResponse{Info: fmt.Sprintf("recurring scheduled post created in channel with id %s", channel.Id)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("scheduled post created in channel with id %s", channel.Id)}
}

func (c *SimulController) updateScheduledPost(u user.User) control.UserActionResponse {
	if ok, resp := control.ScheduledPostsEnabled(u); resp.Err != nil {
		return resp
	} else if !ok {
		return control.UserActionResponse{Info: "scheduled posts not enabled"}
	}

	scheduledPost, err := u.Store().GetRandomScheduledPost()
	if errors.Is(err, memstore.ErrScheduledPostStoreEmpty) {
		return control.UserActionResponse{Info: "no scheduled posts found"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	message, err := createMessage(u, channel, false)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	scheduledPost.Message = message
	scheduledPost.ScheduledAt = loadtest.RandomFutureTime(scheduledPostFutureTimeDeltaStart, scheduledPostFutureTimeMaxUntil)

	recurrenceUpdate := ""
	if c.serverVersion.GTE(control.RecurringScheduledPostsMinVersion) &&
		rand.Float64() < probabilityToggleScheduledPostRecurrence {
		if scheduledPost.IsRecurring() {
			scheduledPost.RepeatType = model.ScheduledPostRepeatTypeNone
			scheduledPost.RepeatTimezone = ""
			recurrenceUpdate = "disabled"
		} else {
			scheduledPost.RepeatType = model.ScheduledPostRepeatTypeWeekly
			scheduledPost.RepeatTimezone = control.RecurringScheduledPostTimezone(u)
			recurrenceUpdate = "enabled"
		}
	}

	if err := u.UpdateScheduledPost(channel.TeamId, scheduledPost); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if recurrenceUpdate != "" {
		return control.UserActionResponse{Info: fmt.Sprintf("scheduled post updated with recurrence %s in channel with id %s", recurrenceUpdate, channel.Id)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("scheduled post updated in channel with id %s", channel.Id)}
}

func (c *SimulController) deleteScheduledPost(u user.User) control.UserActionResponse {
	if ok, resp := control.ScheduledPostsEnabled(u); resp.Err != nil {
		return resp
	} else if !ok {
		return control.UserActionResponse{Info: "scheduled posts not enabled"}
	}

	scheduledPost, err := u.Store().GetRandomScheduledPost()
	if errors.Is(err, memstore.ErrScheduledPostStoreEmpty) {
		return control.UserActionResponse{Info: "no scheduled posts found"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.DeleteScheduledPost(scheduledPost); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("scheduled post with id %s deleted", scheduledPost.Id)}
}

func (c *SimulController) sendScheduledPostNow(u user.User) control.UserActionResponse {
	if ok, resp := control.ScheduledPostsEnabled(u); resp.Err != nil {
		return resp
	} else if !ok {
		return control.UserActionResponse{Info: "scheduled posts not enabled"}
	}

	scheduledPost, err := u.Store().GetRandomScheduledPost()
	if errors.Is(err, memstore.ErrScheduledPostStoreEmpty) {
		return control.UserActionResponse{Info: "no scheduled posts found"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	post, err := scheduledPost.ToPost()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if _, err := u.CreatePost(post); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.DeleteScheduledPost(scheduledPost); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("scheduled post with id %s manually sent now", scheduledPost.Id)}
}
