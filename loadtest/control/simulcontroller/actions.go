// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"

	"github.com/mattermost/mattermost-server/v5/model"
)

type userAction struct {
	run       control.UserAction
	frequency float64
	// Minimum supported server version
	minServerVersion string
}

func (c *SimulController) connect() error {
	if !atomic.CompareAndSwapInt32(&c.connectedFlag, 0, 1) {
		return errors.New("already connected")
	}
	errChan, err := c.user.Connect()
	if err != nil {
		atomic.StoreInt32(&c.connectedFlag, 0)
		return fmt.Errorf("connect failed %w", err)
	}
	c.wg.Add(3)
	go func() {
		defer c.wg.Done()
		for err := range errChan {
			c.status <- c.newErrorStatus(err)
		}
	}()
	go c.wsEventHandler(c.wg)
	go c.periodicActions(c.wg)
	return nil
}

func (c *SimulController) disconnect() error {
	if !atomic.CompareAndSwapInt32(&c.connectedFlag, 1, 0) {
		return errors.New("not connected")
	}

	err := c.user.Disconnect()
	if err != nil {
		return fmt.Errorf("disconnect failed %w", err)
	}

	c.disconnectChan <- struct{}{}
	c.wg.Wait()

	return nil
}

func (c *SimulController) reload(full bool) control.UserActionResponse {
	if full {
		if err := c.disconnect(); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		c.user.ClearUserData()
		if err := c.connect(); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	resp := control.Reload(c.user)
	if resp.Err != nil {
		return resp
	}

	c.status <- c.newInfoStatus(resp.Info)

	team, err := c.user.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		// If the current team is not set we switch to a random one.
		return c.switchTeam(c.user)
	}

	if resp := loadTeam(c.user, team); resp.Err != nil {
		return resp
	}

	channel, err := c.user.Store().CurrentChannel()
	if errors.Is(err, memstore.ErrChannelNotFound) {
		// If the current channel is not set we switch to a random one.
		return switchChannel(c.user)
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return viewChannel(c.user, channel)
}

func (c *SimulController) fullReload(u user.User) control.UserActionResponse {
	return c.reload(true)
}

func (c *SimulController) loginOrSignUp(u user.User) control.UserActionResponse {
	resp := c.login(u)
	if resp.Err != nil {
		if resp = control.SignUp(u); resp.Err != nil {
			return resp
		}
		c.status <- c.newInfoStatus(resp.Info)
		return c.login(u)
	}
	return resp
}

func (c *SimulController) login(u user.User) control.UserActionResponse {
	for {
		resp := control.Login(u)
		if resp.Err == nil {
			err := c.connect()
			if err == nil {
				return resp
			}
			c.status <- c.newErrorStatus(err)
		}

		errId := resp.Err.(*control.UserError).Err.(*model.AppError).Id
		if errId == "api.user.login.invalid_credentials_email_username" {
			return resp
		}

		c.status <- c.newErrorStatus(resp.Err)

		select {
		case <-c.stopChan:
			return control.UserActionResponse{Info: "login canceled"}
		case <-time.After(control.PickIdleTimeMs(c.config.MinIdleTimeMs, c.config.AvgIdleTimeMs, 1.0)):
		}
	}
}

func (c *SimulController) logout() control.UserActionResponse {
	err := c.disconnect()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	ok, err := c.user.Logout()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if !ok {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("user did not logout"))}
	}
	return control.UserActionResponse{Info: "logged out"}
}

func (c *SimulController) logoutLogin(u user.User) control.UserActionResponse {
	// logout
	if resp := c.logout(); resp.Err != nil {
		c.status <- c.newErrorStatus(resp.Err)
	} else {
		c.status <- c.newInfoStatus(resp.Info)
	}

	u.ClearUserData()

	// login
	if resp := c.login(c.user); resp.Err != nil {
		c.status <- c.newErrorStatus(resp.Err)
	} else {
		c.status <- c.newInfoStatus(resp.Info)
	}

	// reload
	return c.reload(false)
}

func (c *SimulController) joinTeam(u user.User) control.UserActionResponse {
	userStore := u.Store()
	userId := userStore.Id()

	if _, err := u.GetAllTeams(0, 100); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	team, err := u.Store().RandomTeam(store.SelectNotMemberOf)
	if errors.Is(err, memstore.ErrTeamStoreEmpty) {
		c.status <- c.newInfoStatus("no team to join")
		return c.switchTeam(u)
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.AddTeamMember(team.Id, userId); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	c.status <- c.newInfoStatus(fmt.Sprintf("joined team %s", team.Id))

	return c.switchTeam(u)
}

func loadTeam(u user.User, team *model.Team) control.UserActionResponse {
	if _, err := u.GetChannelsForTeamForUser(team.Id, u.Store().Id(), true); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.GetChannelMembersForUser(u.Store().Id(), team.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if _, err := u.GetTeamsUnread(""); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// TODO: use more realistic data.
	var userIds []string
	userIds = append(userIds, u.Store().Id())
	if err := u.GetUsersStatusesByIds(userIds); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("loaded team %s", team.Id)}
}

func (c *SimulController) switchTeam(u user.User) control.UserActionResponse {
	team, err := u.Store().RandomTeam(store.SelectMemberOf | store.SelectNotCurrent)
	if errors.Is(err, memstore.ErrTeamStoreEmpty) {
		return control.UserActionResponse{Info: "no other team to switch to"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	c.status <- c.newInfoStatus(fmt.Sprintf("switched to team %s", team.Id))

	if resp := loadTeam(u, &team); resp.Err != nil {
		return resp
	}

	if err := u.SetCurrentTeam(&team); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// We should probably keep track of the last channel viewed in the team but
	// for now we can simplify and randomly pick one each time.

	return switchChannel(u)
}

func (c *SimulController) joinChannel(u user.User) control.UserActionResponse {
	team, err := u.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("current team should be set"))}
	}

	if err := u.GetPublicChannelsForTeam(team.Id, 0, 100); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channel, err := u.Store().RandomChannel(team.Id, store.SelectNotMemberOf)
	if errors.Is(err, memstore.ErrChannelStoreEmpty) {
		return control.UserActionResponse{Info: "no channel to join"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.AddChannelMember(channel.Id, u.Store().Id()); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("joined channel %s", channel.Id)}
}

// fetchPostsInfo fetches additional information for the given posts ids like
// statuses and profile pictures of the posters and thumbnails for file
// attachments.
func fetchPostsInfo(u user.User, postsIds []string) error {
	// We loop through the fetched posts to gather the ids for the users who made
	// those posts. These are later needed to fetch profile images.
	// We also check if posts have any image attachments and if so we fetch the
	// respective thumbnails.
	var missingUsers []string
	var missingStatuses []string
	var missingPictures []string
	var missingUsernames []string

	// used for deduplication
	users := map[string]bool{}
	statuses := map[string]bool{}
	pictures := map[string]bool{}
	mentions := map[string]bool{}

	for _, postId := range postsIds {
		post, err := u.Store().Post(postId)
		if errors.Is(err, memstore.ErrPostNotFound) {
			continue
		} else if err != nil {
			return err
		}
		if username := extractMentionFromMessage(post.Message); username != "" && !mentions[username] {
			missingUsernames = append(missingUsernames, username)
			mentions[username] = true
		}

		var fileInfo []*model.FileInfo
		if post.Metadata != nil {
			fileInfo = post.Metadata.Files
		}
		for _, info := range fileInfo {
			if info.Extension != "png" && info.Extension != "jpg" {
				continue
			}
			if err := u.GetFileThumbnail(info.Id); err != nil {
				return err
			}
			if info.HasPreviewImage {
				if err := u.GetFilePreview(info.Id); err != nil {
					return err
				}
			}
		}

		userId := post.UserId

		if !pictures[userId] {
			missingPictures = append(missingPictures, userId)
			pictures[userId] = true
		}

		if status, err := u.Store().Status(userId); err != nil {
			return err
		} else if status.UserId == "" && !statuses[userId] {
			missingStatuses = append(missingStatuses, userId)
			statuses[userId] = true
		}

		if user, err := u.Store().GetUser(userId); err != nil {
			return err
		} else if user.Id == "" && !users[userId] {
			missingUsers = append(missingUsers, userId)
			users[userId] = true
		}
	}

	if len(missingStatuses) > 0 {
		if err := u.GetUsersStatusesByIds(missingStatuses); err != nil {
			return err
		}
	}

	if len(missingUsers) > 0 {
		if _, err := u.GetUsersByIds(missingUsers); err != nil {
			return err
		}
	}

	if len(missingPictures) > 0 {
		if err := getProfileImageForUsers(u, missingPictures); err != nil {
			return err
		}
	}

	if len(missingUsernames) > 0 {
		if _, err := u.GetUsersByUsernames(missingUsernames); err != nil {
			return err
		}
	}

	return nil
}

func viewChannel(u user.User, channel *model.Channel) control.UserActionResponse {
	collapsedThreads, resp := control.CollapsedThreadsEnabled(u)
	if resp.Err != nil {
		return resp
	}

	var currentChanId string
	if current, err := u.Store().CurrentChannel(); err == nil {
		currentChanId = current.Id
		// Somehow the webapp does a view to the current channel before switching.
		if _, err := u.ViewChannel(&model.ChannelView{ChannelId: current.Id}); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	} else if !errors.Is(err, memstore.ErrChannelNotFound) {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	var postsIds []string
	if view, err := u.Store().ChannelView(channel.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if view == 0 {
		postsIds, err = u.GetPostsAroundLastUnread(channel.Id, 30, 30, collapsedThreads)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	} else {
		postsIds, err = u.GetPostsSince(channel.Id, view, collapsedThreads)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	if err := fetchPostsInfo(u, postsIds); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.GetChannelStats(channel.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if channel.Type == model.CHANNEL_DIRECT || channel.Type == model.CHANNEL_GROUP {
		category := map[string]string{
			model.CHANNEL_DIRECT: model.PREFERENCE_CATEGORY_DIRECT_CHANNEL_SHOW,
			model.CHANNEL_GROUP:  "group_channel_show",
		}

		// We need to update the user's preferences so that
		// on next reload we can properly fetch opened DMs.
		pref := &model.Preferences{
			model.Preference{
				UserId:   u.Store().Id(),
				Category: category[channel.Type],
				Name:     channel.Id,
				Value:    "true",
			},
			model.Preference{
				UserId:   u.Store().Id(),
				Category: "channel_open_time", // This is a client defined constant.
				Name:     channel.Id,
				Value:    time.Now().Format(time.RFC3339),
			},
		}

		if err := u.UpdatePreferences(pref); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	if _, err := u.ViewChannel(&model.ChannelView{ChannelId: channel.Id, PrevChannelId: currentChanId}); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("viewed channel %s", channel.Id)}
}

func switchChannel(u user.User) control.UserActionResponse {
	team, err := u.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("current team should be set"))}
	}

	channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf|store.SelectNotCurrent|store.SelectNotDirect|store.SelectNotGroup)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if resp := viewChannel(u, &channel); resp.Err != nil {
		return control.UserActionResponse{Err: control.NewUserError(resp.Err)}
	}

	if err := u.SetCurrentChannel(&channel); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("switched to channel %s", channel.Id)}
}

func (c *SimulController) getUsersStatuses() control.UserActionResponse {
	channel, err := c.user.Store().CurrentChannel()
	if errors.Is(err, memstore.ErrChannelNotFound) {
		return control.UserActionResponse{Info: "getUsersStatuses: current channel not set"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	posts, err := c.user.Store().ChannelPostsSorted(channel.Id, false)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	// This comes from webapp. It should simulate how many posts the user can
	// actually see without scrolling.
	postVisibility := 60
	if len(posts) > postVisibility {
		posts = posts[:postVisibility]
	}

	currentId := c.user.Store().Id()
	statuses := make(map[string]bool)
	userIds := []string{currentId}
	for _, post := range posts {
		if post.UserId != "" && post.UserId != currentId && !statuses[post.UserId] {
			statuses[post.UserId] = true
			userIds = append(userIds, post.UserId)
		}
	}

	prefs, err := c.user.Store().Preferences()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	for _, p := range prefs {
		switch {
		case p.Category == model.PREFERENCE_CATEGORY_DIRECT_CHANNEL_SHOW:
			userIds = append(userIds, p.Name)
		}
	}

	if err := c.user.GetUsersStatusesByIds(userIds); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: "got statuses"}
}

func editPost(u user.User) control.UserActionResponse {
	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	post, err := u.Store().RandomPostForChannelByUser(channel.Id, u.Store().Id())
	if errors.Is(err, memstore.ErrPostNotFound) {
		return control.UserActionResponse{Info: "no posts to edit"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	isReply := post.RootId != ""
	message, err := createMessage(u, channel, isReply)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	postId, err := u.PatchPost(post.Id, &model.PostPatch{
		Message: &message,
	})
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("post edited, id %v", postId)}
}

func (c *SimulController) createPostReply(u user.User) control.UserActionResponse {
	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	post, err := u.Store().RandomPostForChannel(channel.Id)
	if errors.Is(err, memstore.ErrPostNotFound) {
		return control.UserActionResponse{Info: fmt.Sprintf("no posts found in channel %v", channel.Id)}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	var rootId string
	if post.RootId != "" {
		rootId = post.RootId
	} else {
		rootId = post.Id
	}

	if err := sendTypingEventIfEnabled(u, channel.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	message, err := createMessage(u, channel, true)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	reply := &model.Post{
		Message:   message,
		ChannelId: channel.Id,
		CreateAt:  time.Now().Unix() * 1000,
		RootId:    rootId,
	}

	// 2% of the times post will have files attached.
	if rand.Float64() < 0.02 {
		if err := c.attachFilesToPost(u, reply); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	replyId, err := u.CreatePost(reply)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("post reply created, id %v", replyId)}
}

func (c *SimulController) createPost(u user.User) control.UserActionResponse {
	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := sendTypingEventIfEnabled(u, channel.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	message, err := createMessage(u, channel, false)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	post := &model.Post{
		Message:   message,
		ChannelId: channel.Id,
		CreateAt:  time.Now().Unix() * 1000,
	}

	// 2% of the times post will have files attached.
	if rand.Float64() < 0.02 {
		if err := c.attachFilesToPost(u, post); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	postId, err := u.CreatePost(post)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("post created, id %v", postId)}
}

func (c *SimulController) attachFilesToPost(u user.User, post *model.Post) error {
	type file struct {
		data   []byte
		upload bool
	}
	filenames := []string{"test_upload.png", "test_upload.jpg", "test_upload.mp4"}
	files := make(map[string]*file, len(filenames))

	for _, filename := range filenames {
		files[filename] = &file{
			data:   control.MustAsset(filename),
			upload: rand.Intn(2) == 0,
		}
	}

	// We make sure at least one file gets uploaded.
	files[filenames[rand.Intn(len(filenames))]].upload = true

	var wg sync.WaitGroup
	fileIds := make(chan string, len(files))
	for filename, file := range files {
		if !file.upload {
			continue
		}
		wg.Add(1)
		go func(filename string, data []byte) {
			defer wg.Done()
			resp, err := u.UploadFile(data, post.ChannelId, filename)
			if err != nil {
				c.status <- c.newErrorStatus(err)
				return
			}
			c.status <- c.newInfoStatus(fmt.Sprintf("file uploaded, id %v", resp.FileInfos[0].Id))
			fileIds <- resp.FileInfos[0].Id
		}(filename, file.data)
	}

	wg.Wait()
	numFiles := len(fileIds)
	for i := 0; i < numFiles; i++ {
		post.FileIds = append(post.FileIds, <-fileIds)
	}

	return nil
}

func (c *SimulController) addReaction(u user.User) control.UserActionResponse {
	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	post, err := u.Store().RandomPostForChannel(channel.Id)
	if errors.Is(err, memstore.ErrPostNotFound) {
		return control.UserActionResponse{Info: fmt.Sprintf("no posts found in channel %v", channel.Id)}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	reaction := &model.Reaction{
		UserId: u.Store().Id(),
		PostId: post.Id,
	}

	emojis := []string{"+1", "tada", "point_up", "raised_hands"}
	reaction.EmojiName = emojis[rand.Intn(len(emojis))]

	reactions, err := u.Store().Reactions(post.Id)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	for i := 0; i < len(reactions); i++ {
		if reaction.UserId == reactions[i].UserId &&
			reaction.EmojiName == reactions[i].EmojiName {
			return control.UserActionResponse{Info: "reaction already added"}
		}
	}

	if u.SaveReaction(reaction); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("added reaction to post %s", post.Id)}
}

func (c *SimulController) createDirectChannel(u user.User) control.UserActionResponse {
	// Here we make a call to GetUsers to simulate the user opening the users
	// list when creating a direct channel.
	userIds, err := u.GetUsers(0, 100)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := getProfileImageForUsers(u, userIds); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// TODO: make the selection a bit smarter and pick someone
	// we don't have a direct channel with already.
	user, err := u.Store().RandomUser()
	if errors.Is(err, memstore.ErrLenMismatch) {
		return control.UserActionResponse{Info: "not enough users to create direct channel"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channelId, err := u.CreateDirectChannel(user.Id)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.GetChannel(channelId); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.GetChannelMember(channelId, u.Store().Id()); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channel, err := u.Store().Channel(channelId)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if resp := viewChannel(u, channel); resp.Err != nil {
		return control.UserActionResponse{Err: control.NewUserError(resp.Err)}
	}

	c.status <- c.newInfoStatus(fmt.Sprintf("direct channel created, id %s", channelId))

	return c.createPost(u)
}

func (c *SimulController) createGroupChannel(u user.User) control.UserActionResponse {
	// Here we make a call to GetUsers to simulate the user opening the users
	// list when creating a group channel.
	userIds, err := u.GetUsers(0, 100)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := getProfileImageForUsers(u, userIds); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// TODO: consider making this number range between an interval.
	numUsers := 2
	users, err := u.Store().RandomUsers(numUsers)
	if errors.Is(err, memstore.ErrLenMismatch) {
		return control.UserActionResponse{Info: "not enough users to create group channel"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// TODO: this transformation should be done at the store layer
	// by providing something like RandomUsersIds().
	userIds = make([]string, numUsers)
	for i := range users {
		userIds[i] = users[i].Id
	}

	channelId, err := u.CreateGroupChannel(userIds)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channel, err := u.Store().Channel(channelId)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if resp := viewChannel(u, channel); resp.Err != nil {
		return control.UserActionResponse{Err: control.NewUserError(resp.Err)}
	}

	c.status <- c.newInfoStatus(fmt.Sprintf("group channel created, id %s with users %+v", channelId, userIds))

	return c.createPost(u)
}

func openDirectOrGroupChannel(u user.User) control.UserActionResponse {
	team, err := u.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("current team should be set"))}
	}

	channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf|store.SelectNotCurrent|store.SelectNotPublic|store.SelectNotPrivate)
	if errors.Is(err, memstore.ErrChannelStoreEmpty) {
		return control.UserActionResponse{Info: "no channels to open"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if resp := viewChannel(u, &channel); resp.Err != nil {
		return control.UserActionResponse{Err: control.NewUserError(resp.Err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("opened direct/group channel %s", channel.Id)}
}

func getProfileImageForUsers(u user.User, userIds []string) error {
	for _, userId := range userIds {
		ok, err := u.Store().ProfileImage(userId)
		if err != nil {
			return err
		}
		if ok {
			continue
		}
		if err := u.GetProfileImageForUser(userId); err != nil {
			return err
		}
	}
	return nil
}

func createMessage(u user.User, channel *model.Channel, isReply bool) (string, error) {
	var message string
	// 10% of messages will contain a mention.
	if rand.Float64() < 0.10 {
		user, err := u.Store().RandomUser()
		if err != nil {
			return "", err
		}
		if err := emulateMention(channel.TeamId, channel.Id, user.Username, u.AutocompleteUsersInChannel); err != nil && !errors.Is(err, errNoMatch) {
			return "", err
		}
		message = "@" + user.Username + " "
	}
	message += genMessage(isReply)
	return message, nil
}

// This action includes methods that are called by the webapp client when a user
// unfocuses (switches browser's tab/window) and goes back to the app after some time.
func unreadCheck(u user.User) control.UserActionResponse {
	team, err := u.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("current team should be set"))}
	}

	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if _, err := u.GetChannelsForTeamForUser(team.Id, u.Store().Id(), true); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.GetChannelMembersForUser(u.Store().Id(), team.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if _, err := u.ViewChannel(&model.ChannelView{ChannelId: channel.Id}); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: "unread check done"}
}

func searchChannels(u user.User) control.UserActionResponse {
	team, err := u.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("current team should be set"))}
	}

	channel, err := u.Store().RandomChannel(team.Id, store.SelectAny)
	if errors.Is(err, memstore.ErrChannelStoreEmpty) {
		return control.UserActionResponse{Info: "no channel to search"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// numChars simulates how many characters does a user type
	// to search for a channel. This is an arbitrary value which fits well with the current
	// frequency value for this action.
	numChars := 4
	if numChars > len(channel.Name) {
		// rand.Intn returns a number exclusive of the max limit.
		// So there's no need to subtract 1.
		numChars = len(channel.Name)
	}

	return control.EmulateUserTyping(channel.Name[:1+rand.Intn(numChars)], func(term string) control.UserActionResponse {
		channels, err := u.SearchChannels(team.Id, &model.ChannelSearch{
			Term: term,
		})
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		return control.UserActionResponse{Info: fmt.Sprintf("found %d channels", len(channels))}
	})
}

func searchPosts(u user.User) control.UserActionResponse {
	team, err := u.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("current team should be set"))}
	}

	var words []string
	var opts control.PostsSearchOpts
	// This is an arbitrary limit on the number of words to search for.
	// TODO: possibly use user analytics data to improve this.
	count := 1 + rand.Intn(4)

	// TODO: back the probability of these choices with real data.
	if rand.Float64() < 0.2 {
		user, err := u.Store().RandomUser()
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		opts.From = user.Username
		control.EmulateUserTyping(opts.From, func(term string) control.UserActionResponse {
			users, err := u.AutocompleteUsersInTeam(team.Id, term, 25)
			if err != nil {
				return control.UserActionResponse{Err: control.NewUserError(err)}
			}
			if len(users) == 1 {
				return control.UserActionResponse{Err: errors.New("found")}
			}
			return control.UserActionResponse{}
		})
	}

	if rand.Float64() < 0.2 {
		channel, err := u.Store().RandomChannel(team.Id, store.SelectMemberOf|store.SelectNotDirect|store.SelectNotGroup)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		opts.In = channel.Name
		control.EmulateUserTyping(opts.In, func(term string) control.UserActionResponse {
			channels, err := u.AutocompleteChannelsForTeamForSearch(team.Id, term)
			if err != nil {
				return control.UserActionResponse{Err: control.NewUserError(err)}
			}
			if len(channels) == 1 {
				return control.UserActionResponse{Err: errors.New("found")}
			}
			return control.UserActionResponse{}
		})
	}

	if rand.Float64() < 0.2 {
		// We limit the search to 7 days.
		t := time.Now().Add(-time.Duration(rand.Intn(7)) * time.Hour * 24)
		switch rand.Intn(3) {
		case 0:
			opts.On = t
		case 1:
			opts.Before = t
		case 2:
			opts.After = t
		}
	}

	if rand.Float64() < 0.2 {
		opts.Excluded = []string{control.PickRandomWord()}
	}

	if rand.Float64() < 0.2 {
		opts.IsPhrase = true
	}

	for i := 0; i < count; i++ {
		words = append(words, control.PickRandomWord())
	}

	term := control.GeneratePostsSearchTerm(words, opts)
	list, err := u.SearchPosts(team.Id, term, false)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("found %d posts", len(list.Posts))}
}

func searchUsers(u user.User) control.UserActionResponse {
	user, err := u.Store().RandomUser()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.EmulateUserTyping(user.Username, func(term string) control.UserActionResponse {
		users, err := u.SearchUsers(&model.UserSearch{
			Term:  term,
			Limit: 100,
		})
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		return control.UserActionResponse{Info: fmt.Sprintf("found %d users", len(users))}
	})
}

func searchGroupChannels(u user.User) control.UserActionResponse {
	user, err := u.Store().RandomUser()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// We simulate the user typing up to 4 characters when searching for
	// a group channel. This is an arbitrary value which fits well with the current
	// frequency value for this action.
	return control.EmulateUserTyping(user.Username[:1+rand.Intn(4)], func(term string) control.UserActionResponse {
		channels, err := u.SearchGroupChannels(&model.ChannelSearch{
			Term: user.Username,
		})
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		return control.UserActionResponse{Info: fmt.Sprintf("found %d channels", len(channels))}
	})
}

func createPrivateChannel(u user.User) control.UserActionResponse {
	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channelName := model.NewId()
	channelId, err := u.CreateChannel(&model.Channel{
		Name:        channelName,
		DisplayName: "Channel " + channelName,
		TeamId:      team.Id,
		Type:        "P",
	})
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// This is a series of calls made by the webapp client
	// when opening the `Add Members` dialog.
	if err := u.GetUsersInChannel(channelId, 0, 100); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.GetChannelMembers(channelId, 0, 50); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	ids, err := u.GetUsersNotInChannel(team.Id, channelId, 0, 100)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// we pick up to 4 users to add to the channel.
	for _, id := range pickIds(ids, 1+rand.Intn(4)) {
		if err := u.AddChannelMember(channelId, id); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("private channel created, id %v", channelId)}
}

func (c *SimulController) scrollChannel(u user.User) control.UserActionResponse {
	collapsedThreads, resp := control.CollapsedThreadsEnabled(u)
	if resp.Err != nil {
		return resp
	}

	channel, err := c.user.Store().CurrentChannel()
	if errors.Is(err, memstore.ErrChannelNotFound) {
		return control.UserActionResponse{Info: "scrollChannel: current channel not set"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	posts, err := c.user.Store().ChannelPostsSorted(channel.Id, true)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if len(posts) == 0 {
		return control.UserActionResponse{Info: fmt.Sprintf("no posts in channel %v", channel.Id)}
	}

	// get the oldest post
	postId := posts[0].Id
	// scrolling between 1 and 5 times
	numScrolls := rand.Intn(5) + 1
	for i := 0; i < numScrolls; i++ {
		postsIds, err := c.user.GetPostsBefore(channel.Id, postId, 0, 30, collapsedThreads)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}

		if err := fetchPostsInfo(u, postsIds); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}

		posts, err := c.user.Store().ChannelPostsSorted(channel.Id, false)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		// get the newest post
		postId = posts[0].Id

		// idle time between scrolls, between 1 and 10 seconds.
		idleTime := time.Duration(1+rand.Intn(10)) * time.Second
		select {
		case <-c.stopChan:
			return control.UserActionResponse{Info: "action canceled"}
		case <-time.After(idleTime):
		}
	}
	return control.UserActionResponse{Info: fmt.Sprintf("scrolled channel %v %d times", channel.Id, numScrolls)}
}

func (c *SimulController) initialJoinTeam(u user.User) control.UserActionResponse {
	resp := c.reload(false)
	if resp.Err != nil {
		return resp
	}

	team, err := c.user.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		// only join a team if we are not in one already.
		return c.joinTeam(c.user)
	}

	return resp
}

func shouldSendTypingEvent(u user.User, channelId string) (bool, error) {
	channelStats, err := u.Store().ChannelStats(channelId)
	if err != nil {
		return false, err
	} else if channelStats == nil {
		return false, fmt.Errorf("no stats found for channel %q", channelId)
	}
	maxNotifications, err := strconv.ParseInt(u.Store().ClientConfig()["MaxNotificationsPerChannel"], 10, 64)
	if err != nil {
		return false, err
	}
	enableTyping, err := strconv.ParseBool(u.Store().ClientConfig()["EnableUserTypingMessages"])
	if err != nil {
		return false, err
	}
	return channelStats.MemberCount < maxNotifications && enableTyping, nil
}

func sendTypingEventIfEnabled(u user.User, channelId string) error {
	if ok, err := shouldSendTypingEvent(u, channelId); ok && err == nil {
		// TODO: possibly add some additional idle time here to simulate the
		// user actually taking time to type a post message.
		return u.SendTypingEvent(channelId, "")
	} else if err != nil {
		return err
	}
	return nil
}
