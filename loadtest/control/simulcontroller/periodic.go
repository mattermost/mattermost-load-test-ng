// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"sync"
	"time"
)

const (
	getUsersStatusByIdsInterval = 60 * time.Second
	submitClientMetricsInterval = 60 * time.Second
)

func (c *SimulController) periodicActions(wg *sync.WaitGroup) {
	getUserStatusTicker := time.NewTicker(getUsersStatusByIdsInterval)
	submitMetricsTicker := time.NewTicker(submitClientMetricsInterval)

	defer func() {
		submitMetricsTicker.Stop()
		getUserStatusTicker.Stop()
		wg.Done()
	}()

	for {
		select {
		case <-getUserStatusTicker.C:
			if resp := c.getUsersStatuses(); resp.Err != nil {
				c.status <- c.newErrorStatus(resp.Err)
			} else {
				c.status <- c.newInfoStatus(resp.Info)
			}
		case <-submitMetricsTicker.C:
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
