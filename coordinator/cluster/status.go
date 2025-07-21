// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package cluster

type Status struct {
	ActiveUsers int   // Total number of currently active users across the load-test agents cluster.
	NumErrors   int64 // Total number of errors received from the load-test agents cluster.
}
