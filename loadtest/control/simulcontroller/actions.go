// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
)

type userAction struct {
	run       control.UserAction
	frequency int
}

func (c *SimulController) connect() {
	errChan := c.user.Connect()
	go func() {
		for err := range errChan {
			c.status <- c.newErrorStatus(err)
		}
	}()
}

func (c *SimulController) reload(full bool) control.UserActionResponse {
	if full {
		err := c.user.Disconnect()
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}

		c.connect()
	}

	return control.Reload(c.user)
}
