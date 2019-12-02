package simplecontroller

import (
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
)

func (c *SimpleController) newInfoStatus(info string) control.UserStatus {
	return control.UserStatus{
		c.user,
		control.USER_STATUS_INFO,
		info,
		nil,
	}
}

func (c *SimpleController) newErrorStatus(err error) control.UserStatus {
	return control.UserStatus{
		c.user,
		control.USER_STATUS_ERROR,
		"",
		err,
	}
}
