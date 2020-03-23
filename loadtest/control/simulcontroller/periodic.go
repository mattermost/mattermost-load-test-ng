// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"time"
)

const (
	getUsersStatusByIdsInterval = 60 * time.Second
)

func (c *SimulController) periodicActions() {
	for {
		select {
		case <-time.After(getUsersStatusByIdsInterval):
			if resp := c.getUsersStatuses(); resp.Err != nil {
				c.status <- c.newErrorStatus(resp.Err)
			} else {
				c.status <- c.newInfoStatus(resp.Info)
			}
		// We can add more periodic actions here.
		case <-c.stop:
			return
		}
	}
}
