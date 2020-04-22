// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"fmt"
	"sync"

	"github.com/mattermost/mattermost-server/v5/model"
)

// wsEventHandler listens for WebSocket events to be handled.
// This is used to model user behavior by responding to certain events with
// the appropriate actions. It differs from userentity.wsEventHandler which is
// instead used to manage the internal user state.
func (c *SimulController) wsEventHandler() {
	for ev := range c.user.Events() {
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
					c.wg.Add(1)
					c.semaphore <- struct{}{}
					go func(wg *sync.WaitGroup, id string) {
						defer wg.Done()

						if err := c.user.GetUsersStatusesByIds([]string{id}); err != nil {
							c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: GetUsersStatusesByIds failed %w", err))
						}

						<-c.semaphore
					}(c.wg, user.Id)
				}

				break
			}

			// We couldn't find the user so we fetch it and its status.
			c.wg.Add(1)
			c.semaphore <- struct{}{}
			go func(wg *sync.WaitGroup, id string) {
				defer wg.Done()

				if _, err := c.user.GetUsersByIds([]string{id}); err != nil {
					c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: GetUsersByIds failed %w", err))
					<-c.semaphore
					return
				}

				if err := c.user.GetUsersStatusesByIds([]string{id}); err != nil {
					c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: GetUsersStatusesByIds failed %w", err))
				}

				<-c.semaphore
			}(c.wg, userId)
		}
	}
}
