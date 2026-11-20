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

	var policyChurnC, attributeChurnC <-chan time.Time
	if c.isScopedConfigLeader() {
		if intervalMs := c.config.ScopedConfig.PolicyChurnIntervalMs; intervalMs > 0 {
			policyChurnTicker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
			defer policyChurnTicker.Stop()
			policyChurnC = policyChurnTicker.C
		}

		if intervalMs := c.config.ScopedConfig.AttributeChurnIntervalMs; intervalMs > 0 {
			attributeChurnTicker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
			defer attributeChurnTicker.Stop()
			attributeChurnC = attributeChurnTicker.C
		}

		if !c.scopedConfig.ready {
			c.sendResponseStatus(c.setupScopedConfig(c.user))
		}
	}

	for {
		select {
		case <-policyChurnC:
			c.sendResponseStatus(c.churnSystemSettingPolicy(c.user))
		case <-attributeChurnC:
			c.sendResponseStatus(c.churnUserAttribute(c.user))
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
