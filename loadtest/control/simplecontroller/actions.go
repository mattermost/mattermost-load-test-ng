// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simplecontroller

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math"
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

	email := c.user.Store().Email()
	username := c.user.Store().Username()
	password := c.user.Store().Password()

	err := c.user.SignUp(email, username, password)
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus(fmt.Sprintf("signed up as %s", username))
}

func (c *SimpleController) login() control.UserStatus {
	// return here if already logged in
	err := c.user.Login()
	if err != nil {
		return c.newErrorStatus(err)
	}

	c.connect()

	// Populate teams and channels.
	teamIds, err := c.user.GetAllTeams(0, 100)
	if err != nil {
		return c.newErrorStatus(err)
	}
	for _, teamId := range teamIds {
		err := c.user.GetChannelsForTeam(teamId)
		if err != nil {
			return c.newErrorStatus(err)
		}
	}

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

func (c *SimpleController) joinChannel() control.UserStatus {
	userStore := c.user.Store()
	userId := userStore.Id()
	teams, err := userStore.Teams()
	if err != nil {
		return c.newErrorStatus(err)
	}
	for _, team := range teams {
		channels, err := userStore.Channels(team.Id)
		for _, channel := range channels {
			if err != nil {
				return c.newErrorStatus(err)
			}
			cm, err := userStore.ChannelMember(team.Id, userId)
			if err != nil {
				return c.newErrorStatus(err)
			}
			if cm.UserId == "" {
				err := c.user.AddChannelMember(channel.Id, userId)
				if err != nil {
					return c.newErrorStatus(err)
				}
				return c.newInfoStatus(fmt.Sprintf("joined channel %s", channel.Id))
			}
		}
	}
	return c.newInfoStatus("no channel to join")
}

func (c *SimpleController) leaveChannel() control.UserStatus {
	userStore := c.user.Store()
	userId := userStore.Id()
	teams, err := userStore.Teams()
	if err != nil {
		return c.newErrorStatus(err)
	}
	for _, team := range teams {
		channels, err := userStore.Channels(team.Id)
		for _, channel := range channels {
			if err != nil {
				return c.newErrorStatus(err)
			}
			cm, err := userStore.ChannelMember(team.Id, userId)
			if err != nil {
				return c.newErrorStatus(err)
			}
			if cm.UserId != "" {
				_, err := c.user.RemoveUserFromChannel(channel.Id, userId)
				if err != nil {
					return c.newErrorStatus(err)
				}
				return c.newInfoStatus(fmt.Sprintf("left channel %s", channel.Id))
			}
		}
	}
	return c.newInfoStatus("unable to leave, not member of any channel")
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
	team, err := c.user.Store().RandomTeam()
	if err != nil {
		return c.newErrorStatus(err)
	}
	channel, err := c.user.Store().RandomChannel(team.Id)
	if err != nil {
		return c.newErrorStatus(err)
	}

	postId, err := c.user.CreatePost(&model.Post{
		Message:   "Lorem ipsum dolor sit amet, consectetur adipiscing elit",
		ChannelId: channel.Id,
		CreateAt:  time.Now().Unix() * 1000,
	})

	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus(fmt.Sprintf("post created, id %v", postId))
}

func (c *SimpleController) addReaction() control.UserStatus {
	// get posts from UserStore that have been created in the last minute
	posts, err := c.user.Store().PostsSince(time.Now().Unix()*1000 - 60000)
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
	posts, err := c.user.Store().PostsSince(time.Now().Unix()*1000 - 60000)
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
	var userIds []string
	users, err := c.user.Store().RandomUsers(3)
	if err != nil {
		return c.newErrorStatus(err)
	}
	for _, user := range users {
		userIds = append(userIds, user.Id)
	}

	channelId, err := c.user.CreateGroupChannel(userIds)
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus(fmt.Sprintf("group channel created, id %v with users %+v", channelId, userIds))
}

// func (c *SimpleController) createPublicChannel() control.UserStatus {
// 	team, err := c.user.Store().RandomTeam()
// 	if err != nil {
// 		return c.newErrorStatus(err)
// 	}

// 	channelId, err := c.user.CreateChannel(&model.Channel{
// 		Name:   model.NewId(),
// 		TeamId: team.Id,
// 		Type:   "O",
// 	})

// 	if err != nil {
// 		return c.newErrorStatus(err)
// 	}

// 	return c.newInfoStatus(fmt.Sprintf("public channel created, id %v", channelId))
// }

// func (c *SimpleController) createPrivateChannel() control.UserStatus {
// 	team, err := c.user.Store().RandomTeam()
// 	if err != nil {
// 		return c.newErrorStatus(err)
// 	}

// 	channelId, err := c.user.CreateChannel(&model.Channel{
// 		Name:   model.NewId(),
// 		TeamId: team.Id,
// 		Type:   "P",
// 	})

// 	if err != nil {
// 		return c.newErrorStatus(err)
// 	}

// 	return c.newInfoStatus(fmt.Sprintf("private channel created, id %v", channelId))
// }

func (c *SimpleController) createDirectChannel() control.UserStatus {
	user, err := c.user.Store().RandomUser()
	if err != nil {
		return c.newErrorStatus(err)
	}

	channelId, err := c.user.CreateDirectChannel(user.Id)

	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus(fmt.Sprintf("direct channel  for user %v created, id %v", user.Id, channelId))
}

func (c *SimpleController) viewChannel() control.UserStatus {
	team, err := c.user.Store().RandomTeam()
	if err != nil {
		return c.newErrorStatus(err)
	}
	channel, err := c.user.Store().RandomChannel(team.Id)
	if err != nil {
		return c.newErrorStatus(err)
	}

	channelViewResponse, err := c.user.ViewChannel(&model.ChannelView{
		ChannelId:     channel.Id,
		PrevChannelId: "",
	})
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus(fmt.Sprintf("channel viewed. result: %v", channelViewResponse.ToJson()))
}

func (c *SimpleController) scrollChannel() control.UserStatus {
	team, err := c.user.Store().RandomTeam()
	if err != nil {
		return c.newErrorStatus(err)
	}
	channel, err := c.user.Store().RandomChannel(team.Id)
	if err != nil {
		return c.newErrorStatus(err)
	}

	err = c.user.GetPostsForChannel(channel.Id, 0, 1)
	if err != nil {
		return c.newErrorStatus(err)
	}
	posts, err := c.user.Store().ChannelPostsSorted(channel.Id, true)
	if err != nil {
		return c.newErrorStatus(err)
	}

	postId := posts[0].Id // get the oldest post
	const NUM_OF_SCROLLS = 3
	const SLEEP_BETWEEN_SCROLL = 1000
	for i := 0; i < NUM_OF_SCROLLS; i++ {
		if err = c.user.GetPostsBefore(channel.Id, postId, 0, 10); err != nil {
			return c.newErrorStatus(err)
		}
		posts, err := c.user.Store().ChannelPostsSorted(channel.Id, false)
		if err != nil {
			return c.newErrorStatus(err)
		}
		postId = posts[0].Id // get the newest post
		idleTime := time.Duration(math.Round(float64(SLEEP_BETWEEN_SCROLL) * c.rate))
		time.Sleep(time.Millisecond * idleTime)
	}
	return c.newInfoStatus(fmt.Sprintf("scrolled channel %v %d times", channel.Id, NUM_OF_SCROLLS))
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
	userName := fmt.Sprintf("%s-new", c.user.Store().Username())
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
	team, err := c.user.Store().RandomTeam()
	if err != nil {
		return c.newErrorStatus(err)
	}

	channels, err := c.user.SearchChannels(team.Id, &model.ChannelSearch{
		Term: "test",
	})
	if err != nil {
		return c.newErrorStatus(err)
	}

	return c.newInfoStatus(fmt.Sprintf("found %d channels", len(channels)))
}

func (c *SimpleController) searchPosts() control.UserStatus {
	team, err := c.user.Store().RandomTeam()
	if err != nil {
		return c.newErrorStatus(err)
	}

	list, err := c.user.SearchPosts(team.Id, "test search", false)
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

func (c *SimpleController) viewUser() control.UserStatus {
	team, err := c.user.Store().RandomTeam()
	if err != nil {
		return c.newErrorStatus(err)
	}
	channel, err := c.user.Store().RandomChannel(team.Id)
	if err != nil {
		return c.newErrorStatus(err)
	}

	err = c.user.GetChannelMembers(channel.Id, 0, 100)
	if err != nil {
		return c.newErrorStatus(err)
	}

	member, err := c.user.Store().RandomChannelMember(channel.Id)
	if err != nil {
		return c.newErrorStatus(err)
	}

	// GetUsersByIds for that userid
	_, err = c.user.GetUsersByIds([]string{member.UserId})
	if err != nil {
		return c.newErrorStatus(err)
	}
	return c.newInfoStatus(fmt.Sprintf("viewed user %s", member.UserId))
}

func (c *SimpleController) connect() {
	errChan := c.user.Connect()
	go func() {
		for err := range errChan {
			c.status <- c.newErrorStatus(err)
		}
	}()
}
