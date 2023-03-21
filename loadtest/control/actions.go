// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package control

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost-server/v6/model"
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

	if err := u.SignUp(email, username, password); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return UserActionResponse{Info: fmt.Sprintf("%s has already signed up", email)}
		}
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
			if err := u.GetChannelsForTeam(teamId, true); err != nil {
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

	err = u.Logout()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
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
			if channel.Name == model.DefaultChannelName {
				continue
			}
			cm, err := userStore.ChannelMember(channel.Id, userId)
			if err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
			if cm.UserId != "" {
				err = u.RemoveUserFromChannel(channel.Id, userId)
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
			if err := u.AddTeamMember(teamId, userId); err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
			if err := u.GetChannelsForTeam(teamId, true); err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
			if err := u.GetChannelMembersForUser(userId, teamId); err != nil {
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

// EditPost updates a post.
func EditPost(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	post, err := u.Store().RandomPostForChannel(channel.Id)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	message := GenerateRandomSentences(rand.Intn(10))
	postId, err := u.PatchPost(post.Id, &model.PostPatch{
		Message: &message,
	})
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	return UserActionResponse{Info: fmt.Sprintf("post updated, id %v -> %v", post.Id, postId)}
}

// CreatePostReply replies to a randomly picked post.
func CreatePostReply(u user.User) UserActionResponse {
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
	postsIds, err := u.Store().PostsIdsSince(time.Now().Add(-1*time.Minute).Unix() * 1000)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	if len(postsIds) == 0 {
		return UserActionResponse{Info: "no posts to add reaction to"}
	}

	postId := postsIds[rand.Intn(len(postsIds))]

	err = u.SaveReaction(&model.Reaction{
		UserId:    u.Store().Id(),
		PostId:    postId,
		EmojiName: "grinning",
	})

	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	return UserActionResponse{Info: fmt.Sprintf("added reaction to post %s", postId)}
}

// RemoveReaction removes a reaction from a random post which is added by the user.
func RemoveReaction(u user.User) UserActionResponse {
	// get posts from UserStore that have been created in the last minute
	postsIds, err := u.Store().PostsIdsSince(time.Now().Add(-1*time.Minute).Unix() * 1000)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	if len(postsIds) == 0 {
		return UserActionResponse{Info: "no posts to remove reaction from"}
	}

	postId := postsIds[rand.Intn(len(postsIds))]
	reactions, err := u.Store().Reactions(postId)
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
			return UserActionResponse{Info: fmt.Sprintf("removed reaction from post %s", postId)}
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

	channelName := model.NewId()
	channelId, err := u.CreateChannel(&model.Channel{
		Name:        channelName,
		DisplayName: "Channel " + channelName,
		TeamId:      team.Id,
		Type:        "O",
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

	_, err = u.ViewChannel(&model.ChannelView{
		ChannelId:     channel.Id,
		PrevChannelId: "",
	})
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	return UserActionResponse{Info: fmt.Sprintf("viewed channel %s", channel.Id)}
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

	return EmulateUserTyping("test", func(term string) UserActionResponse {
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
	err := u.SetProfileImage(MustAsset("test_profile.png"))
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

	return EmulateUserTyping("ch-", func(term string) UserActionResponse {
		channels, err := u.SearchChannelsForTeam(team.Id, &model.ChannelSearch{
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

// FetchStaticAssets parses index.html and fetches static assets mentioned in it
func FetchStaticAssets(u user.User) UserActionResponse {
	err := u.FetchStaticAssets()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	return UserActionResponse{Info: "static assets fetched"}
}

// GetPinnedPosts fetches the pinned posts in a channel that user is a member of.
func GetPinnedPosts(u user.User) UserActionResponse {
	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	list, err := u.GetPinnedPosts(channel.Id)
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

	prefs, err := u.Store().Preferences()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	var userIds []string
	for _, p := range prefs {
		switch {
		case p.Category == model.PreferenceCategoryDirectChannelShow:
			userIds = append(userIds, p.Name)
		case p.Category == "group_channel_show":
			if err := u.GetUsersInChannel(p.Name, 0, 8); err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
		}
	}

	var chanId string
	if c, err := u.Store().CurrentChannel(); err == nil {
		chanId = c.Id
	} else if err != nil && err != memstore.ErrChannelNotFound {
		return UserActionResponse{Err: NewUserError(err)}
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
	isSysAdmin, err := u.IsSysAdmin()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}
	if isSysAdmin {
		err = u.GetConfig()
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}
	} else {
		err = u.GetClientConfig()
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}
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

	var teamId string
	if team, err := u.Store().CurrentTeam(); err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	} else if team != nil {
		teamId = team.Id
	}

	if teamId != "" {
		err = u.GetTeamMembersForUser(u.Store().Id())
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}

		if tm, err := u.Store().TeamMember(teamId, u.Store().Id()); err == nil && tm.UserId != "" {
			if err := u.GetChannelsForTeam(teamId, true); err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
			err = u.GetChannelMembersForUser(userId, teamId)
			if err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
		}

		ver, err := u.Store().ServerVersion()
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}

		ok, err := IsVersionSupported("6.4.0", ver)
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}
		if ok {
			_, err = u.GetChannelsForUser(userId)
			if err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
		}
	}

	// Getting unread teams.
	_, err = u.GetTeamsUnread("", false)
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
		err = u.GetChannelStats(chanId, true)
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

// ReloadGQL is same as Reload but with the REST calls replaced with GraphQL
func ReloadGQL(u user.User) UserActionResponse {
	err := u.GetInitialDataGQL()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	prefs, err := u.Store().Preferences()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	var userIds []string
	for _, p := range prefs {
		switch {
		case p.Category == model.PreferenceCategoryDirectChannelShow:
			userIds = append(userIds, p.Name)
		case p.Category == "group_channel_show":
			if err := u.GetUsersInChannel(p.Name, 0, 8); err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
		}
	}

	var chanId string
	if c, err := u.Store().CurrentChannel(); err == nil {
		chanId = c.Id
	} else if err != nil && err != memstore.ErrChannelNotFound {
		return UserActionResponse{Err: NewUserError(err)}
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

	userId := u.Store().Id()

	err = u.GetWebappPlugins()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	_, err = u.GetAllTeams(0, 50)
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	var teamId string
	if team, err := u.Store().CurrentTeam(); err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	} else if team != nil {
		teamId = team.Id
	}

	if teamId != "" {
		if tm, err := u.Store().TeamMember(teamId, u.Store().Id()); err == nil && tm.UserId != "" {
			if err := u.GetChannelsForTeam(teamId, true); err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
			err = u.GetChannelMembersForUser(userId, teamId)
			if err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
		}

		ver, err := u.Store().ServerVersion()
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}

		ok, err := IsVersionSupported("6.4.0", ver)
		if err != nil {
			return UserActionResponse{Err: NewUserError(err)}
		}
		if ok {
			_, err = u.GetChannelsForUser(userId)
			if err != nil {
				return UserActionResponse{Err: NewUserError(err)}
			}
		}
	}

	// Getting unread teams.
	_, err = u.GetTeamsUnread("", false)
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
		err = u.GetChannelStats(chanId, true)
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

func CollapsedThreadsEnabled(u user.User) (bool, UserActionResponse) {
	if u.Store().ClientConfig()["CollapsedThreads"] == model.CollapsedThreadsDisabled {
		return false, UserActionResponse{}
	}

	collapsedThreads := u.Store().ClientConfig()["CollapsedThreads"] == model.CollapsedThreadsDefaultOn
	prefs, err := u.Store().Preferences()
	if err != nil {
		return false, UserActionResponse{Err: NewUserError(err)}
	}

	for _, p := range prefs {
		if p.Category == model.PreferenceCategoryDisplaySettings && p.Name == model.PreferenceNameCollapsedThreadsEnabled {
			collapsedThreads = p.Value == "true"
			break
		}
	}
	return collapsedThreads, UserActionResponse{}
}

// MessageExport simulates the given user performing
// a compliance message export
func MessageExport(u user.User) UserActionResponse {
	isAdmin, err := u.IsSysAdmin()
	if err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	if !isAdmin {
		return UserActionResponse{Info: "user is not a sysadmin and cannot perform a message export"}
	}

	if err := u.GetConfig(); err != nil {
		return UserActionResponse{Err: NewUserError(err)}
	}

	cfg := u.Store().Config()

	if cfg.MessageExportSettings.EnableExport == nil || !*cfg.MessageExportSettings.EnableExport {
		return UserActionResponse{Info: "message export is not enabled"}
	}

	err = u.MessageExport()
	if err != nil {
		return UserActionResponse{Err: err}
	}

	return UserActionResponse{Info: "message export triggered"}
}
