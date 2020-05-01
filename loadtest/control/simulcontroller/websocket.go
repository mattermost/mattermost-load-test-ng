// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/mattermost/mattermost-server/v5/model"
)

// wsEventHandler listens for WebSocket events to be handled.
// This is used to model user behaviour by responding to certain events with
// the appropriate actions. It differs from userentity.wsEventHandler which is
// instead used to manage the internal user state.
func (c *SimulController) wsEventHandler() {
	semCount := runtime.NumCPU() * 8
	semaphore := make(chan struct{}, semCount)

	defer func() {
		for i := 0; i < semCount; i++ {
			semaphore <- struct{}{}
		}
		c.wsWaitChan <- struct{}{}
	}()

	for {
		select {
		case ev := <-c.user.Events():
			switch ev.EventType() {
			case model.WEBSOCKET_EVENT_TYPING:
				userId, ok := ev.GetData()["user_id"].(string)
				if !ok || userId == "" {
					c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: invalid data found in event data"))
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
		case <-c.disconnectChan:
			return
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

	if _, err := c.user.GetUsersByIds([]string{id}); err != nil {
		c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: GetUsersByIds failed %w", err))
		return
	}

	if err := c.user.GetUsersStatusesByIds([]string{id}); err != nil {
		c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: GetUsersStatusesByIds failed %w", err))
	}
}
