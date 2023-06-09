// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package gencontroller

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost-server/server/v8/platform/shared/mlog"
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
}

// New creates and initializes a new GenController with given parameters.
// An id is provided to identify the controller, a User is passed as the entity to be controlled and
// a UserStatus channel is passed to communicate errors and information about the user's status.
func New(id int, user user.User, sysadmin user.User, config *Config, status chan<- control.UserStatus) (*GenController, error) {
	if config == nil || user == nil {
		return nil, errors.New("nil params passed")
	}

	if err := config.IsValid(); err != nil {
		return nil, fmt.Errorf("could not validate configuration: %w", err)
	}

	weights := make([]int, len(config.ChannelMembersDistribution))
	for i := range config.ChannelMembersDistribution {
		weights[i] = int(config.ChannelMembersDistribution[i].Probability * 100)
	}

	sc := &GenController{
		id:                      id,
		user:                    user,
		sysadmin:                sysadmin,
		stop:                    make(chan struct{}),
		status:                  status,
		rate:                    1.0,
		config:                  config,
		channelSelectionWeights: weights,
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
			st.get(StateTargetChannels) >= c.config.NumChannels &&
			st.get(StateTargetPosts) >= c.config.NumPosts &&
			st.get(StateTargetReactions) >= c.config.NumReactions &&
			st.get(StateTargetPostReminders) >= c.config.NumPostReminders &&
			st.get(StateTargetSidebarCategories) >= c.config.NumSidebarCategories &&
			st.get(StateTargetFollowedThreads) >= c.config.NumFollowedThreads
	}

	c.status <- control.UserStatus{ControllerId: c.id, User: c.user, Info: "user started", Code: control.USER_STATUS_STARTED}

	initActions := []control.UserAction{
		control.SignUp,
		control.Login,
		control.GetPreferences,
		c.createTeam,
		c.joinTeam,
	}

	for i := 0; i < len(initActions); i++ {
		if done() {
			c.status <- c.newInfoStatus("user done")
			return
		}

		if resp := initActions[i](c.user); resp.Err != nil {
			c.status <- c.newErrorStatus(resp.Err)
			i--
		} else {
			c.status <- c.newInfoStatus(resp.Info)
		}

		idleTime := time.Duration(math.Round(100 * c.rate))

		select {
		case <-c.stop:
			return
		case <-time.After(idleTime * time.Millisecond):
		}
	}

	actions := map[string]userAction{
		"createPublicChannel": {
			run:        c.createPublicChannel,
			frequency:  int(math.Ceil(float64(c.config.NumChannels) * c.config.PercentPublicChannels)),
			idleTimeMs: 1000,
		},
		"createPrivateChannel": {
			run:        c.createPrivateChannel,
			frequency:  int(math.Ceil(float64(c.config.NumChannels) * c.config.PercentPrivateChannels)),
			idleTimeMs: 1000,
		},
		"createDirectChannel": {
			run:        c.createDirectChannel,
			frequency:  int(math.Ceil(float64(c.config.NumChannels) * c.config.PercentDirectChannels)),
			idleTimeMs: 1000,
		},
		"createGroupChannel": {
			run:        c.createGroupChannel,
			frequency:  int(math.Ceil(float64(c.config.NumChannels) * c.config.PercentGroupChannels)),
			idleTimeMs: 1000,
		},
	}
	c.runActions(actions, func() bool { return st.get("channels") >= c.config.NumChannels })

	actions = map[string]userAction{
		"joinTeam": {
			run:        control.JoinTeam,
			frequency:  100,
			idleTimeMs: 0,
		},
		"joinChannel": {
			run:        c.joinChannel,
			frequency:  int(math.Ceil(float64(c.config.NumChannels))) * 2, // making this proportional to number of channels.
			idleTimeMs: 0,
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
	}

	c.runActions(actions, func() bool {
		return st.get("teams") >= c.config.NumTeams &&
			st.get("channels") >= c.config.NumChannels && // having this again just for clarity
			st.get("posts") >= c.config.NumPosts &&
			st.get("reactions") >= c.config.NumReactions
	})
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

		if st.get(StateTargetChannels) >= c.config.NumChannels {
			delete(actions, "createPublicChannel")
			delete(actions, "createPrivateChannel")
			delete(actions, "createDirectChannel")
			delete(actions, "createGroupChannel")
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
