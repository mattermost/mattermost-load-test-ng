// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simplecontroller

import (
	"fmt"
	"math"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost-server/v5/model"
)

type UserAction struct {
	run          control.UserAction
	waitAfter    time.Duration
	runFrequency int
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
	team, err := c.user.Store().RandomTeam()
	if err != nil {
		return control.UserActionResponse{Err: err}
	}
	channel, err := c.user.Store().RandomChannel(team.Id)
	if err != nil {
		return control.UserActionResponse{Err: err}
	}

	err = c.user.GetPostsForChannel(channel.Id, 0, 1)
	if err != nil {
		return control.UserActionResponse{Err: err}
	}
	posts, err := c.user.Store().ChannelPostsSorted(channel.Id, true)
	if err != nil {
		return control.UserActionResponse{Err: err}
	}

	postId := posts[0].Id // get the oldest post
	const NUM_OF_SCROLLS = 3
	const SLEEP_BETWEEN_SCROLL = 1000
	for i := 0; i < NUM_OF_SCROLLS; i++ {
		if err = c.user.GetPostsBefore(channel.Id, postId, 0, 10); err != nil {
			return control.UserActionResponse{Err: err}
		}
		posts, err := c.user.Store().ChannelPostsSorted(channel.Id, false)
		if err != nil {
			return control.UserActionResponse{Err: err}
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
		return control.UserActionResponse{Err: err}
	}

	return control.UserActionResponse{Info: "user patched"}
}

// reload performs all actions done when a user reloads the browser.
// If full parameter is enabled, it also disconnects and reconnects
// the WebSocket connection.
func (c *SimpleController) reload(full bool) control.UserActionResponse {
	if full {
		err := c.user.Disconnect()
		if err != nil {
			return control.UserActionResponse{Err: err}
		}

		c.connect()
	}

	// Getting preferences.
	err := c.user.GetPreferences()
	if err != nil {
		return control.UserActionResponse{Err: err}
	}

	prefs, _ := c.user.Store().Preferences()
	var userIds []string
	chanId := ""
	teamId := ""
	for _, p := range prefs {
		switch {
		case p.Name == model.PREFERENCE_NAME_LAST_CHANNEL:
			chanId = p.Value
		case p.Name == model.PREFERENCE_NAME_LAST_TEAM:
			teamId = p.Value
		case p.Category == model.PREFERENCE_CATEGORY_DIRECT_CHANNEL_SHOW:
			userIds = append(userIds, p.Name)
		}
	}

	if chanId != "" {
		// Marking the channel as viewed
		_, err := c.user.ViewChannel(&model.ChannelView{
			ChannelId:     chanId,
			PrevChannelId: "",
		})
		if err != nil {
			return control.UserActionResponse{Err: err}
		}
	}

	if ok, err := c.user.IsSysAdmin(); ok && err != nil {
		err = c.user.GetConfig()
		if err != nil {
			return control.UserActionResponse{Err: err}
		}
	}

	err = c.user.GetClientLicense()
	if err != nil {
		return control.UserActionResponse{Err: err}
	}

	// Getting the user.
	userId, err := c.user.GetMe()
	if err != nil {
		return control.UserActionResponse{Err: err}
	}

	_, err = c.user.GetTeams()
	if err != nil {
		return control.UserActionResponse{Err: err}
	}

	err = c.user.GetTeamMembersForUser(userId)
	if err != nil {
		return control.UserActionResponse{Err: err}
	}

	roles, _ := c.user.Store().Roles()
	var roleNames []string
	for _, role := range roles {
		roleNames = append(roleNames, role.Name)
	}
	if len(roleNames) > 0 {
		_, err = c.user.GetRolesByNames(roleNames)
		if err != nil {
			return control.UserActionResponse{Err: err}
		}
	}

	err = c.user.GetWebappPlugins()
	if err != nil {
		return control.UserActionResponse{Err: err}
	}

	_, err = c.user.GetAllTeams(0, 50)
	if err != nil {
		return control.UserActionResponse{Err: err}
	}

	if teamId != "" {
		err = c.user.GetChannelsForTeam(teamId)
		if err != nil {
			return control.UserActionResponse{Err: err}
		}

		err = c.user.GetChannelMembersForUser(userId, teamId)
		if err != nil {
			return control.UserActionResponse{Err: err}
		}
	}

	// Getting unread teams.
	_, err = c.user.GetTeamsUnread("")
	if err != nil {
		return control.UserActionResponse{Err: err}
	}

	if len(userIds) > 0 {
		// Get users by Ids.
		_, err := c.user.GetUsersByIds(userIds)
		if err != nil {
			return control.UserActionResponse{Err: err}
		}

		// Get user statuses by Ids.
		err = c.user.GetUsersStatusesByIds(userIds)
		if err != nil {
			return control.UserActionResponse{Err: err}
		}
	}

	err = c.user.GetUserStatus()
	if err != nil {
		return control.UserActionResponse{Err: err}
	}

	if chanId != "" {
		// Getting the channel stats.
		err = c.user.GetChannelStats(chanId)
		if err != nil {
			return control.UserActionResponse{Err: err}
		}

		// Getting channel unread.
		_, err = c.user.GetChannelUnread(chanId)
		if err != nil {
			return control.UserActionResponse{Err: err}
		}
	}

	return control.UserActionResponse{Info: "page reloaded"}
}

func (c *SimpleController) connect() {
	errChan := c.user.Connect()
	go func() {
		for err := range errChan {
			c.status <- c.newErrorStatus(err)
		}
	}()
}
