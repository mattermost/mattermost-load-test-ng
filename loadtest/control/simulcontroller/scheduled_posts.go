package simulcontroller

import (
	"fmt"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost/server/public/model"
	"math/rand"
	"time"
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
		ScheduledAt: loadtest.RandomFutureTime(time.Hour*24*2, time.Hour*24*10),
	}

	if rand.Float64() < probabilityAttachFileToPost {
		if err := control.AttachFilesToDraft(u, &scheduledPost.Draft); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	if err := u.CreateScheduledPost(channel.TeamId, scheduledPost); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("scheduled post created in channel id %v", channel.Id)}
}

func (c *SimulController) updateScheduledPost(u user.User) control.UserActionResponse {
	if ok, resp := control.ScheduledPostsEnabled(u); resp.Err != nil {
		return resp
	} else if !ok {
		return control.UserActionResponse{Info: "scheduled posts not enabled"}
	}

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
	scheduledPost.ScheduledAt = loadtest.RandomFutureTime(time.Hour*24*2, time.Hour*24*10)

	if err := u.UpdateScheduledPost(channel.TeamId, scheduledPost); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("scheduled post updated in channel id %v", channel.Id)}
}

func (c *SimulController) deleteScheduledPost(u user.User) control.UserActionResponse {
	if ok, resp := control.ScheduledPostsEnabled(u); resp.Err != nil {
		return resp
	} else if !ok {
		return control.UserActionResponse{Info: "scheduled posts not enabled"}
	}

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

	return control.UserActionResponse{Info: fmt.Sprintf("scheduled post deleted with id %v", scheduledPost.Id)}
}

func (c *SimulController) sendScheduledPostNow(u user.User) control.UserActionResponse {
	if ok, resp := control.ScheduledPostsEnabled(u); resp.Err != nil {
		return resp
	} else if !ok {
		return control.UserActionResponse{Info: "scheduled posts not enabled"}
	}

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

	return control.UserActionResponse{Info: fmt.Sprintf("scheduled post with id %v manually sent now", scheduledPost.Id)}
}