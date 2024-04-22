// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

func getActionList(c *SimulController) []userAction {
	actions := []userAction{
		{
			name:      "SwitchChannel",
			run:       switchChannel,
			frequency: 6.5219,
		},
		{
			name:      "SwitchTeam",
			run:       c.switchTeam,
			frequency: 0.0001,
		},
		{
			name:      "ScrollChannel",
			run:       c.scrollChannel,
			frequency: 1.9873,
		},
		{
			name:      "OpenDirectOrGroupChannel",
			run:       openDirectOrGroupChannel,
			frequency: 0.9843,
		},
		{
			name:      "UnreadCheck",
			run:       unreadCheck,
			frequency: 1,
		},
		{
			name:      "CreatePost",
			run:       c.createPost,
			frequency: 1,
		},
		{
			name:      "JoinChannel",
			run:       c.joinChannel,
			frequency: 0.0049,
		},
		{
			name:      "SearchChannels",
			run:       c.searchChannels,
			frequency: 0.0150,
		},
		{
			name:      "AddReaction",
			run:       c.addReaction,
			frequency: 0.1306,
		},
		{
			name:      "FullReload",
			run:       c.fullReload,
			frequency: 0.0008,
		},
		{
			name:      "CreateDirectChannel",
			run:       c.createDirectChannel,
			frequency: 0.0055,
		},
		{
			name:      "LogoutLogin",
			run:       c.logoutLogin,
			frequency: 0.0006,
		},
		{
			name:      "SearchUsers",
			run:       searchUsers,
			frequency: 0.0320,
		},
		{
			name:      "SearchPosts",
			run:       searchPosts,
			frequency: 0.0218,
		},
		{
			name:      "CreatePostReminder",
			run:       c.createPostReminder,
			frequency: 0.0005,
		},
		{
			name:      "EditPost",
			run:       editPost,
			frequency: 0.0400,
		},
		{
			name:      "DeletePost",
			run:       deletePost,
			frequency: 0.0049,
		},
		{
			name:      "UpdateCustomStatus",
			run:       c.updateCustomStatus,
			frequency: 0.0028,
		},
		{
			name:      "RemoveCustomStatus",
			run:       c.removeCustomStatus,
			frequency: 0.0026,
		},
		{
			name:      "CreateSidebarCategory",
			run:       c.createSidebarCategory,
			frequency: 0.0001,
		},
		{
			name:      "UpdateSidebarCategory",
			run:       c.updateSidebarCategory,
			frequency: 0.0040,
		},
		{
			name:      "SearchGroupChannels",
			run:       searchGroupChannels,
			frequency: 0.0204,
		},
		{
			name:      "CreateGroupChannel",
			run:       c.createGroupChannel,
			frequency: 0.0029,
		},
		{
			name:      "CreatePrivateChannel",
			run:       createPrivateChannel,
			frequency: 0.0002,
		},
		{
			name:      "CreatePublicChannel",
			run:       createPublicChannel,
			frequency: 0.0001,
		},
		{
			name:      "ViewGlobalThreads",
			run:       c.viewGlobalThreads,
			frequency: 0.6023,
		},
		{
			name:      "FollowThread",
			run:       c.followThread,
			frequency: 0.0005,
		},
		{
			name:      "UnfollowThread",
			run:       c.unfollowThread,
			frequency: 0.0050,
		},
		{
			name:      "ViewThread",
			run:       c.viewThread,
			frequency: 0.2841,
		},
		{
			name:      "MarkAllThreadsInTeamAsRead",
			run:       c.markAllThreadsInTeamAsRead,
			frequency: 0.0001,
		},
		{
			name:      "UpdateThreadRead",
			run:       c.updateThreadRead,
			frequency: 0.3236,
		},
		{
			name:      "CreateAclPost",
			run:       control.CreateAckPost,
			frequency: 0.0001,
		},
		{
			name:      "AckToPost",
			run:       control.AckToPost,
			frequency: 0.0001,
		},
		{
			name:      "CreatePersistentNotificationPost",
			run:       control.CreatePersistentNotificationPost,
			frequency: 0.0001,
		},
		{
			name:      "ClickUserProfile",
			run:       c.openUserProfile,
			frequency: 0.03,
		},
		{
			name:      "ClickPermalink",
			run:       c.openPermalink,
			frequency: 0.3,
		},
		{
			name:      "ReconnectWebSocket",
			run:       c.reconnectWebSocket,
			frequency: 0.144,
		},
		{
			name:      "GenerateUserReport",
			run:       c.generateUserReport,
			frequency: 0.0001,
		},
	}

	return actions
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

	// Init controller's server version
	serverVersionString, err := c.user.Store().ServerVersion()
	if err != nil {
		c.sendFailStatus("server version could not be retrieved")
		return
	}
	serverVersion, err := control.ParseServerVersion(serverVersionString)
	if err != nil {
		c.sendFailStatus("server version could not be parsed")
		return
	}
	c.serverVersion = serverVersion

	// Early check that the server version is greater or equal than the initialVersion
	if !c.isVersionSupported(control.MinSupportedVersion) {
		c.sendFailStatus(fmt.Sprintf(
			"server version %q is lower than the minimum supported version %q",
			serverVersion.String(),
			control.MinSupportedVersion.String(),
		))
		return
	}

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
		} else if resp.Info != "" {
			c.status <- c.newInfoStatus(resp.Info)
		}
	}

	// Make sure client config has been set.
	if len(c.user.Store().ClientConfig()) == 0 {
		c.sendFailStatus("the login init action should have populated the user config, but it is empty")
		return
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

	if resp := action.run(c.user); resp.Err != nil {
		c.status <- c.newErrorStatus(resp.Err)
	} else if resp.Info != "" {
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

func (c *SimulController) isVersionSupported(version semver.Version) bool {
	return version.LTE(c.serverVersion)
}

// ensure SimulController implements UserController interface
var _ control.UserController = (*SimulController)(nil)
