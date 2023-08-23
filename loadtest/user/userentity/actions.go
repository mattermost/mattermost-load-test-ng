// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/graph-gophers/graphql-go"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost/server/public/model"
)

// SignUp signs up the user with the given credentials.
func (ue *UserEntity) SignUp(email, username, password string) error {
	user := model.User{
		Email:    email,
		Username: username,
		Password: password,
	}

	newUser, _, err := ue.client.CreateUser(context.Background(), &user)
	if err != nil {
		return err
	}

	newUser.Password = password
	return ue.store.SetUser(newUser)
}

// Login logs the user in. It authenticates a user and starts a new session.
func (ue *UserEntity) Login() error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}

	loggedUser, _, err := ue.client.Login(context.Background(), user.Email, user.Password)
	if err != nil {
		return err
	}

	// We need to set user again because the user ID does not get set
	// if a user is already signed up.
	if err := ue.store.SetUser(loggedUser); err != nil {
		return err
	}

	return nil
}

// Logout logs the user out. It terminates the current user's session.
func (ue *UserEntity) Logout() error {
	_, err := ue.client.Logout(context.Background())
	if err != nil {
		return err
	}

	return nil
}

// GetClientConfig fetches and stores the limited server's configuration for logged in user.
func (ue *UserEntity) GetClientConfig() error {
	config, _, err := ue.client.GetOldClientConfig(context.Background(), "")
	if err != nil {
		return err
	}
	ue.store.SetClientConfig(config)
	return nil
}

// GetConfig fetches and stores the server's configuration.
func (ue *UserEntity) GetConfig() error {
	config, _, err := ue.client.GetConfig(context.Background())
	if err != nil {
		return err
	}
	ue.store.SetConfig(config)
	return nil
}

// GetMe loads user's information into the store and returns its id.
func (ue *UserEntity) GetMe() (string, error) {
	user, _, err := ue.client.GetMe(context.Background(), "")
	if err != nil {
		return "", err
	}

	if err := ue.store.SetUser(user); err != nil {
		return "", err
	}

	return user.Id, nil
}

// GetPreferences fetches and store the user's preferences.
func (ue *UserEntity) GetPreferences() error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}

	preferences, _, err := ue.client.GetPreferences(context.Background(), user.Id)
	if err != nil {
		return err
	}

	if err := ue.store.SetPreferences(preferences); err != nil {
		return err
	}
	return nil
}

// UpdatePreferences updates the user's preferences.
func (ue *UserEntity) UpdatePreferences(pref model.Preferences) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}

	if pref == nil {
		return errors.New("userentity: pref should not be nil")
	}

	_, err = ue.client.UpdatePreferences(context.Background(), user.Id, pref)
	if err != nil {
		return err
	}

	return nil
}

// CreateUser creates a new user with the given information.
func (ue *UserEntity) CreateUser(user *model.User) (string, error) {
	user, _, err := ue.client.CreateUser(context.Background(), user)
	if err != nil {
		return "", err
	}

	return user.Id, nil
}

// UpdateUser updates the given user with the given information.
func (ue *UserEntity) UpdateUser(user *model.User) error {
	user, _, err := ue.client.UpdateUser(context.Background(), user)
	if err != nil {
		return err
	}

	if user.Id == ue.store.Id() {
		return ue.store.SetUser(user)
	}

	return nil
}

// UpdateUserRoles updates the given userId with the given role ids.
func (ue *UserEntity) UpdateUserRoles(userId, roles string) error {
	_, err := ue.client.UpdateUserRoles(context.Background(), userId, roles)
	if err != nil {
		return err
	}

	return nil
}

// PatchUser patches a given user with the given information.
func (ue *UserEntity) PatchUser(userId string, patch *model.UserPatch) error {
	user, _, err := ue.client.PatchUser(context.Background(), userId, patch)

	if err != nil {
		return err
	}

	if userId == ue.store.Id() {
		return ue.store.SetUser(user)
	}

	return nil
}

// CreatePost creates and stores a new post made by the user.
func (ue *UserEntity) CreatePost(post *model.Post) (string, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return "", err
	}

	post.PendingPostId = model.NewId()
	post.UserId = user.Id

	post, _, err = ue.client.CreatePost(context.Background(), post)
	if err != nil {
		return "", err
	}

	err = ue.store.SetPost(post)

	return post.Id, err
}

// PatchPost modifies a post for the given postId and stores the updated result.
func (ue *UserEntity) PatchPost(postId string, patch *model.PostPatch) (string, error) {
	post, _, err := ue.client.PatchPost(context.Background(), postId, patch)
	if err != nil {
		return "", err
	}

	if err := ue.store.SetPost(post); err != nil {
		return "", err
	}

	return post.Id, nil
}

// DeletePost deletes a post for the given postId.
func (ue *UserEntity) DeletePost(postId string) error {
	_, err := ue.client.DeletePost(context.Background(), postId)
	if err != nil {
		return err
	}

	if err := ue.store.DeletePost(postId); err != nil {
		return err
	}

	return nil
}

// SearchPosts performs a search for posts in the given teamId with the given terms.
func (ue *UserEntity) SearchPosts(teamId, terms string, isOrSearch bool) (*model.PostList, error) {
	postList, _, err := ue.client.SearchPosts(context.Background(), teamId, terms, isOrSearch)
	if err != nil {
		return nil, err
	}
	return postList, nil
}

// GetPostsForChannel fetches and stores posts in a given channelId.
func (ue *UserEntity) GetPostsForChannel(channelId string, page, perPage int, collapsedThreads bool) error {
	postList, _, err := ue.client.GetPostsForChannel(context.Background(), channelId, page, perPage, "", collapsedThreads, false)
	if err != nil {
		return err
	}
	if postList == nil || len(postList.Posts) == 0 {
		return nil
	}
	return ue.store.SetPosts(postsMapToSlice(postList.Posts))
}

// GetPostsBefore fetches and stores posts in a given channelId that were made before
// a given postId. It returns a list of posts ids.
func (ue *UserEntity) GetPostsBefore(channelId, postId string, page, perPage int, collapsedThreads bool) ([]string, error) {
	postList, _, err := ue.client.GetPostsBefore(context.Background(), channelId, postId, page, perPage, "", collapsedThreads, false)
	if err != nil {
		return nil, err
	}
	if postList == nil || len(postList.Posts) == 0 {
		return nil, nil
	}

	return postList.Order, ue.store.SetPosts(postListToSlice(postList))
}

// GetPostsAfter fetches and stores posts in a given channelId that were made after
// a given postId.
func (ue *UserEntity) GetPostsAfter(channelId, postId string, page, perPage int, collapsedThreads bool) error {
	postList, _, err := ue.client.GetPostsAfter(context.Background(), channelId, postId, page, perPage, "", collapsedThreads, false)
	if err != nil {
		return err
	}
	if postList == nil || len(postList.Posts) == 0 {
		return nil
	}
	return ue.store.SetPosts(postsMapToSlice(postList.Posts))
}

// GetPostsSince fetches and stores posts in a given channelId that were made
// since the given time. It returns a list of posts ids.
func (ue *UserEntity) GetPostsSince(channelId string, time int64, collapsedThreads bool) ([]string, error) {
	postList, _, err := ue.client.GetPostsSince(context.Background(), channelId, time, collapsedThreads)
	if err != nil {
		return nil, err
	}
	if postList == nil || len(postList.Posts) == 0 {
		return nil, nil
	}

	return postList.Order, ue.store.SetPosts(postListToSlice(postList))
}

// GetPinnedPosts fetches and returns pinned posts in a given channelId.
func (ue *UserEntity) GetPinnedPosts(channelId string) (*model.PostList, error) {
	postList, _, err := ue.client.GetPinnedPosts(context.Background(), channelId, "")
	if err != nil {
		return nil, err
	}
	return postList, nil
}

// GetPostsAroundLastUnread fetches and stores the posts made around last
// unread in a given channelId. It returns a list of posts ids.
func (ue *UserEntity) GetPostsAroundLastUnread(channelId string, limitBefore, limitAfter int, collapsedThreads bool) ([]string, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return nil, err
	}

	postList, _, err := ue.client.GetPostsAroundLastUnread(context.Background(), user.Id, channelId, limitBefore, limitAfter, collapsedThreads)
	if err != nil {
		return nil, err
	}
	if postList == nil || len(postList.Posts) == 0 {
		return nil, nil
	}

	return postList.Order, ue.store.SetPosts(postListToSlice(postList))
}

// CreateChannel creates and stores a new channel with the given information.
// It returns the channel's id.
func (ue *UserEntity) CreateChannel(channel *model.Channel) (string, error) {
	_, err := ue.getUserFromStore()
	if err != nil {
		return "", err
	}

	channel, _, err = ue.client.CreateChannel(context.Background(), channel)
	if err != nil {
		return "", err
	}

	err = ue.store.SetChannel(channel)
	if err != nil {
		return "", err
	}

	return channel.Id, nil
}

// CreateGroupChannel creates and stores a new group channel with the given
// members. It returns the channel's id.
func (ue *UserEntity) CreateGroupChannel(memberIds []string) (string, error) {
	channel, _, err := ue.client.CreateGroupChannel(context.Background(), memberIds)
	if err != nil {
		return "", err
	}

	err = ue.store.SetChannel(channel)
	if err != nil {
		return "", err
	}

	return channel.Id, nil
}

// CreateGroupChannel creates and stores a new direct channel with the given
// user. It returns the channel's id.
func (ue *UserEntity) CreateDirectChannel(otherUserId string) (string, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return "", err
	}

	channel, _, err := ue.client.CreateDirectChannel(context.Background(), user.Id, otherUserId)
	if err != nil {
		return "", err
	}

	err = ue.store.SetChannel(channel)
	if err != nil {
		return "", err
	}

	return channel.Id, nil
}

// RemoveUserFromChannel removes the specified user from the specified channel.
// It returns whether the user was successfully removed or not.
func (ue *UserEntity) RemoveUserFromChannel(channelId, userId string) error {
	_, err := ue.client.RemoveUserFromChannel(context.Background(), channelId, userId)
	if err != nil {
		return err
	}
	return ue.store.RemoveChannelMember(channelId, userId)
}

// AddChannelMember adds the specified user to the specified channel.
func (ue *UserEntity) AddChannelMember(channelId, userId string) error {
	member, _, err := ue.client.AddChannelMember(context.Background(), channelId, userId)
	if err != nil {
		return nil
	}

	return ue.store.SetChannelMember(channelId, member)
}

// GetChannel fetches and stores the specified channel.
func (ue *UserEntity) GetChannel(channelId string) error {
	channel, _, err := ue.client.GetChannel(context.Background(), channelId, "")
	if err != nil {
		return err
	}

	return ue.store.SetChannel(channel)
}

// GetChannelsForTeam fetches and stores channels in the specified team.
func (ue *UserEntity) GetChannelsForTeam(teamId string, includeDeleted bool) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}
	channels, _, err := ue.client.GetChannelsForTeamForUser(context.Background(), teamId, user.Id, includeDeleted, "")
	if err != nil {
		return err
	}

	return ue.store.SetChannels(channels)
}

// GetPublicChannelsForTeam fetches and stores public channels in the
// specified team.
func (ue *UserEntity) GetPublicChannelsForTeam(teamId string, page, perPage int) error {
	channels, _, err := ue.client.GetPublicChannelsForTeam(context.Background(), teamId, page, perPage, "")
	if err != nil {
		return err
	}
	return ue.store.SetChannels(channels)
}

// SearchChannelsForTeam performs a search for channels in the specified team.
// It returns channels that matches the search.
func (ue *UserEntity) SearchChannelsForTeam(teamId string, search *model.ChannelSearch) ([]*model.Channel, error) {
	channels, _, err := ue.client.SearchChannels(context.Background(), teamId, search)
	if err != nil {
		return nil, err
	}
	return channels, nil
}

// SearchChannels performs a search for channels in all teams for a user.
func (ue *UserEntity) SearchChannels(search *model.ChannelSearch) (model.ChannelListWithTeamData, error) {
	channels, _, err := ue.client.SearchAllChannelsForUser(context.Background(), search.Term)
	if err != nil {
		return nil, err
	}
	return channels, nil
}

// SearchGroupChannels performs a search for group channels.
// It returns channels whose members' usernames match the search term.
func (ue *UserEntity) SearchGroupChannels(search *model.ChannelSearch) ([]*model.Channel, error) {
	channels, _, err := ue.client.SearchGroupChannels(context.Background(), search)
	if err != nil {
		return nil, err
	}
	return channels, nil
}

// GetChannelsForTeamForUser fetches and stores chanels for the specified user in
// the specified team. It returns a list of those channels.
func (ue *UserEntity) GetChannelsForTeamForUser(teamId, userId string, includeDeleted bool) ([]*model.Channel, error) {
	channels, _, err := ue.client.GetChannelsForTeamForUser(context.Background(), teamId, userId, includeDeleted, "")
	if err != nil {
		return nil, err
	}

	if err := ue.store.SetChannels(channels); err != nil {
		return nil, err
	}

	return channels, nil
}

// GetChannelsForUser returns all channels from all teams for a given user.
func (ue *UserEntity) GetChannelsForUser(userID string) ([]*model.Channel, error) {
	channels, _, err := ue.client.GetChannelsForUserWithLastDeleteAt(context.Background(), userID, 0)
	if err != nil {
		return nil, err
	}

	if err := ue.store.SetChannels(channels); err != nil {
		return nil, err
	}

	return channels, nil
}

// ViewChannels performs a channel view for the user.
func (ue *UserEntity) ViewChannel(view *model.ChannelView) (*model.ChannelViewResponse, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return nil, err
	}

	channelViewResponse, _, err := ue.client.ViewChannel(context.Background(), user.Id, view)
	if err != nil {
		return nil, err
	}

	if err := ue.store.SetChannelView(view.ChannelId); err != nil {
		return nil, err
	}

	return channelViewResponse, nil
}

// GetChannelUnread fetches and returns information about the specified channel's unread
// messages.
func (ue *UserEntity) GetChannelUnread(channelId string) (*model.ChannelUnread, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return nil, err
	}

	channelUnreadResponse, _, err := ue.client.GetChannelUnread(context.Background(), channelId, user.Id)
	if err != nil {
		return nil, err
	}

	return channelUnreadResponse, nil
}

// GetChannelMembers fetches and stores channel members for the specified channel.
func (ue *UserEntity) GetChannelMembers(channelId string, page, perPage int) error {
	channelMembers, _, err := ue.client.GetChannelMembers(context.Background(), channelId, page, perPage, "")
	if err != nil {
		return err
	}

	return ue.store.SetChannelMembers(channelMembers)
}

// GetChannelMembersForUser gets the channel members for the specified user in
// the specified team.
func (ue *UserEntity) GetChannelMembersForUser(userId, teamId string) error {
	channelMembers, _, err := ue.client.GetChannelMembersForUser(context.Background(), userId, teamId, "")
	if err != nil {
		return err
	}

	return ue.store.SetChannelMembers(channelMembers)
}

// GetChannelMember fetches and stores the channel member for the specified user in
// the specified channel.
func (ue *UserEntity) GetChannelMember(channelId, userId string) error {
	cm, _, err := ue.client.GetChannelMember(context.Background(), channelId, userId, "")
	if err != nil {
		return err
	}

	return ue.store.SetChannelMember(channelId, cm)
}

// GetChannelStats fetches statistics for the specified channel.
func (ue *UserEntity) GetChannelStats(channelId string, excludeFileCount bool) error {
	stats, _, err := ue.client.GetChannelStats(context.Background(), channelId, "", excludeFileCount)
	if err != nil {
		return err
	}

	return ue.store.SetChannelStats(channelId, stats)
}

// AutocompleteChannelsForTeam fetches and stores an ordered list of channels for a given
// name in a specified team.
func (ue *UserEntity) AutocompleteChannelsForTeam(teamId, name string) error {
	channelList, _, err := ue.client.AutocompleteChannelsForTeam(context.Background(), teamId, name)
	if err != nil {
		return err
	}

	return ue.store.SetChannels(channelList)
}

// AutocompleteChannelsForTeamForSearch fetches and stores an ordered list of the
// user's channels autocomplete suggestions. It returns a map of found channel names.
func (ue *UserEntity) AutocompleteChannelsForTeamForSearch(teamId, name string) (map[string]bool, error) {
	channelList, _, err := ue.client.AutocompleteChannelsForTeamForSearch(context.Background(), teamId, name)
	if err != nil {
		return nil, err
	}

	if channelList == nil {
		return nil, errors.New("nil channel list")
	}
	channelsMap := make(map[string]bool, len(channelList))
	for _, u := range channelList {
		channelsMap[u.Name] = true
	}

	return channelsMap, ue.store.SetChannels(channelList)
}

// CreateTeam creates a new team with the given information.
func (ue *UserEntity) CreateTeam(team *model.Team) (string, error) {
	team, _, err := ue.client.CreateTeam(context.Background(), team)
	if err != nil {
		return "", err
	}

	return team.Id, nil
}

// GetTeam fetches and returns the specified team.
func (ue *UserEntity) GetTeam(teamId string) error {
	team, _, err := ue.client.GetTeam(context.Background(), teamId, "")
	if err != nil {
		return err
	}
	return ue.store.SetTeam(team)
}

// UpdateTeam updates and stores the given team.
func (ue *UserEntity) UpdateTeam(team *model.Team) error {
	team, _, err := ue.client.UpdateTeam(context.Background(), team)
	if err != nil {
		return err
	}
	return ue.store.SetTeam(team)
}

// GetTeamsForUser fetches and stores the teams for the specified user.
// It returns a list of team ids.
func (ue *UserEntity) GetTeamsForUser(userId string) ([]string, error) {
	teams, _, err := ue.client.GetTeamsForUser(context.Background(), userId, "")
	if err != nil {
		return nil, err
	}

	if err := ue.store.SetTeams(teams); err != nil {
		return nil, err
	}

	teamIds := make([]string, len(teams))
	for i, team := range teams {
		teamIds[i] = team.Id
	}

	return teamIds, nil
}

// AddTeamMember adds the specified user to the specified team.
func (ue *UserEntity) AddTeamMember(teamId, userId string) error {
	tm, _, err := ue.client.AddTeamMember(context.Background(), teamId, userId)
	if err != nil {
		return err
	}

	return ue.store.SetTeamMember(teamId, tm)
}

// RemoveTeamMember removes the specified user from the specified team.
func (ue *UserEntity) RemoveTeamMember(teamId, userId string) error {
	_, err := ue.client.RemoveTeamMember(context.Background(), teamId, userId)
	if err != nil {
		return err
	}

	return ue.store.RemoveTeamMember(teamId, userId)
}

// GetTeamMembers fetches and stores team members for the specified team.
func (ue *UserEntity) GetTeamMembers(teamId string, page, perPage int) error {
	members, _, err := ue.client.GetTeamMembers(context.Background(), teamId, page, perPage, "")
	if err != nil {
		return err
	}
	return ue.store.SetTeamMembers(teamId, members)
}

// GetTeamMember returns a team member based on the provided team and user id strings.
func (ue *UserEntity) GetTeamMember(teamId, userId string) error {
	member, _, err := ue.client.GetTeamMember(context.Background(), teamId, userId, "")
	if err != nil {
		return err
	}

	return ue.store.SetTeamMember(teamId, member)
}

// GetTeamMembersForUser fetches and stores team members for the specified user.
func (ue *UserEntity) GetTeamMembersForUser(userId string) error {
	members, _, err := ue.client.GetTeamMembersForUser(context.Background(), userId, "")
	if err != nil {
		return err
	}

	for _, m := range members {
		err := ue.store.SetTeamMember(m.TeamId, m)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetUsersByIds fetches and stores the specified users.
// It returns a list of those users' ids.
func (ue *UserEntity) GetUsersByIds(userIds []string) ([]string, error) {
	users, _, err := ue.client.GetUsersByIds(context.Background(), userIds)
	if err != nil {
		return nil, err
	}

	if err := ue.store.SetUsers(users); err != nil {
		return nil, err
	}

	newUserIds := make([]string, len(users))
	for i, user := range users {
		newUserIds[i] = user.Id
	}
	return newUserIds, nil
}

// GetUsersByUsername fetches and stores users for the given usernames.
// It returns a list of those users' ids.
func (ue *UserEntity) GetUsersByUsernames(usernames []string) ([]string, error) {
	users, _, err := ue.client.GetUsersByUsernames(context.Background(), usernames)
	if err != nil {
		return nil, err
	}

	if err := ue.store.SetUsers(users); err != nil {
		return nil, err
	}

	newUserIds := make([]string, len(users))
	for i, user := range users {
		newUserIds[i] = user.Id
	}
	return newUserIds, nil
}

// GetUserStatus fetches and stores the status for the user.
func (ue *UserEntity) GetUserStatus() error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}

	_, _, err = ue.client.GetUserStatus(context.Background(), user.Id, "")
	if err != nil {
		return err
	}

	return nil
}

// GetUsersStatusesByIds fetches and stores statuses for the specified users.
func (ue *UserEntity) GetUsersStatusesByIds(userIds []string) error {
	statusList, _, err := ue.client.GetUsersStatusesByIds(context.Background(), userIds)
	if err != nil {
		return err
	}

	for _, status := range statusList {
		if err := ue.store.SetStatus(status.UserId, status); err != nil {
			return err
		}
	}

	return nil
}

// GetUsersInChannel fetches and stores users in the specified channel.
func (ue *UserEntity) GetUsersInChannel(channelId string, page, perPage int) error {
	if len(channelId) == 0 {
		return errors.New("userentity: channelId should not be empty")
	}

	users, _, err := ue.client.GetUsersInChannel(context.Background(), channelId, page, perPage, "")
	if err != nil {
		return err
	}

	return ue.store.SetUsers(users)
}

// GetUsers fetches and stores all users. It returns a list of those users' ids.
// If perPage is more than the maxPageSize at the server, it will paginate
// through the list. In that case, it might fetch more than users asked since
// it will always get maxPageSize sized chunks.
func (ue *UserEntity) GetUsers(page, perPage int) ([]string, error) {
	userIds := make([]string, 0, perPage)

	// 200 is the hardcoded limit of the server.
	// It's exposed via the web package, but it's outside the contract
	// of the public module, so we hardcode here for simplicity.
	const maxPageSize = 200
	var remaining int
	if perPage > maxPageSize {
		remaining = perPage
		perPage = maxPageSize
	}

	for {
		users, _, err := ue.client.GetUsers(context.Background(), page, perPage, "")
		if err != nil {
			return nil, err
		}
		err = ue.store.SetUsers(users)
		if err != nil {
			return nil, err
		}
		for i := range users {
			userIds = append(userIds, users[i].Id)
		}

		if len(users) < remaining {
			page++
			remaining -= perPage
			continue
		}
		break
	}

	return userIds, nil
}

// GetUsersNotInChannel returns a list of user ids not in a given channel.
func (ue *UserEntity) GetUsersNotInChannel(teamId, channelId string, page, perPage int) ([]string, error) {
	users, _, err := ue.client.GetUsersNotInChannel(context.Background(), teamId, channelId, page, perPage, "")
	if err != nil {
		return nil, err
	}

	userIds := make([]string, len(users))
	for i := range users {
		userIds[i] = users[i].Id
	}

	return userIds, ue.store.SetUsers(users)
}

// GetTeamStats fetches statistics for the specified team.
func (ue *UserEntity) GetTeamStats(teamId string) error {
	_, _, err := ue.client.GetTeamStats(context.Background(), teamId, "")
	if err != nil {
		return err
	}

	return nil
}

// GetTeamsUnread fetches and returns information about unreads messages for
// the user in the teams it belongs to.
func (ue *UserEntity) GetTeamsUnread(teamIdToExclude string, includeCollapsedThreads bool) ([]*model.TeamUnread, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return nil, err
	}

	unread, _, err := ue.client.GetTeamsUnreadForUser(context.Background(), user.Id, teamIdToExclude, includeCollapsedThreads)
	if err != nil {
		return nil, err
	}

	return unread, nil
}

// UploadFile uploads the given data in the specified channel.
func (ue *UserEntity) UploadFile(data []byte, channelId, filename string) (*model.FileUploadResponse, error) {
	fresp, _, err := ue.client.UploadFile(context.Background(), data, channelId, filename)
	if err != nil {
		return nil, err
	}

	return fresp, nil
}

// GetFileInfosForPost returns file information for the specified post.
func (ue *UserEntity) GetFileInfosForPost(postId string) ([]*model.FileInfo, error) {
	infos, _, err := ue.client.GetFileInfosForPost(context.Background(), postId, "")
	if err != nil {
		return nil, err
	}
	return infos, nil
}

// GetFileThumbnail fetches the thumbnail for the specified file.
func (ue *UserEntity) GetFileThumbnail(fileId string) error {
	_, _, err := ue.client.GetFileThumbnail(context.Background(), fileId)
	if err != nil {
		return err
	}
	return nil
}

// GetFilePreview fetches the preview for the specified file.
func (ue *UserEntity) GetFilePreview(fileId string) error {
	_, _, err := ue.client.GetFilePreview(context.Background(), fileId)
	if err != nil {
		return err
	}

	return nil
}

// AddTeamMemberFromInvite adds a user to a team using the given token and
// inviteId.
func (ue *UserEntity) AddTeamMemberFromInvite(token, inviteId string) error {
	tm, _, err := ue.client.AddTeamMemberFromInvite(context.Background(), token, inviteId)
	if err != nil {
		return err
	}

	return ue.store.SetTeamMember(tm.TeamId, tm)
}

// SetProfileImage sets the profile image for the user.
func (ue *UserEntity) SetProfileImage(data []byte) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}
	_, err = ue.client.SetProfileImage(context.Background(), user.Id, data)
	if err != nil {
		return err
	}
	return nil
}

// GetProfileImageForUser fetches and stores the profile image for the user.
func (ue *UserEntity) GetProfileImageForUser(userId string, lastPictureUpdate int) error {
	_, resp, err := ue.client.GetProfileImage(context.Background(), userId, strconv.Itoa(lastPictureUpdate))
	if err != nil {
		return err
	}

	if resp.Etag == "" {
		lastPictureUpdate = 0
	} else {
		lastPictureUpdate, err = strconv.Atoi(resp.Etag)
		if err != nil {
			return fmt.Errorf("failed to parse response ETag as an integer: %q", resp.Etag)
		}
	}

	return ue.store.SetProfileImage(userId, lastPictureUpdate)
}

// SearchUsers performs a user search. It returns a list of users that matched.
func (ue *UserEntity) SearchUsers(search *model.UserSearch) ([]*model.User, error) {
	users, _, err := ue.client.SearchUsers(context.Background(), search)
	if err != nil {
		return nil, err
	}
	return users, nil
}

// AutocompleteUsersInChannel performs autocomplete of a username in a specified team and channel.
// It returns the users in the system based on the given username.
func (ue *UserEntity) AutocompleteUsersInChannel(teamId, channelId, username string, limit int) (map[string]bool, error) {
	users, _, err := ue.client.AutocompleteUsersInChannel(context.Background(), teamId, channelId, username, limit, "")
	if err != nil {
		return nil, err
	}
	if users == nil {
		return nil, errors.New("nil users")
	}
	usersMap := make(map[string]bool, len(users.Users)+len(users.OutOfChannel))
	for _, u := range users.Users {
		usersMap[u.Username] = true
	}
	for _, u := range users.OutOfChannel {
		usersMap[u.Username] = false
	}

	return usersMap, nil
}

// AutoCompleteUsersInTeam performs autocomplete of a username
// in a specified team.
// It returns the users in the system based on the given username.
func (ue *UserEntity) AutocompleteUsersInTeam(teamId, username string, limit int) (map[string]bool, error) {
	users, _, err := ue.client.AutocompleteUsersInTeam(context.Background(), teamId, username, limit, "")
	if err != nil {
		return nil, err
	}
	if users == nil {
		return nil, errors.New("nil users")
	}
	usersMap := make(map[string]bool, len(users.Users)+len(users.OutOfChannel))
	for _, u := range users.Users {
		usersMap[u.Username] = true
	}
	for _, u := range users.OutOfChannel {
		usersMap[u.Username] = false
	}

	return usersMap, nil
}

// GetEmojiList fetches and stores a list of custom emoji.
func (ue *UserEntity) GetEmojiList(page, perPage int) error {
	emojis, _, err := ue.client.GetEmojiList(context.Background(), page, perPage)
	if err != nil {
		return err
	}
	return ue.store.SetEmojis(emojis)
}

// GetEmojiImage fetches the image for a given emoji.
func (ue *UserEntity) GetEmojiImage(emojiId string) error {
	_, _, err := ue.client.GetEmojiImage(context.Background(), emojiId)
	if err != nil {
		return err
	}

	return nil
}

// SaveReaction stores the given reaction.
func (ue *UserEntity) SaveReaction(reaction *model.Reaction) error {
	r, _, err := ue.client.SaveReaction(context.Background(), reaction)
	if err != nil {
		return err
	}

	return ue.store.SetReaction(r)
}

// DeleteReaction deletes the given reaction.
func (ue *UserEntity) DeleteReaction(reaction *model.Reaction) error {
	_, err := ue.client.DeleteReaction(context.Background(), reaction)
	if err != nil {
		return err
	}

	if _, err := ue.store.DeleteReaction(reaction); err != nil {
		return err
	}

	return nil
}

// GetAllTeams returns all teams based on permissions.
// It returns a list of team ids.
func (ue *UserEntity) GetAllTeams(page, perPage int) ([]string, error) {
	teams, _, err := ue.client.GetAllTeams(context.Background(), "", page, perPage)
	if err != nil {
		return nil, err
	}

	if err := ue.store.SetTeams(teams); err != nil {
		return nil, err
	}

	teamIds := make([]string, len(teams))
	for i, team := range teams {
		teamIds[i] = team.Id
	}

	return teamIds, nil
}

// GetRolesByName fetches and stores roles for the given names.
// It returns a list of role ids.
func (ue *UserEntity) GetRolesByNames(roleNames []string) ([]string, error) {
	roles, _, err := ue.client.GetRolesByNames(context.Background(), roleNames)
	if err != nil {
		return nil, err
	}

	if err := ue.store.SetRoles(roles); err != nil {
		return nil, err
	}

	roleIds := make([]string, len(roles))
	for i, role := range roles {
		roleIds[i] = role.Id
	}
	return roleIds, nil
}

// GetWebappPlugins fetches webapp plugins.
func (ue *UserEntity) GetWebappPlugins() error {
	_, _, err := ue.client.GetWebappPlugins(context.Background())
	if err != nil {
		return err
	}

	return nil
}

// GetClientLicense fetched and stores the client license.
// It returns the client license in the old format.
func (ue *UserEntity) GetClientLicense() error {
	license, _, err := ue.client.GetOldClientLicense(context.Background(), "")
	if err != nil {
		return err
	}
	if err := ue.store.SetLicense(license); err != nil {
		return err
	}
	return nil
}

// SetCurrentTeam sets the given team as the current team for the user.
func (ue *UserEntity) SetCurrentTeam(team *model.Team) error {
	return ue.store.SetCurrentTeam(team)
}

// SetCurrentChannel sets the given channel as the current channel for the user.
func (ue *UserEntity) SetCurrentChannel(channel *model.Channel) error {
	return ue.store.SetCurrentChannel(channel)
}

// ClearUserData calls the Clear method on the underlying UserStore.
func (ue *UserEntity) ClearUserData() {
	ue.store.Clear()
}

// GetLogs fetches the server logs.
func (ue *UserEntity) GetLogs(page, perPage int) error {
	_, _, err := ue.client.GetLogs(context.Background(), page, perPage)
	if err != nil {
		return err
	}
	return nil
}

// GetAnalytics fetches the system analytics.
func (ue *UserEntity) GetAnalytics() error {
	_, _, err := ue.client.GetAnalyticsOld(context.Background(), "", "")
	if err != nil {
		return err
	}
	return nil
}

// GetClusterStatus fetches the cluster status.
func (ue *UserEntity) GetClusterStatus() error {
	_, _, err := ue.client.GetClusterStatus(context.Background())
	if err != nil {
		return err
	}

	return nil
}

// GetPluginStatuses fetches the plugin statuses.
func (ue *UserEntity) GetPluginStatuses() error {
	// Need to do it manually until MM-25405 is resolved.
	_, _, err := ue.client.GetPluginStatuses(context.Background())
	if err != nil {
		return err
	}

	return nil
}

// UpdateConfig updates the config with cfg.
func (ue *UserEntity) UpdateConfig(cfg *model.Config) error {
	cfg, _, err := ue.client.UpdateConfig(context.Background(), cfg)
	if err != nil {
		return err
	}
	ue.store.SetConfig(cfg)
	return nil
}

// MessageExport creates a job for a compliance message export
func (ue *UserEntity) MessageExport() error {
	messageExportJob := &model.Job{
		Type: "message_export",
	}

	_, _, err := ue.client.CreateJob(context.Background(), messageExportJob)
	if err != nil {
		return err
	}
	return nil
}

// GetPostsAfter fetches and stores posts in a given channelId that were made after
// a given postId.
func (ue *UserEntity) GetUserThreads(teamId string, options *model.GetUserThreadsOpts) ([]*model.ThreadResponse, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return nil, err
	}
	threads, _, err := ue.client.GetUserThreads(context.Background(), user.Id, teamId, *options)
	if err != nil {
		return nil, err
	}

	return threads.Threads, ue.store.SetThreads(threads.Threads)
}

// UpdateThreadFollow updates the follow state of the the given thread
func (ue *UserEntity) UpdateThreadFollow(teamId, threadId string, state bool) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}
	_, err = ue.client.UpdateThreadFollowForUser(context.Background(), user.Id, teamId, threadId, state)
	return err
}

// GetPostThread gets a post with all the other posts in the same thread.
func (ue *UserEntity) GetPostThreadWithOpts(threadId, etag string, opts model.GetPostsOptions) ([]string, bool, error) {
	postList, _, err := ue.client.GetPostThreadWithOpts(context.Background(), threadId, "", opts)
	if err != nil {
		return nil, false, err
	}
	if postList == nil || len(postList.Posts) == 0 {
		return nil, false, nil
	}
	return postList.Order, postList.HasNext, ue.store.SetPosts(postListToSlice(postList))
}

// MarkAllThreadsInTeamAsRead marks all threads in the given team as read
func (ue *UserEntity) MarkAllThreadsInTeamAsRead(teamId string) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}
	_, err = ue.client.UpdateThreadsReadForUser(context.Background(), user.Id, teamId)
	if err != nil {
		return err
	}

	// Keep threads in our local store in sync
	return ue.store.MarkAllThreadsInTeamAsRead(teamId)
}

func (ue *UserEntity) UpdateThreadRead(teamId, threadId string, timestamp int64) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}
	thread, _, err := ue.client.UpdateThreadReadForUser(context.Background(), user.Id, teamId, threadId, timestamp)
	if err != nil {
		return err
	}
	return ue.store.SetThreads([]*model.ThreadResponse{thread})
}

// GetSidebarCategories fetches and stores the sidebar categories for an user.
func (ue *UserEntity) GetSidebarCategories(userID, teamID string) error {
	categories, _, err := ue.client.GetSidebarCategoriesForTeamForUser(context.Background(), userID, teamID, "")
	if err != nil {
		return err
	}

	return ue.store.SetCategories(teamID, categories)
}

func (ue *UserEntity) CreateSidebarCategory(userID, teamID string, category *model.SidebarCategoryWithChannels) (*model.SidebarCategoryWithChannels, error) {
	cat, _, err := ue.client.CreateSidebarCategoryForTeamForUser(context.Background(), userID, teamID, category)
	if err != nil {
		return nil, err
	}

	// The client fetches and stores all categories again.
	if err := ue.GetSidebarCategories(userID, teamID); err != nil {
		return nil, err
	}
	return cat, nil
}

func (ue *UserEntity) UpdateSidebarCategory(userID, teamID string, categories []*model.SidebarCategoryWithChannels) error {
	_, _, err := ue.client.UpdateSidebarCategoriesForTeamForUser(context.Background(), userID, teamID, categories)
	if err != nil {
		return err
	}

	// The client fetches and stores all categories again.
	return ue.GetSidebarCategories(userID, teamID)
}

func (ue *UserEntity) UpdateCustomStatus(userID string, status *model.CustomStatus) error {
	_, _, err := ue.client.UpdateUserCustomStatus(context.Background(), userID, status)
	if err != nil {
		return err
	}
	return nil
}

func (ue *UserEntity) RemoveCustomStatus(userID string) error {
	_, err := ue.client.RemoveUserCustomStatus(context.Background(), userID)
	if err != nil {
		return err
	}
	return nil
}

func (ue *UserEntity) CreatePostReminder(userID, postID string, targetTime int64) error {
	_, err := ue.client.SetPostReminder(context.Background(), &model.PostReminder{
		TargetTime: targetTime,
		PostId:     postID,
		UserId:     userID,
	})
	if err != nil {
		return err
	}
	return nil
}

// AckToPost acknowledges a post.
func (ue *UserEntity) AckToPost(userID, postID string) error {
	_, _, err := ue.client.AcknowledgePost(context.Background(), postID, userID)
	return err
}

// GetInitialDataGQL is a method to get the initial use data via GraphQL.
func (ue *UserEntity) GetInitialDataGQL() error {
	var q struct {
		Config      map[string]string `json:"config"`
		User        gqlUser           `json:"user"`
		TeamMembers []gqlTeamMember   `json:"teamMembers"`
	}

	input := &user.GraphQLInput{
		OperationName: "gqlWebCurrentUserInfo",
		Query: `
			query gqlWebCurrentUserInfo($id: String = "me") {
				config
				user(id: $id) {
					id
					username
					email
					firstName
					lastName
					createAt
					updateAt
					deleteAt
					emailVerified
					isBot
					isGuest
					isSystemAdmin
					timezone
					props
					notifyProps
					roles {
						id
						name
						permissions
					}
					preferences {
						name
						user_id: userId
						category
						value
					}
				}
				teamMembers(userId: $id) {
					team {
						id
						display_name: displayName
						name
						create_at: createAt
						update_at: updateAt
						delete_at: deleteAt
						description
						email
						type
						company_name: companyName
						allowed_domains: allowedDomains
						invite_id: inviteId
						last_team_icon_update: lastTeamIconUpdate
						group_constrained: groupConstrained
						allow_open_invite: allowOpenInvite
						scheme_id: schemeId
						policy_id: policyId
					}
					delete_at: deleteAt
					scheme_guest: schemeGuest
					scheme_user: schemeUser
					scheme_admin: schemeAdmin
				}
			}
	`}

	gqlResp, err := ue.getGqlResponse(input)
	if err != nil {
		return err
	}

	err = json.Unmarshal(gqlResp.Data, &q)
	if err != nil {
		return err
	}

	// And writing them all back to the store.
	user, prefs, roles := convertToTypedUser(q.User)
	teams, tms := convertToTypedTeams(user.Id, q.TeamMembers)
	ue.store.SetPreferences(prefs)
	ue.store.SetUser(user)
	ue.store.SetRoles(roles)
	ue.store.SetClientConfig(q.Config)
	ue.store.SetTeams(teams)
	for _, tm := range tms {
		ue.store.SetTeamMember(tm.TeamId, tm)
	}
	return nil
}

// GetChannelsAndChannelMembersGQL is a method to get channels and channelMember info via GraphQL
func (ue *UserEntity) GetChannelsAndChannelMembersGQL(teamID string, includeDeleted bool, channelsCursor, channelMembersCursor string) (string, string, error) {
	var q struct {
		Channels       []gqlChannel       `json:"channels"`
		ChannelMembers []gqlChannelMember `json:"channelMembers"`
	}
	const perPage = 200

	input := &user.GraphQLInput{
		OperationName: "gqlWebChannelsAndChannelMembers",
		Query: `
			query gqlWebChannelsAndChannelMembers($teamId: String, $perPage: Int!, $channelsCursor: String, $channelMembersCursor: String, $includeDeleted: Boolean) {
				channels(userId: "me", teamId: $teamId, first: $perPage, after: $channelsCursor, includeDeleted: $includeDeleted) {
		            cursor
			        id
			        create_at: createAt
			        update_at: updateAt
			        delete_at: deleteAt
			        team {
			          id
			        }
			        type
			        display_name: displayName
			        name
			        header
			        purpose
			        last_post_at: lastPostAt
			        last_root_post_at: lastRootPostAt
			        total_msg_count: totalMsgCount
			        total_msg_count_root: totalMsgCountRoot
			        creator_id: creatorId
			        scheme_id: schemeId
			        group_constrained: groupConstrained
			        shared
			        props
			        policy_id: policyId
		        }
		        channelMembers(userId: "me", teamId: $teamId, first: $perPage, after: $channelMembersCursor) {
		            cursor
			        channel {
			            id
			        }
			        user {
			            id
			        }
			        roles {
			            id
			            name
			            permissions
			        }
			        last_viewed_at: lastViewedAt
			        msg_count: msgCount
			        msg_count_root: msgCountRoot
			        mention_count: mentionCount
			        mention_count_root: mentionCountRoot
			        urgent_mention_count: urgentMentionCount
			        notify_props: notifyProps
			        last_update_at: lastUpdateAt
			        scheme_admin: schemeAdmin
			        scheme_user: schemeUser
		        }
			}
	`,
		Variables: map[string]interface{}{
			"teamId":               teamID,
			"perPage":              perPage,
			"includeDeleted":       includeDeleted,
			"channelsCursor":       channelsCursor,
			"channelMembersCursor": channelMembersCursor,
		},
	}

	gqlResp, err := ue.getGqlResponse(input)
	if err != nil {
		return "", "", err
	}

	err = json.Unmarshal(gqlResp.Data, &q)
	if err != nil {
		return "", "", err
	}

	// And writing them all back to the store.
	channels, chCursor := convertToTypedChannels(q.Channels)
	cms, cmCursor := convertToTypedChannelMembers(q.ChannelMembers)

	if len(q.Channels) < perPage {
		chCursor = ""
	}

	if len(q.ChannelMembers) < perPage {
		cmCursor = ""
	}

	if err := ue.store.SetChannels(channels); err != nil {
		return "", "", err
	}
	if err := ue.store.SetChannelMembers(cms); err != nil {
		return "", "", err
	}

	return chCursor, cmCursor, nil
}

func (ue *UserEntity) prepareRequest(method, url string, data io.Reader, headers map[string]string) (*http.Request, error) {
	rq, err := http.NewRequest(method, url, data)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		rq.Header.Set(k, v)
	}

	rq.Header.Set(model.HeaderAuth, ue.client.AuthType+" "+ue.client.AuthToken)

	return rq, nil
}

func (ue *UserEntity) getGqlResponse(input any) (*graphql.Response, error) {
	buf, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	req, err := ue.prepareRequest(http.MethodPost,
		getGQLURL(ue.client.URL),
		bytes.NewReader(buf),
		map[string]string{})
	if err != nil {
		return nil, err
	}

	resp, err := ue.client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	var gqlResp *graphql.Response
	err = json.NewDecoder(resp.Body).Decode(&gqlResp)
	if err != nil {
		return nil, err
	}

	if len(gqlResp.Errors) > 0 {
		tmp := ""
		for _, err := range gqlResp.Errors {
			tmp += err.Error() + " "
		}
		return nil, errors.New(tmp)
	}

	return gqlResp, nil
}
