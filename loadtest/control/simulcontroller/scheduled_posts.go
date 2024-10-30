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
	fmt.Println("createScheduledPost: start")

	if ok, resp := control.ScheduledPostsEnabled(u); resp.Err != nil {
		fmt.Println("createScheduledPost: ScheduledPostsEnabled error", resp.Err)
		return resp
	} else if !ok {
		fmt.Println("createScheduledPost: ScheduledPosts not enabled")
		return control.UserActionResponse{Info: "scheduled posts not enabled"}
	}

	fmt.Println("createScheduledPost: ScheduledPost is enabled")

	channel, err := u.Store().CurrentChannel()
	if err != nil {
		fmt.Println("createScheduledPost: CurrentChannel error", err)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	fmt.Println("createScheduledPost: got the channel", channel.Id)
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
		fmt.Println("createScheduledPost: sendTypingEventIfEnabled error", err)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	message, err := createMessage(u, channel, false)
	if err != nil {
		fmt.Println("createScheduledPost: createMessage error", err)
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

	if rand.Float64() < 0.02 {
		if err := control.AttachFilesToDraft(u, &scheduledPost.Draft); err != nil {
			fmt.Println("createScheduledPost: AttachFilesToDraft error", err)
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	if err := u.CreateScheduledPost(channel.TeamId, scheduledPost); err != nil {
		fmt.Println("createScheduledPost: CreateScheduledPost error", err)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	fmt.Println("createScheduledPost: end")
	return control.UserActionResponse{Info: fmt.Sprintf("scheduled post created in channel id %v", channel.Id)}
}

func (c *SimulController) updateScheduledPost(u user.User) control.UserActionResponse {
	fmt.Println("updateScheduledPost: start")

	scheduledPost, err := u.Store().GetRandomScheduledPost()
	if err != nil {
		fmt.Println("updateScheduledPost: GetRandomScheduledPost error", err)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if scheduledPost == nil {
		fmt.Println("updateScheduledPost: no scheduled posts found")
		return control.UserActionResponse{Info: "no scheduled posts found"}
	}

	channel, err := u.Store().CurrentChannel()
	if err != nil {
		fmt.Println("updateScheduledPost: CurrentChannel error", err)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	message, err := createMessage(u, channel, false)
	if err != nil {
		fmt.Println("updateScheduledPost: createMessage error", err)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	scheduledPost.Message = message
	scheduledPost.ScheduledAt = loadtest.RandomFutureTime(17_28_00_000, 10)

	if err := u.UpdateScheduledPost(channel.TeamId, scheduledPost); err != nil {
		fmt.Println("updateScheduledPost: UpdateScheduledPost error", err)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	fmt.Println("updateScheduledPost: end")
	return control.UserActionResponse{Info: fmt.Sprintf("scheduled post updated in channel id %v", channel.Id)}
}

func (c *SimulController) deleteScheduledPost(u user.User) control.UserActionResponse {
	fmt.Println("deleteScheduledPost: start")

	scheduledPost, err := u.Store().GetRandomScheduledPost()
	if err != nil {
		fmt.Println("deleteScheduledPost: GetRandomScheduledPost error", err)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if scheduledPost == nil {
		fmt.Println("deleteScheduledPost: no scheduled posts found")
		return control.UserActionResponse{Info: "no scheduled posts found"}
	}

	if err := u.DeleteScheduledPost(scheduledPost.Id); err != nil {
		fmt.Println("deleteScheduledPost: DeleteScheduledPost error", err)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	fmt.Println("deleteScheduledPost: end")
	return control.UserActionResponse{Info: fmt.Sprintf("scheduled post deleted with id %v", scheduledPost.Id)}
}

func (c *SimulController) sendScheduledPost(u user.User) control.UserActionResponse {
	fmt.Println("sendScheduledPost: start")

	scheduledPost, err := u.Store().GetRandomScheduledPost()
	if err != nil {
		fmt.Println("sendScheduledPost: GetRandomScheduledPost error", err)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if scheduledPost == nil {
		fmt.Println("sendScheduledPost: no scheduled posts found")
		return control.UserActionResponse{Info: "no scheduled posts found"}
	}

	post, err := scheduledPost.ToPost()
	if err != nil {
		fmt.Println("sendScheduledPost: ToPost error", err)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if _, err := u.CreatePost(post); err != nil {
		fmt.Println("sendScheduledPost: CreatePost error", err)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.DeleteScheduledPost(scheduledPost.Id); err != nil {
		fmt.Println("sendScheduledPost: DeleteScheduledPost error", err)
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	fmt.Println("sendScheduledPost: end")
	return control.UserActionResponse{Info: fmt.Sprintf("scheduled post sent with id %v", scheduledPost.Id)}
}
