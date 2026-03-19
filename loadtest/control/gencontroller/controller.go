// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gencontroller

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"slices"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/plugins"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost/server/public/shared/mlog"

	_ "github.com/mattermost/mattermost-plugin-playbooks/loadtest"
)

// GenController is an implementation of a UserController used to generate
// realistic initial data.
type GenController struct {
	id                      int
	user                    user.User
	sysadmin                user.User
	stop                    chan struct{}
	status                  chan<- control.UserStatus
	rate                    float64
	config                  *Config
	channelSelectionWeights []int
	numUsers                int
	plugins                 []plugins.GenController
}

// New creates and initializes a new GenController with given parameters.
// An id is provided to identify the controller, a User is passed as the entity to be controlled and
// a UserStatus channel is passed to communicate errors and information about the user's status.
func New(id int, user user.User, sysadmin user.User, config *Config, status chan<- control.UserStatus, numUsers int) (*GenController, error) {
	if config == nil || user == nil {
		return nil, errors.New("nil params passed")
	}

	if err := config.IsValid(numUsers); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}

	weights := make([]int, len(config.ChannelMembersDistribution))
	for i := range config.ChannelMembersDistribution {
		weights[i] = int(config.ChannelMembersDistribution[i].Probability * 100)
	}

	installedPlugins := []plugins.GenController{}
	plugins.SpawnPluginControllers(plugins.TypeGenController, func(p plugins.Controller) {
		if slices.Contains(config.EnabledPlugins, p.PluginId()) {
			s, ok := p.(plugins.GenController)
			if !ok {
				// Should never happen
				mlog.Error("The provided plugins.Controller cannot be casted to plugins.GenController", mlog.String("plugin_id", p.PluginId()))
				return
			}
			installedPlugins = append(installedPlugins, s)
		}
	})

	sc := &GenController{
		id:                      id,
		user:                    user,
		sysadmin:                sysadmin,
		stop:                    make(chan struct{}),
		status:                  status,
		rate:                    1.0,
		config:                  config,
		channelSelectionWeights: weights,
		numUsers:                numUsers,
		plugins:                 installedPlugins,
	}

	return sc, nil
}

// Run begins performing a set of actions in a loop with a defined wait
// in between the actions. It keeps on doing it until Stop is invoked.
// This is also a blocking function, so it is recommended to invoke it
// inside a goroutine.
func (c *GenController) Run() {
	if c.user == nil {
		c.sendFailStatus("controller was not initialized")
		return
	}

	defer func() {
		if resp := logout(c.user); resp.Err != nil {
			c.status <- c.newErrorStatus(resp.Err)
		} else {
			c.status <- c.newInfoStatus(resp.Info)
		}
		c.sendStopStatus()
	}()

	done := func() bool {
		return st.get(StateTargetTeams) >= c.config.NumTeams &&
			st.get(StateTargetChannelsDM) >= c.config.NumChannelsDM &&
			st.get(StateTargetChannelsGM) >= c.config.NumChannelsGM &&
			st.get(StateTargetChannelsPrivate) >= c.config.NumChannelsPrivate &&
			st.get(StateTargetChannelsPublic) >= c.config.NumChannelsPublic &&
			st.get(StateTargetPosts) >= c.config.NumPosts &&
			st.get(StateTargetReactions) >= c.config.NumReactions &&
			st.get(StateTargetPostReminders) >= c.config.NumPostReminders &&
			st.get(StateTargetSidebarCategories) >= c.config.NumSidebarCategories &&
			st.get(StateTargetFollowedThreads) >= c.config.NumFollowedThreads &&
			st.get(StateTargetUsers) == int64(c.numUsers)
	}

	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user started", Code: control.USER_STATUS_STARTED}

	initActions := []control.UserAction{
		control.SignUp,
		c.login,
		control.GetPreferences,
		c.createTeam,
	}

	for i := 0; i < len(initActions); i++ {
		if done() {
			return
		}

		if !c.runAction(initActions[i]) {
			i--
		}

		idleTime := time.Duration(math.Round(100 * c.rate))

		select {
		case <-c.stop:
			return
		case <-time.After(idleTime * time.Millisecond):
		}
	}

	c.status <- c.newInfoStatus("user init done")

	cpaEnabled, resp := control.CustomProfileAttributesEnabled(c.user)
	if resp.Err != nil {
		c.sendFailStatus("Failed to retreive CustomProfileAttributesEnabled")
		return
	}

	// Create CPA fields
	if cpaEnabled {
		actions := map[string]userAction{
			"createCPAField": {
				run:        c.createCPAField,
				frequency:  int(c.config.NumCPAFields),
				idleTimeMs: 1000,
			},
		}
		c.runActions(actions, func() bool {
			return st.get(StateTargetCPAFields) >= c.config.NumCPAFields
		})

		c.runAction(c.createCPAValues)
	}

	// Wait for all users to be logged in.
	// This also means now users can join all teams.
	for st.get(StateTargetUsers) != int64(c.numUsers) {
		time.Sleep(time.Second)
	}

	requiredActions := []control.UserAction{
		c.joinAllTeams,
		c.getUsers,
	}
	for i := 0; i < len(requiredActions); i++ {
		if !c.runAction(requiredActions[i]) {
			i--
		}

		// Add jitter to spread out potentially heavy calls.
		idleTime := time.Duration(rand.Intn(5)) * time.Second
		select {
		case <-c.stop:
			return
		case <-time.After(idleTime):
		}
	}

	c.status <- c.newInfoStatus("done with required actions")

	// Create all channels
	actions := map[string]userAction{
		"createPublicChannel": {
			run:        c.createPublicChannel,
			frequency:  int(c.config.NumChannelsPublic),
			idleTimeMs: 1000,
		},
		"createPrivateChannel": {
			run:        c.createPrivateChannel,
			frequency:  int(c.config.NumChannelsPrivate),
			idleTimeMs: 1000,
		},
		"createDirectChannel": {
			run:        c.createDirectChannel,
			frequency:  int(c.config.NumChannelsDM),
			idleTimeMs: 1000,
		},
		"createGroupChannel": {
			run:        c.createGroupChannel,
			frequency:  int(c.config.NumChannelsGM),
			idleTimeMs: 1000,
		},
	}
	c.runActions(actions, func() bool {
		return st.get(StateTargetChannelsDM) >= c.config.NumChannelsDM &&
			st.get(StateTargetChannelsGM) >= c.config.NumChannelsGM &&
			st.get(StateTargetChannelsPrivate) >= c.config.NumChannelsPrivate &&
			st.get(StateTargetChannelsPublic) >= c.config.NumChannelsPublic
	})

	// Run the rest of the actions
	actions = map[string]userAction{
		"joinChannel": {
			run:        c.joinChannel,
			frequency:  int(math.Ceil(float64(c.config.NumTotalChannels()))) * 2, // making this proportional to number of channels.
			idleTimeMs: 1000,
		},
		"createPost": {
			run:        c.createPost,
			frequency:  int(math.Ceil(float64(c.config.NumPosts) * (1 - c.config.PercentReplies))),
			idleTimeMs: 1000,
		},
		"createPostReminder": {
			run:        c.createPostReminder,
			frequency:  int(c.config.NumPostReminders),
			idleTimeMs: 1000,
		},
		"createReply": {
			run:        c.createReply,
			frequency:  int(math.Ceil(float64(c.config.NumPosts) * c.config.PercentReplies)),
			idleTimeMs: 1000,
		},
		"addReaction": {
			run:        c.addReaction,
			frequency:  int(c.config.NumReactions),
			idleTimeMs: 1000,
		},
		"createSidebarCategory": {
			run:        c.createSidebarCategory,
			frequency:  int(c.config.NumSidebarCategories),
			idleTimeMs: 1000,
		},
		"followThread": {
			run:        c.followThread,
			frequency:  int(c.config.NumFollowedThreads),
			idleTimeMs: 1000,
		},
		// This action does not generate data, but is needed for all other actions
		// to have data to work with
		"getPosts": {
			run:        c.getPosts,
			frequency:  int(max(100, c.config.NumPosts/1000)),
			idleTimeMs: 1000,
		},
	}

	for _, p := range c.plugins {
		id := p.PluginId()
		for _, a := range p.Actions() {
			actions[id+"."+a.Name] = userAction{
				run:        a.Run,
				frequency:  100 * int(a.Frequency),
				idleTimeMs: 10,
			}
		}
	}

	c.runActions(actions, func() bool {
		pluginsDone := true
		for _, p := range c.plugins {
			if !p.Done() {
				pluginsDone = false
				break
			}
		}
		return pluginsDone &&
			st.get(StateTargetTeams) >= c.config.NumTeams &&
			st.get(StateTargetChannelsDM) >= c.config.NumChannelsDM &&
			st.get(StateTargetChannelsGM) >= c.config.NumChannelsGM &&
			st.get(StateTargetChannelsPrivate) >= c.config.NumChannelsPrivate &&
			st.get(StateTargetChannelsPublic) >= c.config.NumChannelsPublic &&
			st.get(StateTargetPosts) >= c.config.NumPosts &&
			st.get(StateTargetReactions) >= c.config.NumReactions &&
			st.get(StateTargetPostReminders) >= c.config.NumPostReminders &&
			st.get(StateTargetSidebarCategories) >= c.config.NumSidebarCategories &&
			st.get(StateTargetFollowedThreads) >= c.config.NumFollowedThreads
	})
}

// runAction runs the given action and returns whether this was fully executed
// or not. This is used by the caller to figure out whether to retry the action.
// NOTE: this logic relies on a pattern of using resp.Info when the action
// has been completed successfully, otherwise returning early and setting either
// resp.Err or resp.Warn.
func (c *GenController) runAction(action control.UserAction) bool {
	if resp := action(c.user); resp.Err != nil {
		c.status <- c.newErrorStatus(resp.Err)
		return false
	} else if resp.Warn != "" {
		c.status <- c.newWarnStatus(resp.Warn)
		return false
	} else {
		c.status <- c.newInfoStatus(resp.Info)
		return true
	}
}

func (c *GenController) runActions(actions map[string]userAction, done func() bool) {
	for {
		action, err := pickAction(actions)
		if err != nil {
			c.status <- c.newErrorStatus(err)
			return
		}

		if resp := action.run(c.user); resp.Err != nil {
			c.status <- c.newErrorStatus(resp.Err)
		} else if resp.Warn != "" {
			c.status <- c.newWarnStatus(resp.Warn)
		} else {
			c.status <- c.newInfoStatus(resp.Info)
		}

		if done() {
			c.status <- c.newInfoStatus("user done")
			return
		}

		if st.get(StateTargetChannelsPublic) >= c.config.NumChannelsPublic {
			delete(actions, "createPublicChannel")
		}

		if st.get(StateTargetChannelsPrivate) >= c.config.NumChannelsPrivate {
			delete(actions, "createPrivateChannel")
		}

		if st.get(StateTargetChannelsDM) >= c.config.NumChannelsDM {
			delete(actions, "createDirectChannel")
		}

		if st.get(StateTargetChannelsGM) >= c.config.NumChannelsGM {
			delete(actions, "createGroupChannel")
		}

		if st.get(StateTargetCPAFields) >= c.config.NumCPAFields {
			delete(actions, "createCPAField")
		}

		if st.get(StateTargetPosts) >= c.config.NumPosts {
			delete(actions, "createPost")
			delete(actions, "createReply")
		}

		if st.get(StateTargetReactions) >= c.config.NumReactions {
			delete(actions, "addReaction")
		}

		if st.get(StateTargetPostReminders) >= c.config.NumPostReminders {
			delete(actions, "createPostReminder")
		}

		if st.get(StateTargetSidebarCategories) >= c.config.NumSidebarCategories {
			delete(actions, "createSidebarCategory")
		}

		if st.get(StateTargetFollowedThreads) >= c.config.NumFollowedThreads {
			delete(actions, "followThread")
		}

		idleTime := time.Duration(math.Round(float64(action.idleTimeMs) * c.rate))

		select {
		case <-c.stop:
			return
		case <-time.After(idleTime * time.Millisecond):
		}
	}
}

// SetRate sets the relative speed of execution of actions by the user.
func (c *GenController) SetRate(rate float64) error {
	if rate < 0 {
		return errors.New("rate should be a positive value")
	}
	c.rate = rate
	return nil
}

// Stop stops the controller.
func (c *GenController) Stop() {
	close(c.stop)
}

func (c *GenController) sendFailStatus(reason string) {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Code: control.USER_STATUS_FAILED, Err: errors.New(reason)}
}

func (c *GenController) sendStopStatus() {
	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user stopped", Code: control.USER_STATUS_STOPPED}
}

// InjectAction allows a named UserAction to be injected that is run once, at the next
// available opportunity. These actions can be injected via the coordinator via
// CLI or Rest API.
func (c *GenController) InjectAction(actionID string) error {
	mlog.Debug("Cannot inject action for GenController", mlog.String("action", actionID))
	return nil
}

// ensure GenController implements UserController interface
var _ control.UserController = (*GenController)(nil)
