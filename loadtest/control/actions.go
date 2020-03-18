// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package control

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost-server/v5/model"
)

// UserActionResponse is a structure containing information about the result
// of running a UserAction.
type UserActionResponse struct {
	// Info contains a string with information about the action
	// execution.
	Info string
	// Err contains an error when the action failed.
	Err error
}

// UserAction is a function that simulates a specific behaviour for the provided
// user.User. It returns a UserActionResponse.
type UserAction func(user.User) UserActionResponse

// SignUp signs up the given user to the server.
func SignUp(u user.User) UserActionResponse {
	if u.Store().Id() != "" {
		return UserActionResponse{Info: "user already signed up"}
	}

	email := u.Store().Email()
	username := u.Store().Username()
	password := u.Store().Password()

	err := u.SignUp(email, username, password)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	return UserActionResponse{Info: fmt.Sprintf("signed up as %s", username)}
}

// Login authenticates the user with the server and fetches teams, users and
// channels that are related with the user.
func Login(u user.User) UserActionResponse {
	err := u.Login()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	// Populate teams and channels.
	teamIds, err := u.GetAllTeams(0, 100)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	err = u.GetTeamMembersForUser(u.Store().Id())
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	for _, teamId := range teamIds {
		if tm, err := u.Store().TeamMember(teamId, u.Store().Id()); err == nil && tm.UserId != "" {
			if err := u.GetChannelsForTeam(teamId); err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
		}
	}

	return UserActionResponse{Info: "logged in"}
}

// Logout disconnects the user from the server and logs out from the server.
func Logout(u user.User) UserActionResponse {
	err := u.Disconnect()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	ok, err := u.Logout()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	if !ok {
		return UserActionResponse{Err: NewUserError(errors.New("user did not logout"))}
	}

	return UserActionResponse{Info: "logged out"}
}

// JoinChannel adds the user to the first channel that has been found in the store
// and which the user is not a member of.
func JoinChannel(u user.User) UserActionResponse {
	userStore := u.Store()
	userId := userStore.Id()
	teams, err := userStore.Teams()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	for _, team := range teams {
		channels, err := userStore.Channels(team.Id)
		for _, channel := range channels {
			if err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
			cm, err := userStore.ChannelMember(team.Id, userId)
			if err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
			if cm.UserId == "" {
				err := u.AddChannelMember(channel.Id, userId)
				if err != nil {
					return UserActionResponse{Err: NewUserError(err)}
				}
				return UserActionResponse{Info: fmt.Sprintf("joined channel %s", channel.Id)}
			}
		}
	}
	return UserActionResponse{Info: "no channel to join"}
}

// LeaveChannel removes the user from the first channel that has been found in
// the store and which the user is a member of.
func LeaveChannel(u user.User) UserActionResponse {
	userStore := u.Store()
	userId := userStore.Id()
	teams, err := userStore.Teams()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	for _, team := range teams {
		channels, err := userStore.Channels(team.Id)
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}
		for _, channel := range channels {
			// don't try to leave default channel
			if channel.Name == model.DEFAULT_CHANNEL {
				continue
			}
			cm, err := userStore.ChannelMember(channel.Id, userId)
			if err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
			if cm.UserId != "" {
				_, err := u.RemoveUserFromChannel(channel.Id, userId)
				if err != nil {
					return UserActionResponse{Err: NewUserError(err)}
				}
				return UserActionResponse{Info: fmt.Sprintf("left channel %s", channel.Id)}
			}
		}
	}
	return UserActionResponse{Info: "unable to leave, not member of any channel"}
}

// JoinTeam adds the given user to the first team that has been found in the store
// and which the user is not a member of.
func JoinTeam(u user.User) UserActionResponse {
	userStore := u.Store()
	userId := userStore.Id()
	teamIds, err := u.GetAllTeams(0, 100)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	for _, teamId := range teamIds {
		tm, err := userStore.TeamMember(teamId, userId)
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}
		if tm.UserId == "" {
			err := u.AddTeamMember(teamId, userId)
			if err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
			err = u.GetChannelMembersForUser(userId, teamId)
			if err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
			return UserActionResponse{Info: fmt.Sprintf("joined team %s", teamId)}
		}
	}
	return UserActionResponse{Info: "no teams to join"}
}

// CreatePost creates a new post in a random channel by the given user.
func CreatePost(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf)
	if errors.Is(err, memstore.ErrChannelStoreEmpty) {
		return UserActionResponse{Info: "no channels in store"}
	} else if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	postId, err := u.CreatePost(&model.Post{
		Message:   "Lorem ipsum dolor sit amet, consectetur adipiscing elit",
		ChannelId: channel.Id,
		CreateAt:  time.Now().Unix() * 1000,
	})

	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	return UserActionResponse{Info: fmt.Sprintf("post created, id %v", postId)}
}

// CreatePostReply replies to a randomly picked post.
func CreatePostReply(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeamJoined()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	channel, err := u.Store().RandomChannelJoined(team.Id)
	if errors.Is(err, memstore.ErrChannelStoreEmpty) {
		return UserActionResponse{Info: "no channels in store"}
	} else if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	post, err := u.Store().RandomPostForChannel(channel.Id)
	if errors.Is(err, memstore.ErrPostNotFound) {
		return UserActionResponse{Info: "no post to reply to"}
	} else if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	postId, err := u.CreatePost(&model.Post{
		Message:   "Lorem ipsum dolor sit amet, consectetur adipiscing elit",
		ChannelId: channel.Id,
		CreateAt:  time.Now().Unix() * 1000,
		RootId:    post.Id,
	})

	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	return UserActionResponse{Info: fmt.Sprintf("post reply created, id %v", postId)}
}

// AddReaction adds a reaction by the user to a random post.
func AddReaction(u user.User) UserActionResponse {
	// get posts from UserStore that have been created in the last minute
	posts, err := u.Store().PostsSince(time.Now().Add(-1*time.Minute).Unix() * 1000)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	if len(posts) == 0 {
		return UserActionResponse{Info: "no posts to add reaction to"}
	}

	post := posts[rand.Intn(len(posts))]

	err = u.SaveReaction(&model.Reaction{
		UserId:    u.Store().Id(),
		PostId:    post.Id,
		EmojiName: "grinning",
	})

	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	return UserActionResponse{Info: fmt.Sprintf("added reaction to post %s", post.Id)}
}

// RemoveReaction removes a reaction from a random post which is added by the user.
func RemoveReaction(u user.User) UserActionResponse {
	// get posts from UserStore that have been created in the last minute
	posts, err := u.Store().PostsSince(time.Now().Add(-1*time.Minute).Unix() * 1000)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	if len(posts) == 0 {
		return UserActionResponse{Info: "no posts to remove reaction from"}
	}

	post := posts[rand.Intn(len(posts))]
	reactions, err := u.Store().Reactions(post.Id)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	if len(reactions) == 0 {
		return UserActionResponse{Info: "no reactions to remove"}
	}

	for _, reaction := range reactions {
		if reaction.UserId == u.Store().Id() {
			reaction := reaction
			err = u.DeleteReaction(&reaction)
			if err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
			return UserActionResponse{Info: "removed reaction"}
		}
	}

	return UserActionResponse{Info: "no reactions to remove"}
}

// CreateGroupChannel creates a group channel with 3 random users.
func CreateGroupChannel(u user.User) UserActionResponse {
	var userIds []string
	users, err := u.Store().RandomUsers(3)
	if errors.Is(err, memstore.ErrLenMismatch) {
		return UserActionResponse{Info: "not enough users to create group channel"}
	} else if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	for _, user := range users {
		userIds = append(userIds, user.Id)
	}

	channelId, err := u.CreateGroupChannel(userIds)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	return UserActionResponse{Info: fmt.Sprintf("group channel created, id %v with users %+v", channelId, userIds)}
}

// CreatePublicChannel creates a public channel in a random team.
func CreatePublicChannel(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	channelId, err := u.CreateChannel(&model.Channel{
		Name:   model.NewId(),
		TeamId: team.Id,
		Type:   "O",
	})

	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	return UserActionResponse{Info: fmt.Sprintf("public channel created, id %v", channelId)}
}

// CreatePrivateChannel creates a private channel in a random team.
func CreatePrivateChannel(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	channelId, err := u.CreateChannel(&model.Channel{
		Name:   model.NewId(),
		TeamId: team.Id,
		Type:   "P",
	})

	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	return UserActionResponse{Info: fmt.Sprintf("private channel created, id %v", channelId)}
}

// CreateDirectChannel creates a direct message channel with a random user from a
// random team/channel.
func CreateDirectChannel(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	cm, err := u.Store().RandomChannelMember(channel.Id)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	channelId, err := u.CreateDirectChannel(cm.UserId)

	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	return UserActionResponse{Info: fmt.Sprintf("direct channel for user %v created, id %v", u.Store().Id(), channelId)}
}

// ViewChannel performs a view action in a random team/channel for the given
// user, which will mark all posts as read in the channel.
func ViewChannel(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	channelViewResponse, err := u.ViewChannel(&model.ChannelView{
		ChannelId:     channel.Id,
		PrevChannelId: "",
	})
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	return UserActionResponse{Info: fmt.Sprintf("channel viewed. result: %v", channelViewResponse.ToJson())}
}

// SearchUsers searches for users by the given user.
func SearchUsers(u user.User) UserActionResponse {
	teams, err := u.Store().Teams()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	if len(teams) == 0 {
		return UserActionResponse{Info: "no teams to search for users"}
	}

	return emulateUserTyping("test", func(term string) UserActionResponse {
		users, err := u.SearchUsers(&model.UserSearch{
			Term:  term,
			Limit: 100,
		})
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}
		return UserActionResponse{Info: fmt.Sprintf("found %d users", len(users))}
	})
}

// UpdateProfileImage uploads a new profile picture for the given user.
func UpdateProfileImage(u user.User) UserActionResponse {
	// TODO: take this from the config later.
	imagePath := "./testdata/test_profile.png"
	buf, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	err = u.SetProfileImage(buf)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	return UserActionResponse{Info: "profile image updated"}
}

// SearchChannels searches for channels by the given user.
func SearchChannels(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	return emulateUserTyping("ch-", func(term string) UserActionResponse {
		channels, err := u.SearchChannels(team.Id, &model.ChannelSearch{
			Term: term,
		})
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}
		return UserActionResponse{Info: fmt.Sprintf("found %d channels", len(channels))}
	})
}

// SearchPosts searches for posts by the given user.
func SearchPosts(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	list, err := u.SearchPosts(team.Id, "test search", false)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	return UserActionResponse{Info: fmt.Sprintf("found %d posts", len(list.Posts))}
}

// ViewUser simulates opening a random user profile for the given user.
func ViewUser(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf)
	if errors.Is(err, memstore.ErrChannelStoreEmpty) {
		return UserActionResponse{Info: "no channels in store"}
	} else if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	err = u.GetChannelMembers(channel.Id, 0, 100)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	member, err := u.Store().RandomChannelMember(channel.Id)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	// GetUsersByIds for that userid
	_, err = u.GetUsersByIds([]string{member.UserId})
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	return UserActionResponse{Info: fmt.Sprintf("viewed user %s", member.UserId)}
}

// Reload simulates the given user reloading the page
// while connected to the server by executing the API
// calls seen during a real page reload.
func Reload(u user.User) UserActionResponse {
	// Getting preferences.
	err := u.GetPreferences()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	prefs, _ := u.Store().Preferences()
	var userIds []string
	chanId := ""
	for _, p := range prefs {
		switch {
		case p.Name == model.PREFERENCE_NAME_LAST_CHANNEL:
			chanId = p.Value
		case p.Category == model.PREFERENCE_CATEGORY_DIRECT_CHANNEL_SHOW:
			userIds = append(userIds, p.Name)
		}
	}

	if chanId != "" {
		// Marking the channel as viewed
		_, err := u.ViewChannel(&model.ChannelView{
			ChannelId:     chanId,
			PrevChannelId: "",
		})
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}
	}

	if ok, err := u.IsSysAdmin(); ok && err == nil {
		err = u.GetConfig()
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}
	} else if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	err = u.GetClientLicense()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	// Getting the user.
	userId, err := u.GetMe()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	roles, _ := u.Store().Roles()
	var roleNames []string
	for _, role := range roles {
		roleNames = append(roleNames, role.Name)
	}
	if len(roleNames) > 0 {
		_, err = u.GetRolesByNames(roleNames)
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}
	}

	err = u.GetWebappPlugins()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	_, err = u.GetAllTeams(0, 50)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	_, err = u.GetTeamsForUser(userId)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	// NOTE: during reload, the webapp client will fetch the last viewed team.
	// This information is persistently stored and survives reloads/restarting the browser.
	// Here we simplify that behaviour by randomly picking a team the user is
	// a member of.
	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	if team.Id != "" {
		err = u.GetTeamMembersForUser(u.Store().Id())
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}

		if tm, err := u.Store().TeamMember(team.Id, u.Store().Id()); err == nil && tm.UserId != "" {
			if err := u.GetChannelsForTeam(team.Id); err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
			err = u.GetChannelMembersForUser(userId, team.Id)
			if err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
		}
	}

	// Getting unread teams.
	_, err = u.GetTeamsUnread("")
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	if len(userIds) > 0 {
		// Get users by Ids.
		_, err := u.GetUsersByIds(userIds)
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}

		// Get user statuses by Ids.
		err = u.GetUsersStatusesByIds(userIds)
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}
	}

	err = u.GetUserStatus()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	if chanId != "" {
		// Getting the channel stats.
		err = u.GetChannelStats(chanId)
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}

		// Getting channel unread.
		_, err = u.GetChannelUnread(chanId)
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}
	}

	return UserActionResponse{Info: "page reloaded"}
}
