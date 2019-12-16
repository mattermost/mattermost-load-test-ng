// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package userentity

import (
	"errors"

	"github.com/mattermost/mattermost-server/v5/model"
)

func (ue *UserEntity) SignUp(email, username, password string) error {
	user := model.User{
		Email:    email,
		Username: username,
		Password: password,
	}

	newUser, resp := ue.client.CreateUser(&user)

	if resp.Error != nil {
		return resp.Error
	}

	newUser.Password = password
	return ue.store.SetUser(newUser)
}

func (ue *UserEntity) Login() error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}

	_, resp := ue.client.Login(user.Email, user.Password)
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (ue *UserEntity) Logout() (bool, error) {
	ok, resp := ue.client.Logout()
	if resp.Error != nil {
		return false, resp.Error
	}

	return ok, nil
}

func (ue *UserEntity) GetMe() (string, error) {
	user, resp := ue.client.GetMe("")
	if resp.Error != nil {
		return "", resp.Error
	}

	if err := ue.store.SetUser(user); err != nil {
		return "", err
	}

	return user.Id, nil
}

func (ue *UserEntity) GetPreferences() error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}

	preferences, resp := ue.client.GetPreferences(user.Id)
	if resp.Error != nil {
		return resp.Error
	}

	if err := ue.store.SetPreferences(&preferences); err != nil {
		return err
	}
	return nil
}

func (ue *UserEntity) CreateUser(user *model.User) (string, error) {
	user, resp := ue.client.CreateUser(user)
	if resp.Error != nil {
		return "", resp.Error
	}

	return user.Id, nil
}

func (ue *UserEntity) UpdateUser(user *model.User) error {
	user, resp := ue.client.UpdateUser(user)
	if resp.Error != nil {
		return resp.Error
	}

	if user.Id == ue.store.Id() {
		return ue.store.SetUser(user)
	}

	return nil
}

func (ue *UserEntity) PatchUser(userId string, patch *model.UserPatch) error {
	user, resp := ue.client.PatchUser(userId, patch)

	if resp.Error != nil {
		return resp.Error
	}

	if userId == ue.store.Id() {
		return ue.store.SetUser(user)
	}

	return nil
}

func (ue *UserEntity) CreatePost(post *model.Post) (string, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return "", err
	}

	post.PendingPostId = model.NewId()
	post.UserId = user.Id

	post, resp := ue.client.CreatePost(post)
	if resp.Error != nil {
		return "", resp.Error
	}

	err = ue.store.SetPost(post)

	return post.Id, err
}

func (ue *UserEntity) SearchPosts(teamId, terms string, isOrSearch bool) (*model.PostList, error) {
	postList, resp := ue.client.SearchPosts(teamId, terms, isOrSearch)
	if resp.Error != nil {
		return nil, resp.Error
	}
	return postList, nil
}

func (ue *UserEntity) GetPostsForChannel(channelId string, page, perPage int) error {
	postlist, resp := ue.client.GetPostsForChannel(channelId, page, perPage, "")
	if resp.Error != nil {
		return resp.Error
	}
	return ue.store.SetPosts(postsMapToSlice(postlist.Posts))
}

func (ue *UserEntity) GetPostsBefore(channelId, postId string, page, perPage int) error {
	postlist, resp := ue.client.GetPostsBefore(channelId, postId, page, perPage, "")
	if resp.Error != nil {
		return resp.Error
	}
	return ue.store.SetPosts(postsMapToSlice(postlist.Posts))
}

func (ue *UserEntity) GetPostsAfter(channelId, postId string, page, perPage int) error {
	postlist, resp := ue.client.GetPostsAfter(channelId, postId, page, perPage, "")
	if resp.Error != nil {
		return resp.Error
	}
	return ue.store.SetPosts(postsMapToSlice(postlist.Posts))
}

func (ue *UserEntity) UploadFile(data []byte, channelId, filename string) (*model.FileUploadResponse, error) {
	fresp, resp := ue.client.UploadFile(data, channelId, filename)
	if resp.Error != nil {
		return nil, resp.Error
	}

	return fresp, nil
}

func (ue *UserEntity) CreateChannel(channel *model.Channel) (string, error) {
	_, err := ue.getUserFromStore()
	if err != nil {
		return "", err
	}

	channel, resp := ue.client.CreateChannel(channel)
	if resp.Error != nil {
		return "", resp.Error
	}

	err = ue.store.SetChannel(channel)
	if err != nil {
		return "", err
	}

	return channel.Id, nil
}

func (ue *UserEntity) CreateGroupChannel(memberIds []string) (string, error) {
	channel, resp := ue.client.CreateGroupChannel(memberIds)
	if resp.Error != nil {
		return "", resp.Error
	}

	err := ue.store.SetChannel(channel)
	if err != nil {
		return "", err
	}

	return channel.Id, nil
}

func (ue *UserEntity) CreateDirectChannel(otherUserId string) (string, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return "", err
	}

	channel, resp := ue.client.CreateDirectChannel(user.Id, otherUserId)
	if resp.Error != nil {
		return "", resp.Error
	}

	err = ue.store.SetChannel(channel)
	if err != nil {
		return "", err
	}

	return channel.Id, nil
}
func (ue *UserEntity) RemoveUserFromChannel(channelId, userId string) (bool, error) {
	ok, resp := ue.client.RemoveUserFromChannel(channelId, userId)
	if resp.Error != nil {
		return false, resp.Error
	}
	return ok, ue.store.RemoveChannelMember(channelId, userId)
}

func (ue *UserEntity) AddChannelMember(channelId, userId string) error {
	member, resp := ue.client.AddChannelMember(channelId, userId)
	if resp.Error != nil {
		return nil
	}

	return ue.store.SetChannelMember(channelId, member)
}

func (ue *UserEntity) GetChannel(channelId string) error {
	channel, resp := ue.client.GetChannel(channelId, "")
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetChannel(channel)
}

func (ue *UserEntity) GetChannelsForTeam(teamId string) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}
	channels, resp := ue.client.GetChannelsForTeamForUser(teamId, user.Id, "")
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetChannels(channels)
}

func (ue *UserEntity) SearchChannels(teamId string, search *model.ChannelSearch) ([]*model.Channel, error) {
	channels, resp := ue.client.SearchChannels(teamId, search)
	if resp.Error != nil {
		return nil, resp.Error
	}
	return channels, nil
}

func (ue *UserEntity) GetChannelsForTeamForUser(teamId, userId string) ([]*model.Channel, error) {
	channels, resp := ue.client.GetChannelsForTeamForUser(teamId, userId, "")
	if resp.Error != nil {
		return nil, resp.Error
	}
	for _, ch := range channels {
		err := ue.store.SetChannel(ch)
		if err != nil {
			return nil, err
		}
	}
	return channels, nil
}

func (ue *UserEntity) ViewChannel(view *model.ChannelView) (*model.ChannelViewResponse, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return nil, err
	}

	channelViewResponse, resp := ue.client.ViewChannel(user.Id, view)
	if resp.Error != nil {
		return nil, resp.Error
	}

	return channelViewResponse, nil
}

func (ue *UserEntity) GetChannelUnread(channelId string) (*model.ChannelUnread, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return nil, err
	}

	channelUnreadResponse, resp := ue.client.GetChannelUnread(channelId, user.Id)
	if resp.Error != nil {
		return nil, resp.Error
	}

	return channelUnreadResponse, nil
}

func (ue *UserEntity) GetChannelMembers(channelId string, page, perPage int) error {
	channelMembers, resp := ue.client.GetChannelMembers(channelId, page, perPage, "")
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetChannelMembers(channelId, channelMembers)
}

func (ue *UserEntity) GetChannelMember(channelId, userId string) error {
	cm, resp := ue.client.GetChannelMember(channelId, userId, "")
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetChannelMember(channelId, cm)
}

func (ue *UserEntity) GetChannelStats(channelId string) error {
	_, resp := ue.client.GetChannelStats(channelId, "")
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (ue *UserEntity) CreateTeam(team *model.Team) (string, error) {
	team, resp := ue.client.CreateTeam(team)
	if resp.Error != nil {
		return "", resp.Error
	}

	return team.Id, nil
}

func (ue *UserEntity) GetTeam(teamId string) error {
	team, resp := ue.client.GetTeam(teamId, "")
	if resp.Error != nil {
		return resp.Error
	}
	return ue.store.SetTeam(team)
}

func (ue *UserEntity) AddTeamMember(teamId, userId string) error {
	tm, resp := ue.client.AddTeamMember(teamId, userId)
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetTeamMember(teamId, tm)
}

func (ue *UserEntity) RemoveTeamMember(teamId, userId string) error {
	_, resp := ue.client.RemoveTeamMember(teamId, userId)
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.RemoveTeamMember(teamId, userId)
}

func (ue *UserEntity) GetTeamMembers(teamId string, page, perPage int) error {
	members, resp := ue.client.GetTeamMembers(teamId, page, perPage, "")
	if resp.Error != nil {
		return resp.Error
	}
	return ue.store.SetTeamMembers(teamId, members)
}

func (ue *UserEntity) GetUsersByIds(userIds []string) ([]string, error) {
	users, resp := ue.client.GetUsersByIds(userIds)
	if resp.Error != nil {
		return nil, resp.Error
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

func (ue *UserEntity) GetUsersByUsernames(usernames []string) ([]string, error) {
	users, resp := ue.client.GetUsersByUsernames(usernames)
	if resp.Error != nil {
		return nil, resp.Error
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

func (ue *UserEntity) GetUsersStatusesByIds(userIds []string) error {
	_, resp := ue.client.GetUsersStatusesByIds(userIds)
	return resp.Error
}

func (ue *UserEntity) GetTeamStats(teamId string) error {
	_, resp := ue.client.GetTeamStats(teamId, "")
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (ue *UserEntity) GetTeamsUnread(teamIdToExclude string) ([]*model.TeamUnread, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return nil, err
	}

	unread, resp := ue.client.GetTeamsUnreadForUser(user.Id, teamIdToExclude)
	if resp.Error != nil {
		return nil, resp.Error
	}

	return unread, nil
}

func (ue *UserEntity) GetFileInfosForPost(postId string) ([]*model.FileInfo, error) {
	infos, resp := ue.client.GetFileInfosForPost(postId, "")
	if resp.Error != nil {
		return nil, resp.Error
	}
	return infos, nil
}

func (ue *UserEntity) GetFileThumbnail(fileId string) ([]byte, error) {
	data, resp := ue.client.GetFileThumbnail(fileId)
	if resp.Error != nil {
		return nil, resp.Error
	}
	return data, nil
}

func (ue *UserEntity) AddTeamMemberFromInvite(token, inviteId string) error {
	tm, resp := ue.client.AddTeamMemberFromInvite(token, inviteId)
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetTeamMember(tm.TeamId, tm)
}

func (ue *UserEntity) SetProfileImage(data []byte) error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}
	ok, resp := ue.client.SetProfileImage(user.Id, data)
	if resp.Error != nil {
		return resp.Error
	}
	if !ok {
		return errors.New("cannot set profile image")
	}
	return nil
}

func (ue *UserEntity) GetProfileImage() error {
	user, err := ue.getUserFromStore()
	if err != nil {
		return err
	}
	return ue.GetProfileImageForUser(user.Id)
}

func (ue *UserEntity) GetProfileImageForUser(userId string) error {
	_, resp := ue.client.GetProfileImage(userId, "")
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (ue *UserEntity) SearchUsers(search *model.UserSearch) ([]*model.User, error) {
	users, resp := ue.client.SearchUsers(search)
	if resp.Error != nil {
		return nil, resp.Error
	}
	return users, nil
}

func (ue *UserEntity) GetEmojiList(page, perPage int) error {
	emojis, resp := ue.client.GetEmojiList(page, perPage)
	if resp.Error != nil {
		return resp.Error
	}
	return ue.store.SetEmojis(emojis)
}

func (ue *UserEntity) GetEmojiImage(emojiId string) error {
	_, resp := ue.client.GetEmojiImage(emojiId)
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (ue *UserEntity) GetReactions(postId string) error {
	reactions, resp := ue.client.GetReactions(postId)
	if resp.Error != nil {
		return resp.Error
	}

	return ue.store.SetReactions(postId, reactions)
}

func (ue *UserEntity) SaveReaction(reaction *model.Reaction) error {
	_, resp := ue.client.SaveReaction(reaction)
	if resp.Error != nil {
		return resp.Error
	}

	return nil
}

func (ue *UserEntity) GetTeams() ([]string, error) {
	user, err := ue.getUserFromStore()
	if err != nil {
		return nil, err
	}

	teams, resp := ue.client.GetTeamsForUser(user.Id, "")
	if resp.Error != nil {
		return nil, resp.Error
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
