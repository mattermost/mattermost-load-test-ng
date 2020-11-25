// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package noopcontroller

// wsEventHandler listens for WebSocket events to be handled.
// This is used to model user behaviour by responding to certain events with
// the appropriate actions. It differs from userentity.wsEventHandler which is
// instead used to manage the internal user state.
func (c *NoopController) wsEventHandler() {
	for range c.user.Events() {
		// do nothing
	}
}
