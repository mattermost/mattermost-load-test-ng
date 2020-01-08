package simplecontroller

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
)

const pkgPath string = "github.com/mattermost/mattermost-load-test-ng/loadtest/control/"

func (c *SimpleController) newInfoStatus(info string) control.UserStatus {
	return control.UserStatus{
		ControllerId: c.id,
		User:         c.user,
		Code:         control.USER_STATUS_INFO,
		Info:         info,
		Err:          nil,
	}
}

func (c *SimpleController) newErrorStatus(err error) control.UserStatus {
	var origin string
	if pc, file, line, ok := runtime.Caller(1); ok {
		if f := runtime.FuncForPC(pc); f != nil {
			if wd, err := os.Getwd(); err == nil {
				origin = fmt.Sprintf("%s %s:%d", strings.TrimPrefix(f.Name(), pkgPath), strings.TrimPrefix(file, wd+"/"), line)
			}
		}
	}
	return control.UserStatus{
		ControllerId: c.id,
		User:         c.user,
		Code:         control.USER_STATUS_ERROR,
		Info:         "",
		Err:          err,
		ErrOrigin:    origin,
	}
}
