package simulcontroller

import (
	"fmt"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost/server/public/model"
	"math/rand"
)

func (c *SimulController) createScheduledPost(u user.User) control.UserActionResponse {
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
	// Assuming 25% of the scheduled posts are in a thread
	if rand.Float64() < 0.25 {
		post, err := u.Store().RandomPostForChannel(channel.Id)
		if err == nil {
			if post.RootId != "" {
				rootId = post.RootId
			} else {
				rootId = post.Id
			}
		}
	}

	if err := sendTypingEventIfEnabled(u, channel.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	message, err := createMessage(u, channel, false)
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
		ScheduledAt: loadtest.RandomFutureTime(17_28_00_000, 10),
	}

	// 2% of the times post will have files attached.
	if rand.Float64() < 0.02 {
		if err := control.AttachFilesToDraft(u, &scheduledPost.Draft); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	if err := u.CreateScheduledPost(channel.TeamId, scheduledPost); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("scheudled post created in channel id %v", channel.Id)}
}

func (c *SimulController) updateScheduledPost(u user.User) control.UserActionResponse {
	scheduledPost, err := u.Store().GetRandomScheduledPost()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if scheduledPost == nil {
		return control.UserActionResponse{Info: "no scheduled posts found"}
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
	scheduledPost.ScheduledAt = loadtest.RandomFutureTime(17_28_00_000, 10)

	if err := u.UpdateScheduledPost(channel.TeamId, scheduledPost); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("scheudled post updated in channel id %v", channel.Id)}
}

func (c *SimulController) deleteScheduledPost(u user.User) control.UserActionResponse {
	scheduledPost, err := u.Store().GetRandomScheduledPost()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if scheduledPost == nil {
		return control.UserActionResponse{Info: "no scheduled posts found"}
	}

	if err := u.DeleteScheduledPost(scheduledPost.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("scheudled post deleteted with id %v", scheduledPost.Id)}
}

func (c *SimulController) sendScheduledPost(u user.User) control.UserActionResponse {
	scheduledPost, err := u.Store().GetRandomScheduledPost()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if scheduledPost == nil {
		return control.UserActionResponse{Info: "no scheduled posts found"}
	}

	post, err := scheduledPost.ToPost()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if _, err := u.CreatePost(post); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.DeleteScheduledPost(scheduledPost.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("scheudled post sent with id %v", scheduledPost.Id)}
}
