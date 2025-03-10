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
	<-c.disconnectChan
	wg.Done()
}
