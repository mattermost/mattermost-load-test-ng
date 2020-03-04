// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simplecontroller

import (
	"github.com/mattermost/mattermost-server/v5/model"
)

// wsEventHandler listens for WebSocket events to be handled.
// This is used to model user behaviour by responding to certain events with
// the appropriate actions. It differs from userentity.wsEventHandler which is
// instead used to manage the internal user state.
func (c *SimpleController) wsEventHandler() {
	for ev := range c.user.Events() {
		switch ev.EventType() {
		case model.WEBSOCKET_EVENT_USER_UPDATED:
			// probably do something interesting ?
		case model.WEBSOCKET_EVENT_STATUS_CHANGE:
			// Send a message if the user has come online.
			data := ev.Data // TODO: upgrade the server dependency and move to GetData call
			status, ok := data["status"].(string)
			if !ok || status != "online" {
				continue
			}
			userID, ok := data["user_id"].(string)
			if !ok {
				continue
			}
			c.status <- c.sendDirectMessage(userID)
		default:
			// add other handlers as necessary.
		}
	}
}
