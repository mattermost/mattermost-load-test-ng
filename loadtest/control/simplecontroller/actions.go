// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simplecontroller

import (
	"errors"
	"fmt"
	"math"
	"sync/atomic"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost-server/v5/model"
)

type UserAction struct {
	run       control.UserAction
	waitAfter time.Duration
	runPeriod int
}

func (c *SimpleController) sendDirectMessage(userID string) control.UserStatus {
	channelId := model.GetDMNameFromIds(userID, c.user.Store().Id())
	ok, err := c.user.Store().Channel(channelId)
	if err != nil {
		return c.newErrorStatus(err)
	}
	// We check if a direct channel has been made between the users,
	// and send the message only if it exists.
	if ok == nil {
		return c.newInfoStatus("skipping sending direct message")
	}

	postId, err := c.user.CreatePost(&model.Post{
		Message:   "Lorem ipsum dolor sit amet, consectetur adipiscing elit",
		ChannelId: channelId,
		CreateAt:  time.Now().Unix() * 1000,
	})
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus(fmt.Sprintf("direct post created, id %v", postId))
}

func (c *SimpleController) scrollChannel(u user.User) control.UserActionResponse {
	team, err := c.user.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	channel, err := c.user.Store().RandomChannel(team.Id, store.SelectMemberOf)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	err = c.user.GetPostsForChannel(channel.Id, 0, 1)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	posts, err := c.user.Store().ChannelPostsSorted(channel.Id, true)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if len(posts) == 0 {
		return control.UserActionResponse{Info: fmt.Sprintf("no posts in channel %v", channel.Id)}
	}

	postId := posts[0].Id // get the oldest post
	const NUM_OF_SCROLLS = 3
	const SLEEP_BETWEEN_SCROLL = 1000
	for i := 0; i < NUM_OF_SCROLLS; i++ {
		if err = c.user.GetPostsBefore(channel.Id, postId, 0, 10); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		posts, err := c.user.Store().ChannelPostsSorted(channel.Id, false)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		postId = posts[0].Id // get the newest post
		idleTime := time.Duration(math.Round(float64(SLEEP_BETWEEN_SCROLL) * c.rate))
		time.Sleep(time.Millisecond * idleTime)
	}
	return control.UserActionResponse{Info: fmt.Sprintf("scrolled channel %v %d times", channel.Id, NUM_OF_SCROLLS)}
}

func (c *SimpleController) updateProfile(u user.User) control.UserActionResponse {
	userId := c.user.Store().Id()

	userName := control.RandomizeUserName(c.user.Store().Username())
	nickName := fmt.Sprintf("testNickName%d", c.id)
	firstName := fmt.Sprintf("firstName%d", c.id)
	lastName := fmt.Sprintf("lastName%d", c.id)
	err := c.user.PatchUser(userId, &model.UserPatch{
		Username:  &userName,
		Nickname:  &nickName,
		FirstName: &firstName,
		LastName:  &lastName,
	})
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: "user patched"}
}

func (c *SimpleController) updateTeam(user.User) control.UserActionResponse {
	if ok, err := c.user.IsTeamAdmin(); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if !ok {
		return control.UserActionResponse{Info: "user doesn't have permission to update"}
	}

	team, err := c.user.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	team.DisplayName = control.RandomizeTeamDisplayName(team.DisplayName)

	if err := c.user.UpdateTeam(&team); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: "team updated"}
}

// reload performs all actions done when a user reloads the browser.
// If full parameter is enabled, it also disconnects and reconnects
// the WebSocket connection.
func (c *SimpleController) reload(full bool) control.UserActionResponse {
	if full {
		err := c.disconnect()
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		c.user.ClearUserData()

		if err := c.connect(); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	return control.Reload(c.user)
}

func (c *SimpleController) connect() error {
	if !atomic.CompareAndSwapInt32(&c.connectedFlag, 0, 1) {
		return errors.New("already connected")
	}
	errChan, err := c.user.Connect()
	if err != nil {
		atomic.StoreInt32(&c.connectedFlag, 0)
		return fmt.Errorf("connect failed %w", err)
	}

	c.wg.Add(2)
	go func() {
		defer c.wg.Done()
		for err := range errChan {
			c.status <- c.newErrorStatus(err)
		}
	}()
	go c.wsEventHandler(c.wg)
	return nil
}

func (c *SimpleController) disconnect() error {
	if !atomic.CompareAndSwapInt32(&c.connectedFlag, 1, 0) {
		return errors.New("not connected")
	}

	err := c.user.Disconnect()
	if err != nil {
		return fmt.Errorf("disconnect failed %w", err)
	}
	c.wg.Wait()
	return nil
}
