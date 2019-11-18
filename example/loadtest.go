// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package example

import (
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test/example/samplecontroller"
	"github.com/mattermost/mattermost-load-test/example/samplestore"
	"github.com/mattermost/mattermost-load-test/example/sampleuser"
	"github.com/mattermost/mattermost-load-test/loadtest/control"
	"github.com/mattermost/mattermost-load-test/loadtest/user"
	"github.com/mattermost/mattermost-server/mlog"
)

type SampleLoadTester struct {
	users       []user.User
	controllers []control.UserController
	wg          sync.WaitGroup
	serverURL   string
}

func (lt *SampleLoadTester) initControllers(numUsers int) {
	for i := 0; i < numUsers; i++ {
		lt.users[i] = sampleuser.New(samplestore.New(), i, lt.serverURL)
		lt.controllers[i] = &samplecontroller.SampleController{}
		lt.controllers[i].Init(lt.users[i])
	}
}

func (lt *SampleLoadTester) runControllers(status chan<- user.UserStatus) {
	lt.wg.Add(len(lt.controllers))
	for i := 0; i < len(lt.controllers); i++ {
		go func(controller control.UserController) {
			controller.Run(status)
		}(lt.controllers[i])
	}
}

func (lt *SampleLoadTester) stopControllers() {
	for i := 0; i < len(lt.controllers); i++ {
		lt.controllers[i].Stop()
	}
	lt.wg.Wait()
}

func (lt *SampleLoadTester) handleStatus(status <-chan user.UserStatus) {
	for us := range status {
		if us.Code == user.STATUS_STOPPED || us.Code == user.STATUS_FAILED {
			lt.wg.Done()
		}
		if us.Code == user.STATUS_ERROR {
			mlog.Info(us.Err.Error(), mlog.Int("user_id", us.User.Id()))
			continue
		} else if us.Code == user.STATUS_FAILED {
			mlog.Error(us.Err.Error())
			continue
		}
		mlog.Info(us.Info, mlog.Int("user_id", us.User.Id()))
	}
}

func Run() error {
	const numUsers = 4

	lt := SampleLoadTester{
		users:       make([]user.User, numUsers),
		controllers: make([]control.UserController, numUsers),
		serverURL:   "http://localhost:8065",
	}

	status := make(chan user.UserStatus, numUsers)

	lt.initControllers(numUsers)

	go lt.handleStatus(status)

	lt.runControllers(status)

	<-time.After(60 * time.Second)

	lt.stopControllers()

	return nil
}
