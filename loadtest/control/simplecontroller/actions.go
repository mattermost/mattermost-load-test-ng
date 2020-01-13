// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simplecontroller

import (
	"errors"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-server/v5/model"
)

type UserAction struct {
	run       func() control.UserStatus
	waitAfter time.Duration
}

func (c *SimpleController) signUp() control.UserStatus {
	if c.user.Store().Id() != "" {
		return c.newInfoStatus("user already signed up")
	}

	email := fmt.Sprintf("testuser%d@example.com", c.id)
	username := fmt.Sprintf("testuser%d", c.id)
	password := "testPass123$"

	err := c.user.SignUp(email, username, password)
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus("signed up")
}

func (c *SimpleController) login() control.UserStatus {
	// return here if already logged in
	err := c.user.Login()
	if err != nil {
		return c.newErrorStatus(err)
	}

	c.connect()

	return c.newInfoStatus("logged in")
}

func (c *SimpleController) logout() control.UserStatus {
	// return here if already logged out

	err := c.user.Disconnect()
	if err != nil {
		return c.newErrorStatus(err)
	}

	ok, err := c.user.Logout()
	if err != nil {
		return c.newErrorStatus(err)
	}

	if !ok {
		return c.newErrorStatus(errors.New("user did not logout"))
	}

	return c.newInfoStatus("logged out")
}

func (c *SimpleController) joinTeam() control.UserStatus {
	userStore := c.user.Store()
	userId := userStore.Id()
	teams, err := userStore.Teams()
	if err != nil {
		return c.newErrorStatus(err)
	}
	for _, team := range teams {
		tm, err := userStore.TeamMember(team.Id, userId)
		if err != nil {
			return c.newErrorStatus(err)
		}
		if tm.UserId == "" {
			err := c.user.AddTeamMember(team.Id, userId)
			if err != nil {
				return c.newErrorStatus(err)
			}
			return c.newInfoStatus(fmt.Sprintf("joined team %s", team.Id))
		}
	}
	return c.newInfoStatus("no teams to join")
}

func (c *SimpleController) createPost() control.UserStatus {
	postId, err := c.user.CreatePost(&model.Post{
		Message: "Lorem ipsum dolor sit amet, consectetur adipiscing elit",
	})
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus(fmt.Sprintf("post created, id %v", postId))
}

func (c *SimpleController) addReaction() control.UserStatus {
	// get posts from UserStore that have been created in the last minute
	posts, err := c.user.Store().PostsSince(time.Now().Unix()*1000 + 60000)
	if err != nil {
		return c.newErrorStatus(err)
	}
	if len(posts) == 0 {
		return c.newInfoStatus("no posts to add reaction to")
	}

	err = c.user.SaveReaction(&model.Reaction{
		UserId:    c.user.Store().Id(),
		PostId:    posts[0].Id,
		EmojiName: "grinning",
	})

	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus("added reaction")
}

func (c *SimpleController) removeReaction() control.UserStatus {
	// get posts from UserStore that have been created in the last minute
	posts, err := c.user.Store().PostsSince(time.Now().Unix()*1000 + 60000)
	if err != nil {
		return c.newErrorStatus(err)
	}
	if len(posts) == 0 {
		return c.newInfoStatus("no posts to remove reaction from")
	}

	reactions, err := c.user.Store().Reactions(posts[0].Id)
	if err != nil {
		return c.newErrorStatus(err)
	}
	if len(reactions) == 0 {
		return c.newInfoStatus("no reactions to remove")
	}

	err = c.user.DeleteReaction(&reactions[0])
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus("removed reaction")
}

func (c *SimpleController) createGroupChannel() control.UserStatus {
	channelId, err := c.user.CreateGroupChannel([]string{}) // TODO: populate memberIds parameter with other users
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus(fmt.Sprintf("group channel created, id %v", channelId))
}

func (c *SimpleController) viewChannel() control.UserStatus {
	return c.newErrorStatus(errors.New("not implemented"))
	/*
		channel, err := c.user.Store().Channel("") // TODO: fetch channel randomly?
		if err != nil {
			return control.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
		}

		channelViewResponse, err := c.user.ViewChannel(&model.ChannelView{
			ChannelId: channel.Id,
			PrevChannelId: "",
		})
		if err != nil {
			return control.UserStatus{User: c.user, Err: err, Code: user.STATUS_ERROR}
		}

		return control.UserStatus{User: c.user, Info: fmt.Sprintf("channel viewed. result: %v", channelViewResponse.ToJson())}
	*/
}

func (c *SimpleController) searchUsers() control.UserStatus {
	teams, err := c.user.Store().Teams()
	if err != nil {
		return c.newErrorStatus(err)
	}
	if len(teams) == 0 {
		return c.newInfoStatus("no teams to search for users")
	}

	users, err := c.user.SearchUsers(&model.UserSearch{
		Term:  "test",
		Limit: 100,
	})
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus(fmt.Sprintf("found %d users", len(users)))
}

func (c *SimpleController) updateProfile() control.UserStatus {
	userId := c.user.Store().Id()
	userName := fmt.Sprintf("testuserNew%d", c.id)
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
		return c.newErrorStatus(err)
	}
	return c.newInfoStatus("user patched")
}

func (c *SimpleController) updateProfileImage() control.UserStatus {
	// TODO: take this from the config later.
	imagePath := "./testdata/test_profile.png"
	buf, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return c.newErrorStatus(err)
	}
	err = c.user.SetProfileImage(buf)
	if err != nil {
		return c.newErrorStatus(err)
	}
	return c.newInfoStatus("profile image updated")
}

func (c *SimpleController) searchChannels() control.UserStatus {
	teams, err := c.user.Store().Teams()
	if err != nil {
		return c.newErrorStatus(err)
	}
	if len(teams) == 0 {
		return c.newInfoStatus("no teams to search for channels")
	}

	channels, err := c.user.SearchChannels(teams[0].Id, &model.ChannelSearch{
		Term: "test",
	})
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus(fmt.Sprintf("found %d channels", len(channels)))
}

func (c *SimpleController) searchPosts() control.UserStatus {
	teams, err := c.user.Store().Teams()
	if err != nil {
		return c.newErrorStatus(err)
	}
	if len(teams) == 0 {
		return c.newInfoStatus("no teams to search for posts")
	}

	list, err := c.user.SearchPosts(teams[0].Id, "test search", false)
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus(fmt.Sprintf("found %d posts", len(list.Posts)))
}

// reload performs all actions done when a user reloads the browser.
// If full parameter is enabled, it also disconnects and reconnects
// the WebSocket connection.
func (c *SimpleController) reload(full bool) control.UserStatus {
	if full {
		err := c.user.Disconnect()
		if err != nil {
			c.status <- c.newErrorStatus(err)
		}

		c.connect()
	}

	// Getting preferences.
	err := c.user.GetPreferences()
	if err != nil {
		return c.newErrorStatus(err)
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
			return c.newErrorStatus(err)
		}
	}

	if ok, err := c.user.IsSysAdmin(); ok && err != nil {
		err = c.user.GetConfig()
		if err != nil {
			return c.newErrorStatus(err)
		}
	}

	err = c.user.GetClientLicense()
	if err != nil {
		return c.newErrorStatus(err)
	}

	// Getting the user.
	userId, err := c.user.GetMe()
	if err != nil {
		return c.newErrorStatus(err)
	}

	_, err = c.user.GetTeams()
	if err != nil {
		return c.newErrorStatus(err)
	}

	err = c.user.GetTeamMembersForUser(userId)
	if err != nil {
		return c.newErrorStatus(err)
	}

	roles, _ := c.user.Store().Roles()
	var roleNames []string
	for _, role := range roles {
		roleNames = append(roleNames, role.Name)
	}
	if len(roleNames) > 0 {
		_, err = c.user.GetRolesByNames(roleNames)
		if err != nil {
			return c.newErrorStatus(err)
		}
	}

	err = c.user.GetWebappPlugins()
	if err != nil {
		return c.newErrorStatus(err)
	}

	_, err = c.user.GetAllTeams(0, 50)
	if err != nil {
		return c.newErrorStatus(err)
	}

	if teamId != "" {
		err = c.user.GetChannelsForTeam(teamId)
		if err != nil {
			return c.newErrorStatus(err)
		}

		err = c.user.GetChannelMembersForUser(userId, teamId)
		if err != nil {
			return c.newErrorStatus(err)
		}
	}

	// Getting unread teams.
	_, err = c.user.GetTeamsUnread("")
	if err != nil {
		return c.newErrorStatus(err)
	}

	if len(userIds) > 0 {
		// Get users by Ids.
		_, err := c.user.GetUsersByIds(userIds)
		if err != nil {
			return c.newErrorStatus(err)
		}

		// Get user statuses by Ids.
		err = c.user.GetUsersStatusesByIds(userIds)
		if err != nil {
			return c.newErrorStatus(err)
		}
	}

	err = c.user.GetUserStatus()
	if err != nil {
		return c.newErrorStatus(err)
	}

	if chanId != "" {
		// Getting the channel stats.
		err = c.user.GetChannelStats(chanId)
		if err != nil {
			return c.newErrorStatus(err)
		}

		// Getting channel unread.
		_, err = c.user.GetChannelUnread(chanId)
		if err != nil {
			return c.newErrorStatus(err)
		}
	}

	return c.newInfoStatus("page reloaded")
}

func (c *SimpleController) connect() {
	errChan := c.user.Connect()
	go func() {
		for err := range errChan {
			c.status <- c.newErrorStatus(err)
		}
	}()
}
