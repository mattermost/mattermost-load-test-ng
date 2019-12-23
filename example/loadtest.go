// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package example

import (
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/example/samplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/example/samplestore"
	"github.com/mattermost/mattermost-load-test-ng/example/sampleuser"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-server/v5/mlog"
)

type SampleLoadTester struct {
	controllers []control.UserController
	wg          sync.WaitGroup
	serverURL   string
}

func (lt *SampleLoadTester) initControllers(numUsers int, status chan<- control.UserStatus) {
	for i := 0; i < numUsers; i++ {
		su := sampleuser.New(samplestore.New(), lt.serverURL)
		lt.controllers[i] = samplecontroller.New(i, su, status)
	}
}

func (lt *SampleLoadTester) runControllers() {
	lt.wg.Add(len(lt.controllers))
	for i := 0; i < len(lt.controllers); i++ {
		go func(controller control.UserController) {
			controller.Run()
		}(lt.controllers[i])
	}
}

func (lt *SampleLoadTester) stopControllers() {
	for i := 0; i < len(lt.controllers); i++ {
		lt.controllers[i].Stop()
	}
	lt.wg.Wait()
}

func (lt *SampleLoadTester) handleStatus(status <-chan control.UserStatus) {
	for us := range status {
		if us.Code == control.USER_STATUS_STOPPED || us.Code == control.USER_STATUS_FAILED {
			lt.wg.Done()
		}
		if us.Code == control.USER_STATUS_ERROR {
			mlog.Info(us.Err.Error(), mlog.Int("controller_id", us.ControllerId))
			continue
		} else if us.Code == control.USER_STATUS_FAILED {
			mlog.Error(us.Err.Error())
			continue
		}
		mlog.Info(us.Info, mlog.Int("controller_id", us.ControllerId))
	}
}

func Run() error {
	const numUsers = 4

	lt := SampleLoadTester{
		controllers: make([]control.UserController, numUsers),
		serverURL:   "http://localhost:8065",
	}

	status := make(chan control.UserStatus, numUsers)

	lt.initControllers(numUsers, status)

	go lt.handleStatus(status)

	lt.runControllers()

	<-time.After(60 * time.Second)

	lt.stopControllers()

	return nil
}
