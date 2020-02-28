// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package performance

// Status is a structure containing information on the performance status
// of the target instance.
type Status struct {
	// A boolean value indicating if performance degradation occurred.
	Alert bool
}
