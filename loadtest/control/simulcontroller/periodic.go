// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"sync"
	"time"
)

const (
	getUsersStatusByIdsInterval = 60 * time.Second
	submitClientMetircsInterval = 60 * time.Second
)

func (c *SimulController) periodicActions(wg *sync.WaitGroup) {
	st := time.NewTicker(getUsersStatusByIdsInterval)
	mt := time.NewTicker(submitClientMetircsInterval)

	defer func() {
		mt.Stop()
		st.Stop()
		wg.Done()
	}()

	for {
		select {
		case <-st.C:
			if resp := c.getUsersStatuses(); resp.Err != nil {
				c.status <- c.newErrorStatus(resp.Err)
			} else {
				c.status <- c.newInfoStatus(resp.Info)
			}
		case <-mt.C:
			if resp := submitPerformanceReport(c.user); resp.Err != nil {
				c.status <- c.newErrorStatus(resp.Err)
			} else {
				c.status <- c.newInfoStatus(resp.Info)
			}
		case <-c.disconnectChan:
			return
		}
	}
}
