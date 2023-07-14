// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simplecontroller

import (
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
)

// wsEventHandler listens for WebSocket events to be handled.
// This is used to model user behaviour by responding to certain events with
// the appropriate actions. It differs from userentity.wsEventHandler which is
// instead used to manage the internal user state.
func (c *SimpleController) wsEventHandler(wg *sync.WaitGroup) {
	defer wg.Done()
	for ev := range c.user.Events() {
		switch ev.EventType() {
		case model.WebsocketEventUserUpdated:
			// probably do something interesting ?
		case model.WebsocketEventStatusChange:
			// Send a message if the user has come online.
			data := ev.GetData()
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
