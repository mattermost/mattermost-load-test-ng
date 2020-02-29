// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
)

// func (c *SimulController) newInfoStatus(info string) control.UserStatus {
// 	return control.UserStatus{
// 		ControllerId: c.id,
// 		User:         c.user,
// 		Code:         control.USER_STATUS_INFO,
// 		Info:         info,
// 		Err:          nil,
// 	}
// }

func (c *SimulController) newErrorStatus(err error) control.UserStatus {
	return control.UserStatus{
		ControllerId: c.id,
		User:         c.user,
		Code:         control.USER_STATUS_ERROR,
		Info:         "",
		Err:          err,
	}
}
