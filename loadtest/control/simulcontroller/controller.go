// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost-server/server/v8/platform/shared/mlog"
)

func getActionList(c *SimulController) []userAction {
	return []userAction{
		{
			name:      "SwitchChannel",
			run:       switchChannel,
			frequency: 4,
		},
		{
			name:      "SwitchTeam",
			run:       c.switchTeam,
			frequency: 3,
		},
		{
			name:      "ScrollChannel",
			run:       c.scrollChannel,
			frequency: 2,
		},
		{
			name:      "OpenDirectOrGroupChannel",
			run:       openDirectOrGroupChannel,
			frequency: 2,
		},
		{
			name:      "UnreadCheck",
			run:       unreadCheck,
			frequency: 1.5,
		},
		{
			name:      "CreatePost",
			run:       c.createPost,
			frequency: 1.225,
		},
		{
			name:      "JoinChannel",
			run:       c.joinChannel,
			frequency: 0.8,
		},
		{
			name:      "SearchChannels",
			run:       c.searchChannels,
			frequency: 0.5,
		},
		{
			name:      "AddReaction",
			run:       c.addReaction,
			frequency: 0.5,
		},
		{
			name:      "FullReload",
			run:       c.fullReload,
			frequency: 0.2,
		},
		{
			name:      "CreateDirectChannel",
			run:       c.createDirectChannel,
			frequency: 0.25,
		},
		{
			name:      "LogoutLogin",
			run:       c.logoutLogin,
			frequency: 0.1,
		},
		{
			name:      "SearchUsers",
			run:       searchUsers,
			frequency: 0.1,
		},
		{
			name:      "SearchPosts",
			run:       searchPosts,
			frequency: 0.1,
		},
		{
			name:      "CreatePostReminder",
			run:       c.createPostReminder,
			frequency: 0.002,
		},
		{
			name:      "EditPost",
			run:       editPost,
			frequency: 0.1,
		},
		{
			name:      "DeletePost",
			run:       deletePost,
			frequency: 0.06,
		},
		{
			name:      "UpdateCustomStatus",
			run:       c.updateCustomStatus,
			frequency: 0.05,
		},
		{
			name:      "RemoveCustomStatus",
			run:       c.removeCustomStatus,
			frequency: 0.05,
		},
		{
			name:      "CreateSidebarCategory",
			run:       c.createSidebarCategory,
			frequency: 0.06,
		},
		{
			name:      "UpdateSidebarCategory",
			run:       c.updateSidebarCategory,
			frequency: 0.06,
		},
		{
			name:      "SearchGroupChannels",
			run:       searchGroupChannels,
			frequency: 0.1,
		},
		{
			name:      "CreateGroupChannel",
			run:       c.createGroupChannel,
			frequency: 0.05,
		},
		{
			name:      "CreatePrivateChannel",
			run:       createPrivateChannel,
			frequency: 0.022,
		},
		{
			name:      "CreatePublicChannel",
			run:       control.CreatePublicChannel,
			frequency: 0.011,
		},
		{
			name:      "ViewGlobalThreads",
			run:       c.viewGlobalThreads,
			frequency: 5.4,
		},
		{
			name:      "FollowThread",
			run:       c.followThread,
			frequency: 0.041,
		},
		{
			name:      "UnfollowThread",
			run:       c.unfollowThread,
			frequency: 0.055,
		},
		{
			name:      "ViewThread",
			run:       c.viewThread,
			frequency: 4.8,
		},
		{
			name:      "MarkAllThreadsInTeamAsRead",
			run:       c.markAllThreadsInTeamAsRead,
			frequency: 0.013,
		},
		{
			name:      "UpdateThreadRead",
			run:       c.updateThreadRead,
			frequency: 1.17,
		},
		{
			name:      "GetInsights",
			run:       c.getInsights,
			frequency: 0.011,
		},
		{
			name:      "CreateAclPost",
			run:       control.CreateAckPost,
			frequency: 0.225,
		},
		{
			name:      "AckToPost",
			run:       control.AckToPost,
			frequency: 0.22,
		},
		{
			name:      "CreatePersistentNotificationPost",
			run:       control.CreatePersistentNotificationPost,
			frequency: 0.05,
		},
	}
}

func getActionMap(actionList []userAction) map[string]userAction {
	actionMap := make(map[string]userAction)
	for _, action := range actionList {
		actionMap[action.name] = action
	}
	return actionMap
}

// SimulController is a simulative implementation of a UserController.
type SimulController struct {
	id                 int
	user               user.User
	status             chan<- control.UserStatus
	rate               float64
	config             *Config
	actionList         []userAction
	actionMap          map[string]userAction
	injectedActionChan chan userAction
	stopChan           chan struct{}   // this channel coordinates the stop sequence of the controller
	stoppedChan        chan struct{}   // blocks until controller cleans up everything
	disconnectChan     chan struct{}   // notifies disconnection to the ws and periodic goroutines
	connectedFlag      int32           // indicates that the controller is connected
	wg                 *sync.WaitGroup // to keep the track of every goroutine created by the controller
	serverVersion      string          // stores the current server version
	featureFlags       featureFlags    // stores the server's feature flags
}

type featureFlags struct {
	GraphQLEnabled bool
}

// New creates and initializes a new SimulController with given parameters.
// An id is provided to identify the controller, a User is passed as the entity to be controlled and
// a UserStatus channel is passed to communicate errors and information about the user's status.
func New(id int, user user.User, config *Config, status chan<- control.UserStatus) (*SimulController, error) {
	if config == nil || user == nil {
		return nil, errors.New("nil params passed")
	}

	if err := defaults.Validate(config); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}

	controller := &SimulController{
		id:                 id,
		user:               user,
		status:             status,
		rate:               1.0,
		config:             config,
		injectedActionChan: make(chan userAction, 10),
		disconnectChan:     make(chan struct{}),
		stopChan:           make(chan struct{}),
		stoppedChan:        make(chan struct{}),
		wg:                 &sync.WaitGroup{},
	}

	controller.actionList = getActionList(controller)
	controller.actionMap = getActionMap(controller.actionList)

	return controller, nil
}

// Run begins performing a set of user actions in a loop.
// It keeps on doing it until Stop() is invoked.
// This is also a blocking function, so it is recommended to invoke it
// inside a goroutine.
func (c *SimulController) Run() {
	if c.user == nil {
		c.sendFailStatus("controller was not initialized")
		return
	}

	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user started", Code: control.USER_STATUS_STARTED}

	defer func() {
		if err := c.disconnect(); err != nil {
			c.status <- c.newErrorStatus(control.NewUserError(err))
		}
		c.user.ClearUserData()
		c.sendStopStatus()
		close(c.stoppedChan)
	}()

	c.serverVersion, _ = c.user.Store().ServerVersion()

	initActions := []userAction{
		{
			run: c.loginOrSignUp,
		},
		{
			run: c.initialJoinTeam,
		},
	}

	for i := 0; i < len(initActions); i++ {
		select {
		case <-c.stopChan:
			return
		case <-time.After(control.PickIdleTimeMs(c.config.MinIdleTimeMs, c.config.AvgIdleTimeMs, 1.0)):
		}

		if resp := initActions[i].run(c.user); resp.Err != nil {
			c.status <- c.newErrorStatus(resp.Err)
			i--
		} else {
			c.status <- c.newInfoStatus(resp.Info)
		}
	}

	// Populate the server feature flags struct
	clientCfg := c.user.Store().ClientConfig()
	if len(clientCfg) == 0 {
		c.sendFailStatus("the login init action should have populated the user config, but it is empty")
		return
	}
	c.featureFlags = featureFlags{
		GraphQLEnabled: c.user.Store().ClientConfig()["FeatureFlagGraphQL"] == "true",
	}

	var action *userAction
	var err error

	for {
		select {
		case ia := <-c.injectedActionChan: // injected actions are run first
			action = &ia
		default:
			action, err = pickAction(c.actionList)
			if err != nil {
				panic(fmt.Sprintf("simulcontroller: failed to pick action %s", err.Error()))
			}
		}

		c.runAction(action)

		select {
		case <-c.stopChan:
			return
		case <-time.After(control.PickIdleTimeMs(c.config.MinIdleTimeMs, c.config.AvgIdleTimeMs, c.rate)):
		case ia := <-c.injectedActionChan: // run injected actions immediately
			c.runAction(&ia)
		}
	}
}

func (c *SimulController) runAction(action *userAction) {
	if action == nil {
		return
	}

	if action.minServerVersion != "" {
		supported, err := control.IsVersionSupported(action.minServerVersion, c.serverVersion)
		if err != nil {
			c.status <- c.newErrorStatus(err)
		} else if !supported {
			return
		}
	}

	if resp := action.run(c.user); resp.Err != nil {
		c.status <- c.newErrorStatus(resp.Err)
	} else {
		c.status <- c.newInfoStatus(resp.Info)
	}
}

// SetRate sets the relative speed of execution of actions by the user.
func (c *SimulController) SetRate(rate float64) error {
	if rate < 0 {
		return errors.New("rate should be a positive value")
	}
	c.rate = rate
	return nil
}

// Stop stops the controller.
func (c *SimulController) Stop() {
	close(c.stopChan)
	<-c.stoppedChan
	// re-initialize for the next use
	c.injectedActionChan = make(chan userAction, 10)
	c.stopChan = make(chan struct{})
	c.stoppedChan = make(chan struct{})
}

func (c *SimulController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Code: control.USER_STATUS_FAILED, Err: errors.New(reason)}
}

func (c *SimulController) sendStopStatus() {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}

// InjectAction allows a named UserAction to be injected that is run once, at the next
// available opportunity. These actions can be injected via the coordinator via
// CLI or Rest API.
func (c *SimulController) InjectAction(actionID string) error {
	var action userAction
	var ok bool

	// include some actions that are not normally supported by SimulController
	switch actionID {
	case "Reload":
		action = userAction{
			name: "Reload",
			run:  func(_ user.User) control.UserActionResponse { return c.reload(false) },
		}
	default:
		action, ok = c.actionMap[actionID]
		if !ok {
			mlog.Debug("Could not inject action for SimulController", mlog.String("action", actionID))
			return nil
		}
	}

	select {
	case c.injectedActionChan <- action:
		return nil
	default:
		return fmt.Errorf("action %s could not be queued: %w", actionID, control.ErrInjectActionQueueFull)
	}
}

// ensure SimulController implements UserController interface
var _ control.UserController = (*SimulController)(nil)
