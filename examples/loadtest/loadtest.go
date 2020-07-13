// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package example

import (
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/examples/loadtest/samplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/examples/loadtest/samplestore"
	"github.com/mattermost/mattermost-load-test-ng/examples/loadtest/sampleuser"
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
	for st := range status {
		if st.Code == control.USER_STATUS_STOPPED || st.Code == control.USER_STATUS_FAILED {
			lt.wg.Done()
		}
		if st.Code == control.USER_STATUS_ERROR {
			mlog.Info(st.Err.Error(), mlog.Int("controller_id", st.ControllerId))
			continue
		} else if st.Code == control.USER_STATUS_FAILED {
			mlog.Error(st.Err.Error())
			continue
		}
		mlog.Info(st.Info, mlog.Int("controller_id", st.ControllerId))
	}
}

func (lt *SampleLoadTester) Run(numUsers int) error {
	status := make(chan control.UserStatus, numUsers)
	lt.initControllers(numUsers, status)
	go lt.handleStatus(status)
	lt.runControllers()
	<-time.After(60 * time.Second)
	lt.stopControllers()
	return nil
}

func New(serverURL string) *SampleLoadTester {
	const numUsers = 4
	return &SampleLoadTester{
		controllers: make([]control.UserController, numUsers),
		serverURL:   "http://localhost:8065",
	}
}
