package simplecontroller

import (
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
)

func (c *SimpleController) newInfoStatus(info string) control.UserStatus {
	return control.UserStatus{
		User: c.user,
		Code: control.USER_STATUS_INFO,
		Info: info,
		Err:  nil,
	}
}

func (c *SimpleController) newErrorStatus(err error) control.UserStatus {
	return control.UserStatus{
		User: c.user,
		Code: control.USER_STATUS_ERROR,
		Info: "",
		Err:  err,
	}
}
