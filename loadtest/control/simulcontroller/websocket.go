// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"fmt"

	"github.com/mattermost/mattermost-server/v5/model"
)

// wsEventHandler listens for WebSocket events to be handled.
// This is used to model user behaviour by responding to certain events with
// the appropriate actions. It differs from userentity.wsEventHandler which is
// instead used to manage the internal user state.
func (c *SimulController) wsEventHandler() {
	for ev := range c.user.Events() {
		switch ev.EventType() {
		case model.WEBSOCKET_EVENT_TYPING:
			userId, ok := ev.Data["user_id"].(string)
			if !ok || len(userId) == 0 {
				c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: invalid data found in event data"))
				break
			}

			user, err := c.user.Store().GetUser(userId)
			if err != nil {
				c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: GetUser failed %w", err))
				break
			}

			// The user was found, we check if we have the status for it.
			if len(user.Id) != 0 {
				status, err := c.user.Store().Status(userId)
				if err != nil {
					c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: Status failed %w", err))
					break
				}

				// If we can't find the user status in the store we fetch it.
				if len(status.UserId) == 0 {
					if err := c.user.GetUsersStatusesByIds([]string{user.Id}); err != nil {
						c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: GetUsersStatusesByIds failed %w", err))
						break
					}
				}

				break
			}

			// We couldn't find the user so we fetch it and its status.
			if _, err := c.user.GetUsersByIds([]string{userId}); err != nil {
				c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: GetUsersByIds failed %w", err))
				break
			}

			if err := c.user.GetUsersStatusesByIds([]string{userId}); err != nil {
				c.status <- c.newErrorStatus(fmt.Errorf("simulcontroller: GetUsersStatusesByIds failed %w", err))
				break
			}
		}
	}
}
