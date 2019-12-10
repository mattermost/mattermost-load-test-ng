// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package loadtest

import (
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/config"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simplecontroller"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/mattermost/mattermost-server/v5/mlog"
)

type LoadTester struct {
	users       []user.User
	controllers []control.UserController
	config      *config.LoadTestConfig
	wg          sync.WaitGroup
}

func (lt *LoadTester) initControllers(numUsers int) {
	config := userentity.Config{
		ServerURL:    lt.config.ConnectionConfiguration.ServerURL,
		WebSocketURL: lt.config.ConnectionConfiguration.WebSocketURL,
	}
	for i := 0; i < numUsers; i++ {
		lt.users[i] = userentity.New(memstore.New(), i, config)
		lt.controllers[i] = &simplecontroller.SimpleController{}
		lt.controllers[i].Init(lt.users[i])
	}
}

func (lt *LoadTester) runControllers(status chan<- control.UserStatus) {
	lt.wg.Add(len(lt.controllers))
	for i := 0; i < len(lt.controllers); i++ {
		go func(controller control.UserController) {
			controller.Run(status)
		}(lt.controllers[i])
	}
}

func (lt *LoadTester) stopControllers() {
	for i := 0; i < len(lt.controllers); i++ {
		lt.controllers[i].Stop()
	}
	lt.wg.Wait()
}

func (lt *LoadTester) handleStatus(status <-chan control.UserStatus) {
	for us := range status {
		if us.Code == control.USER_STATUS_STOPPED || us.Code == control.USER_STATUS_FAILED {
			lt.wg.Done()
		}
		if us.Code == control.USER_STATUS_ERROR {
			mlog.Info(us.Err.Error(), mlog.Int("user_id", us.User.Id()))
			continue
		} else if us.Code == control.USER_STATUS_FAILED {
			mlog.Error(us.Err.Error())
			continue
		}
		mlog.Info(us.Info, mlog.Int("user_id", us.User.Id()))
	}
}

func Run() error {
	mlog.Info("loadtest started")

	const numUsers = 4

	lt := LoadTester{
		users:       make([]user.User, numUsers),
		controllers: make([]control.UserController, numUsers),
	}

	var err error
	if lt.config, err = config.GetConfig(); err != nil {
		return err
	}

	lt.initControllers(numUsers)

	status := make(chan control.UserStatus, numUsers)

	go lt.handleStatus(status)

	start := time.Now()

	lt.runControllers(status)

	time.Sleep(60 * time.Second)

	lt.stopControllers()

	mlog.Info("loadtest done", mlog.String("elapsed", time.Since(start).String()))

	return nil
}
