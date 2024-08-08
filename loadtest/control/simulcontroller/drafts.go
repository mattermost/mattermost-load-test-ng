// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"math/rand"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost/server/public/model"
)

func (c *SimulController) getDrafts(u user.User) control.UserActionResponse {
	if ok, resp := control.DraftsEnabled(u); resp.Err != nil {
		return resp
	} else if !ok {
		return control.UserActionResponse{Info: "drafts not enabled"}
	}

	userId := u.Store().Id()

	team, err := u.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("current team should be set"))}
	}

	err = u.GetDrafts(team.Id)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("viewed drafts for user id %v in team id %v", userId, team.Id)}
}

func (c *SimulController) upsertDraft(u user.User) control.UserActionResponse {
	if ok, resp := control.DraftsEnabled(u); resp.Err != nil {
		return resp
	} else if !ok {
		return control.UserActionResponse{Info: "drafts not enabled"}
	}

	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	var rootId = ""
	// 87% of the time draft will be a thread reply
	// source: https://hub.mattermost.com/private-core/pl/qqr4t6n3wpbdxnouhy9qrabewh
	if rand.Float64() < 0.87 {
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

	message, err := createMessage(u, channel, false)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	draft := &model.Draft{
		Message:   message,
		ChannelId: channel.Id,
		RootId:    rootId,
		CreateAt:  model.GetMillis(),
	}

	// 2% of the times post will have files attached.
	if rand.Float64() < 0.02 {
		if err := control.AttachFilesToDraft(u, draft); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	err = u.UpsertDraft(channel.TeamId, draft)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("draft created in channel id %v", channel.Id)}
}

func (c *SimulController) deleteDraft(u user.User) control.UserActionResponse {
	if ok, resp := control.DraftsEnabled(u); resp.Err != nil {
		return resp
	} else if !ok {
		return control.UserActionResponse{Info: "drafts not enabled"}
	}

	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	team, err := u.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("current team should be set"))}
	}

	draftId, err := u.Store().RandomDraftForTeam(team.Id)
	if errors.Is(err, memstore.ErrDraftNotFound) {
		return control.UserActionResponse{Info: fmt.Sprintf("no drafts found in team %v", team.Id)}
	}

	err = u.DeleteDraft(channel.Id, draftId)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("draft deleted in channel id %v", channel.Id)}
}
