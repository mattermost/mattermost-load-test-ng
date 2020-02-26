// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package control

import (
	"errors"
	"fmt"
	"io/ioutil"
	"time"

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

func SignUp(u user.User) UserActionResponse {
	if u.Store().Id() != "" {
		return UserActionResponse{Info: "user already signed up"}
	}

	email := u.Store().Email()
	username := u.Store().Username()
	password := u.Store().Password()

	err := u.SignUp(email, username, password)
	if err != nil {
		return UserActionResponse{Err: err}
	}

	return UserActionResponse{Info: fmt.Sprintf("signed up as %s", username)}
}

func Login(u user.User) UserActionResponse {
	err := u.Login()
	if err != nil {
		return UserActionResponse{Err: err}
	}

	// Populate teams and channels.
	teamIds, err := u.GetAllTeams(0, 100)
	if err != nil {
		return UserActionResponse{Err: err}
	}
	for _, teamId := range teamIds {
		if err := u.GetChannelsForTeam(teamId); err != nil {
			return UserActionResponse{Err: err}
		}
	}

	return UserActionResponse{Info: "logged in"}
}

func Logout(u user.User) UserActionResponse {
	err := u.Disconnect()
	if err != nil {
		return UserActionResponse{Err: err}
	}

	ok, err := u.Logout()
	if err != nil {
		return UserActionResponse{Err: err}
	}

	if !ok {
		return UserActionResponse{Err: errors.New("user did not logout")}
	}

	return UserActionResponse{Info: "logged out"}
}

func JoinChannel(u user.User) UserActionResponse {
	userStore := u.Store()
	userId := userStore.Id()
	teams, err := userStore.Teams()
	if err != nil {
		return UserActionResponse{Err: err}
	}
	for _, team := range teams {
		channels, err := userStore.Channels(team.Id)
		for _, channel := range channels {
			if err != nil {
				return UserActionResponse{Err: err}
			}
			cm, err := userStore.ChannelMember(team.Id, userId)
			if err != nil {
				return UserActionResponse{Err: err}
			}
			if cm.UserId == "" {
				err := u.AddChannelMember(channel.Id, userId)
				if err != nil {
					return UserActionResponse{Err: err}
				}
				return UserActionResponse{Info: fmt.Sprintf("joined channel %s", channel.Id)}
			}
		}
	}
	return UserActionResponse{Info: "no channel to join"}
}

func LeaveChannel(u user.User) UserActionResponse {
	userStore := u.Store()
	userId := userStore.Id()
	teams, err := userStore.Teams()
	if err != nil {
		return UserActionResponse{Err: err}
	}
	for _, team := range teams {
		channels, err := userStore.Channels(team.Id)
		for _, channel := range channels {
			if err != nil {
				return UserActionResponse{Err: err}
			}
			cm, err := userStore.ChannelMember(team.Id, userId)
			if err != nil {
				return UserActionResponse{Err: err}
			}
			if cm.UserId != "" {
				_, err := u.RemoveUserFromChannel(channel.Id, userId)
				if err != nil {
					return UserActionResponse{Err: err}
				}
				return UserActionResponse{Info: fmt.Sprintf("left channel %s", channel.Id)}
			}
		}
	}
	return UserActionResponse{Info: "unable to leave, not member of any channel"}
}

func JoinTeam(u user.User) UserActionResponse {
	userStore := u.Store()
	userId := userStore.Id()
	teams, err := userStore.Teams()
	if err != nil {
		return UserActionResponse{Err: err}
	}
	for _, team := range teams {
		tm, err := userStore.TeamMember(team.Id, userId)
		if err != nil {
			return UserActionResponse{Err: err}
		}
		if tm.UserId == "" {
			err := u.AddTeamMember(team.Id, userId)
			if err != nil {
				return UserActionResponse{Err: err}
			}
			return UserActionResponse{Info: fmt.Sprintf("joined team %s", team.Id)}
		}
	}
	return UserActionResponse{Info: "no teams to join"}
}

func CreatePost(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam()
	if err != nil {
		return UserActionResponse{Err: err}
	}
	channel, err := u.Store().RandomChannel(team.Id)
	if err != nil {
		return UserActionResponse{Err: err}
	}

	postId, err := u.CreatePost(&model.Post{
		Message:   "Lorem ipsum dolor sit amet, consectetur adipiscing elit",
		ChannelId: channel.Id,
		CreateAt:  time.Now().Unix() * 1000,
	})

	if err != nil {
		return UserActionResponse{Err: err}
	}

	return UserActionResponse{Info: fmt.Sprintf("post created, id %v", postId)}
}

func AddReaction(u user.User) UserActionResponse {
	// get posts from UserStore that have been created in the last minute
	posts, err := u.Store().PostsSince(time.Now().Add(-1*time.Minute).Unix() * 1000)
	if err != nil {
		return UserActionResponse{Err: err}
	}
	if len(posts) == 0 {
		return UserActionResponse{Info: "no posts to add reaction to"}
	}

	err = u.SaveReaction(&model.Reaction{
		UserId:    u.Store().Id(),
		PostId:    posts[0].Id,
		EmojiName: "grinning",
	})

	if err != nil {
		return UserActionResponse{Err: err}
	}

	return UserActionResponse{Info: "added reaction"}
}

func RemoveReaction(u user.User) UserActionResponse {
	// get posts from UserStore that have been created in the last minute
	posts, err := u.Store().PostsSince(time.Now().Add(-1*time.Minute).Unix() * 1000)
	if err != nil {
		return UserActionResponse{Err: err}
	}
	if len(posts) == 0 {
		return UserActionResponse{Info: "no posts to remove reaction from"}
	}

	reactions, err := u.Store().Reactions(posts[0].Id)
	if err != nil {
		return UserActionResponse{Err: err}
	}
	if len(reactions) == 0 {
		return UserActionResponse{Info: "no reactions to remove"}
	}

	err = u.DeleteReaction(&reactions[0])
	if err != nil {
		return UserActionResponse{Err: err}
	}

	return UserActionResponse{Info: "removed reaction"}
}

func CreateGroupChannel(u user.User) UserActionResponse {
	var userIds []string
	users, err := u.Store().RandomUsers(3)
	if err != nil {
		return UserActionResponse{Err: err}
	}
	for _, user := range users {
		userIds = append(userIds, user.Id)
	}

	channelId, err := u.CreateGroupChannel(userIds)
	if err != nil {
		return UserActionResponse{Err: err}
	}

	return UserActionResponse{Info: fmt.Sprintf("group channel created, id %v with users %+v", channelId, userIds)}
}

func CreatePublicChannel(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam()
	if err != nil {
		return UserActionResponse{Err: err}
	}

	channelId, err := u.CreateChannel(&model.Channel{
		Name:   model.NewId(),
		TeamId: team.Id,
		Type:   "O",
	})

	if err != nil {
		return UserActionResponse{Err: err}
	}

	return UserActionResponse{Info: fmt.Sprintf("public channel created, id %v", channelId)}
}

func CreatePrivateChannel(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam()
	if err != nil {
		return UserActionResponse{Err: err}
	}

	channelId, err := u.CreateChannel(&model.Channel{
		Name:   model.NewId(),
		TeamId: team.Id,
		Type:   "P",
	})

	if err != nil {
		return UserActionResponse{Err: err}
	}

	return UserActionResponse{Info: fmt.Sprintf("private channel created, id %v", channelId)}
}

func CreateDirectChannel(u user.User) UserActionResponse {
	user, err := u.Store().RandomUser()
	if err != nil {
		return UserActionResponse{Err: err}
	}

	channelId, err := u.CreateDirectChannel(user.Id)

	if err != nil {
		return UserActionResponse{Err: err}
	}

	return UserActionResponse{Info: fmt.Sprintf("direct channel for user %v created, id %v", user.Id, channelId)}
}

func ViewChannel(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam()
	if err != nil {
		return UserActionResponse{Err: err}
	}
	channel, err := u.Store().RandomChannel(team.Id)
	if err != nil {
		return UserActionResponse{Err: err}
	}

	channelViewResponse, err := u.ViewChannel(&model.ChannelView{
		ChannelId:     channel.Id,
		PrevChannelId: "",
	})
	if err != nil {
		return UserActionResponse{Err: err}
	}

	return UserActionResponse{Info: fmt.Sprintf("channel viewed. result: %v", channelViewResponse.ToJson())}
}

func SearchUsers(u user.User) UserActionResponse {
	teams, err := u.Store().Teams()
	if err != nil {
		return UserActionResponse{Err: err}
	}
	if len(teams) == 0 {
		return UserActionResponse{Info: "no teams to search for users"}
	}

	users, err := u.SearchUsers(&model.UserSearch{
		Term:  "test",
		Limit: 100,
	})
	if err != nil {
		return UserActionResponse{Err: err}
	}

	return UserActionResponse{Info: fmt.Sprintf("found %d users", len(users))}
}

func UpdateProfileImage(u user.User) UserActionResponse {
	// TODO: take this from the config later.
	imagePath := "./testdata/test_profile.png"
	buf, err := ioutil.ReadFile(imagePath)
	if err != nil {
		return UserActionResponse{Err: err}
	}
	err = u.SetProfileImage(buf)
	if err != nil {
		return UserActionResponse{Err: err}
	}
	return UserActionResponse{Info: "profile image updated"}
}

func SearchChannels(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam()
	if err != nil {
		return UserActionResponse{Err: err}
	}

	channels, err := u.SearchChannels(team.Id, &model.ChannelSearch{
		Term: "test",
	})
	if err != nil {
		return UserActionResponse{Err: err}
	}

	return UserActionResponse{Info: fmt.Sprintf("found %d channels", len(channels))}
}

func SearchPosts(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam()
	if err != nil {
		return UserActionResponse{Err: err}
	}

	list, err := u.SearchPosts(team.Id, "test search", false)
	if err != nil {
		return UserActionResponse{Err: err}
	}

	return UserActionResponse{Info: fmt.Sprintf("found %d posts", len(list.Posts))}
}

func ViewUser(u user.User) UserActionResponse {
	team, err := u.Store().RandomTeam()
	if err != nil {
		return UserActionResponse{Err: err}
	}
	channel, err := u.Store().RandomChannel(team.Id)
	if err != nil {
		return UserActionResponse{Err: err}
	}

	err = u.GetChannelMembers(channel.Id, 0, 100)
	if err != nil {
		return UserActionResponse{Err: err}
	}

	member, err := u.Store().RandomChannelMember(channel.Id)
	if err != nil {
		return UserActionResponse{Err: err}
	}

	// GetUsersByIds for that userid
	_, err = u.GetUsersByIds([]string{member.UserId})
	if err != nil {
		return UserActionResponse{Err: err}
	}
	return UserActionResponse{Info: fmt.Sprintf("viewed user %s", member.UserId)}
}
