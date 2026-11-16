// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"

	"github.com/mattermost/mattermost/server/public/model"
)

const (
	// The webapp debounces incoming posted events with a 100 ms timer that is
	// restarted on every event. The first 5 events of a burst are handled
	// immediately, the following ones are queued and handled as a batch.
	// See debouncePostEvent in webapp/channels/src/actions/websocket_actions.ts.
	postedBurstGap       = 100 * time.Millisecond
	postedImmediateLimit = 5
)

// postedBurst tracks bursts of posted events the same way the webapp's
// debouncePostEvent does. It is not safe for concurrent use.
type postedBurst struct {
	count int
	last  time.Time
}

// batched registers an event received at the given time and returns whether
// the webapp would handle it as part of a batch (which skips the mark-as-read
// call) instead of immediately.
func (b *postedBurst) batched(now time.Time) bool {
	if now.Sub(b.last) > postedBurstGap {
		b.count = 0
	}
	b.last = now
	b.count++
	return b.count > postedImmediateLimit
}

// reactToPostedEvent models what the webapp does when it receives a new post
// through the websocket (completePostReceive and handleNewPostEvent):
//   - for a reply whose root is not loaded in the current channel, it fetches
//     the thread;
//   - it hydrates the author and the mentioned user of the post (profile,
//     status and picture) when they are unknown;
//   - for posts of other users in the channel being viewed, it marks the
//     channel as read on the server, unless the event is handled as part of a
//     batch.
func (c *SimulController) reactToPostedEvent(post *model.Post, batched bool) error {
	u := c.user

	current, err := u.Store().CurrentChannel()
	if errors.Is(err, memstore.ErrChannelNotFound) {
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to get current channel: %w", err)
	}
	if current.Id != post.ChannelId {
		return nil
	}

	collapsedThreads, resp := control.CollapsedThreadsEnabled(u)
	if resp.Err != nil {
		return resp.Err
	}

	if post.RootId != "" {
		if _, err := u.Store().Post(post.RootId); errors.Is(err, memstore.ErrPostNotFound) {
			if _, _, err := u.GetPostThreadWithOpts(post.RootId, "", model.GetPostsOptions{
				CollapsedThreads: collapsedThreads,
				Direction:        "down",
				PerPage:          25,
			}); err != nil {
				return fmt.Errorf("GetPostThreadWithOpts failed: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to get root post from store: %w", err)
		}
	}

	if err := hydratePostedEvent(u, post); err != nil {
		return err
	}

	// System messages are ignored by the webapp, and our own posts are marked
	// as read locally only.
	if batched || post.UserId == u.Store().Id() || strings.HasPrefix(post.Type, model.PostSystemMessagePrefix) {
		return nil
	}

	if _, err := u.ViewChannel(&model.ChannelView{
		ChannelId:                 post.ChannelId,
		CollapsedThreadsSupported: collapsedThreads,
	}); err != nil {
		return fmt.Errorf("ViewChannel failed: %w", err)
	}

	return nil
}

// hydratePostedEvent loads the author and the mentioned user of the post, plus
// the author's status and profile picture, when they are not known yet. This
// mirrors batchFetchStatusesProfilesGroupsFromPosts in the webapp, which only
// requests what is missing from its state (and the browser caches pictures, so
// the picture is only requested for newly seen users).
func hydratePostedEvent(u user.User, post *model.Post) error {
	// The current user is always known (and online).
	if post.UserId == u.Store().Id() {
		return nil
	}

	author, err := u.Store().GetUser(post.UserId)
	if err != nil {
		return fmt.Errorf("failed to get author from store: %w", err)
	}
	if author.Id == "" {
		if _, err := u.GetUsersByIds([]string{post.UserId}, 0); err != nil {
			return fmt.Errorf("GetUsersByIds failed: %w", err)
		}
		if err := getProfileImageForUsers(u, []string{post.UserId}); err != nil {
			return fmt.Errorf("getProfileImageForUsers failed: %w", err)
		}
	}

	status, err := u.Store().Status(post.UserId)
	if err != nil {
		return fmt.Errorf("failed to get status from store: %w", err)
	}
	if status.UserId == "" {
		if err := u.GetUsersStatusesByIds([]string{post.UserId}); err != nil {
			return fmt.Errorf("GetUsersStatusesByIds failed: %w", err)
		}
	}

	if username := extractMentionFromMessage(post.Message); username != "" {
		users, err := u.Store().Users()
		if err != nil {
			return fmt.Errorf("failed to get users from store: %w", err)
		}
		if !slices.ContainsFunc(users, func(usr model.User) bool { return usr.Username == username }) {
			if _, err := u.GetUsersByUsernames([]string{username}); err != nil {
				return fmt.Errorf("GetUsersByUsernames failed: %w", err)
			}
		}
	}

	return nil
}
