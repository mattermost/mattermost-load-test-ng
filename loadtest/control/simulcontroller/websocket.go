// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
)

// wsEventHandler listens for WebSocket events to be handled.
// This is used to model user behaviour by responding to certain events with
// the appropriate actions. It differs from userentity.wsEventHandler which is
// instead used to manage the internal user state.
func (c *SimulController) wsEventHandler(wg *sync.WaitGroup) {
	semCount := runtime.NumCPU() * 8
	semaphore := make(chan struct{}, semCount)

	defer func() {
		for i := 0; i < semCount; i++ {
			semaphore <- struct{}{}
		}
		wg.Done()
	}()

	for ev := range c.user.Events() {
		switch ev.EventType() {
		case model.WebsocketEventPosted:
			post, err := getPostFromEvent(ev)
			if err != nil {
				c.status <- c.newErrorStatus(fmt.Errorf("failed to get post from event: %w", err))
				break
			}

			if ack, ok := ev.GetData()["should_ack"]; ok && ack.(bool) {
				if err := c.user.PostedAck(post.Id, "success", "", ""); err != nil {
					c.status <- c.newErrorStatus(err)
					break
				}
			}

			cm, _ := c.user.Store().ChannelMember(post.ChannelId, c.user.Store().Id())
			if cm.UserId != "" {
				break
			}

			c.status <- c.newInfoStatus(fmt.Sprintf("channel member for post's channel missing from store, fetching: %q", post.ChannelId))

			if err := c.user.GetChannelMember(post.ChannelId, c.user.Store().Id()); err != nil {
				c.status <- c.newErrorStatus(fmt.Errorf("GetChannelMember failed: %w", err))
				break
			}

			// If we were to follow webapp literally we'd have to check against user's
			// preferences and possibly reload direct/group channels (and users) if
			// necessary. However this only happens if the preferences for these
			// channels are set not to show them, something we don't support yet as we default to
			// showing them.
		case model.WebsocketEventTyping:
			userId, ok := ev.GetData()["user_id"].(string)
			if !ok || userId == "" {
				c.status <- c.newErrorStatus(errors.New("simulcontroller: invalid data found in event data"))
				break
			}
			user, err := c.user.Store().GetUser(userId)
			if err != nil {
				c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: GetUser failed %w", err))
				break
			}

			// The user was found, we check if we have the status for it.
			if user.Id != "" {
				status, err := c.user.Store().Status(userId)
				if err != nil {
					c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: Status failed %w", err))
					break
				}

				// If we can't find the user status in the store we fetch it.
				if status.UserId == "" {
					select {
					case semaphore <- struct{}{}:
						go fetchStatus(c, semaphore, user.Id)
					default:
						c.status <- c.newErrorStatus(errors.New("simulcontroller: dropping call"))
					}
				}

				break
			}

			// We couldn't find the user so we fetch it and its status.
			select {
			case semaphore <- struct{}{}:
				go fetchUserAndStatus(c, semaphore, userId)
			default:
				c.status <- c.newErrorStatus(errors.New("simulcontroller: dropping call"))
			}
		}
	}
}

func fetchStatus(c *SimulController, sem chan struct{}, id string) {
	defer func() { <-sem }()

	if err := c.user.GetUsersStatusesByIds([]string{id}); err != nil {
		c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: GetUsersStatusesByIds failed %w", err))
	}
}

func fetchUserAndStatus(c *SimulController, sem chan struct{}, id string) {
	defer func() { <-sem }()

	if _, err := c.user.GetUsersByIds([]string{id}, 0); err != nil {
		c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: GetUsersByIds failed %w", err))
		return
	}

	if err := c.user.GetUsersStatusesByIds([]string{id}); err != nil {
		c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: GetUsersStatusesByIds failed %w", err))
	}
}
