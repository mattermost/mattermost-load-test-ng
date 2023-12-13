// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"

	"github.com/mattermost/mattermost/server/public/model"
)

type userAction struct {
	name      string
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

	// A full reload always calls GET /api/v4/users?page=0&per_page=100,
	// regardless of GraphQL enabled or not
	_, err := c.user.GetUsers(0, 100)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	var resp control.UserActionResponse
	if c.user.Store().FeatureFlags()["GraphQL"] {
		resp = control.ReloadGQL(c.user)
	} else {
		resp = control.Reload(c.user)
	}
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

	if resp := loadTeam(c.user, team, c.user.Store().FeatureFlags()["GraphQL"]); resp.Err != nil {
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

		appErr, ok := resp.Err.(*control.UserError).Err.(*model.AppError)
		if !ok || strings.Contains(appErr.Id, "invalid_credentials") {
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
	err = c.user.Logout()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
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

	if _, err := c.user.GetUsers(0, 100); err != nil {
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

func loadTeam(u user.User, team *model.Team, gqlEnabled bool) control.UserActionResponse {
	if gqlEnabled {
		chCursor := ""
		cmCursor := ""
		var err error
		for {
			chCursor, cmCursor, err = u.GetChannelsAndChannelMembersGQL(team.Id, true, chCursor, cmCursor)
			if err != nil {
				return control.UserActionResponse{Err: control.NewUserError(err)}
			}
			if chCursor == "" || cmCursor == "" {
				break
			}
		}
	} else {
		if _, err := u.GetChannelsForTeamForUser(team.Id, u.Store().Id(), true); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}

		if err := u.GetChannelMembersForUser(u.Store().Id(), team.Id); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	collapsedThreads, resp := control.CollapsedThreadsEnabled(u)
	if resp.Err != nil {
		return resp
	}

	if _, err := u.GetTeamsUnread("", collapsedThreads); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if _, err := u.GetUserThreads(team.Id, &model.GetUserThreadsOpts{
		TotalsOnly:  true,
		ThreadsOnly: false,
	}); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.GetSidebarCategories(u.Store().Id(), team.Id); err != nil {
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

	if resp := loadTeam(u, &team, c.user.Store().FeatureFlags()["GraphQL"]); resp.Err != nil {
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
		if _, err := u.GetUsersByIds(missingUsers, 0); err != nil {
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
		if _, err := u.ViewChannel(&model.ChannelView{ChannelId: current.Id, CollapsedThreadsSupported: collapsedThreads}); err != nil {
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

	excludeFileCount := true
	// 1% of the time, users will open RHS, which will include the file count as well.
	// This is not an entirely accurate representation of events as we are mixing
	// a normal viewChannel with a viewRHS event
	// But we cannot distinguish between the two at an API level, so our action
	// frequencies are also calculated that way.
	// This is a good enough approximation.
	if rand.Float64() < 0.01 {
		excludeFileCount = false
	}

	if err := u.GetChannelStats(channel.Id, excludeFileCount); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if channel.Type == model.ChannelTypeDirect || channel.Type == model.ChannelTypeGroup {
		category := map[model.ChannelType]string{
			model.ChannelTypeDirect: model.PreferenceCategoryDirectChannelShow,
			model.ChannelTypeGroup:  model.PreferenceCategoryGroupChannelShow,
		}

		// We need to update the user's preferences so that
		// on next reload we can properly fetch opened DMs.
		pref := model.Preferences{
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

	if _, err := u.ViewChannel(&model.ChannelView{ChannelId: channel.Id, PrevChannelId: currentChanId, CollapsedThreadsSupported: collapsedThreads}); err != nil {
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
		case p.Category == model.PreferenceCategoryDirectChannelShow:
			userIds = append(userIds, p.Name)
		}
	}

	if err := c.user.GetUsersStatusesByIds(userIds); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: "got statuses"}
}

func deletePost(u user.User) control.UserActionResponse {
	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	post, err := u.Store().RandomPostForChannelByUser(channel.Id, u.Store().Id())
	if errors.Is(err, memstore.ErrPostNotFound) {
		return control.UserActionResponse{Info: "no posts to delete"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.DeletePost(post.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("post deleted, id %v", post.Id)}
}

func (c *SimulController) updateCustomStatus(u user.User) control.UserActionResponse {
	status := &model.CustomStatus{
		Emoji:     control.RandomEmoji(),
		Text:      control.GenerateRandomSentences(1),
		Duration:  "thirty_minutes",
		ExpiresAt: time.Now().UTC().Add(30 * time.Minute),
	}
	err := u.UpdateCustomStatus(u.Store().Id(), status)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	return control.UserActionResponse{Info: fmt.Sprintf("updated custom status: %s", status.Emoji)}
}

func (c *SimulController) removeCustomStatus(u user.User) control.UserActionResponse {
	err := u.RemoveCustomStatus(u.Store().Id())
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	return control.UserActionResponse{Info: "removed custom status"}
}

func (c *SimulController) createSidebarCategory(u user.User) control.UserActionResponse {
	team, err := u.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("current team should be set"))}
	}

	category := &model.SidebarCategoryWithChannels{
		SidebarCategory: model.SidebarCategory{
			UserId:      u.Store().Id(),
			TeamId:      team.Id,
			DisplayName: "category" + control.PickRandomWord(),
		},
	}

	sidebarCategory, err := u.CreateSidebarCategory(u.Store().Id(), team.Id, category)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("created sidebar category, id %s", sidebarCategory.Id)}
}

func (c *SimulController) updateSidebarCategory(u user.User) control.UserActionResponse {
	team, err := u.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("current team should be set"))}
	}

	cat1, err := u.Store().RandomCategory(team.Id)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	cat2, err := u.Store().RandomCategory(team.Id)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// Not repeatedly looping until we get a different category because there have been edge-cases before
	// ending in infinite loop.s
	if cat1.Id == cat2.Id {
		return control.UserActionResponse{Info: "same categories returned. Skipping."}
	}
	if len(cat1.Channels) <= 1 {
		return control.UserActionResponse{Info: "Not enough categories to remove. Skipping."}
	}

	// We pick a random channel from first category and move to second category.
	channelToMove := control.PickRandomString(cat1.Channels)

	// Find index
	i := findIndex(cat1.Channels, channelToMove)
	// Defense in depth
	if i == -1 {
		return control.UserActionResponse{Info: fmt.Sprintf("Channel %s not found in the category", channelToMove)}
	}

	// Move from the first, and add to second.
	cat1.Channels = append(cat1.Channels[:i], cat1.Channels[i+1:]...)
	cat2.Channels = append(cat2.Channels, channelToMove)

	if err := u.UpdateSidebarCategory(u.Store().Id(), team.Id, []*model.SidebarCategoryWithChannels{&cat1, &cat2}); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("updated sidebar categories, ids [%s, %s]", cat1.Id, cat2.Id)}
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

func (c *SimulController) createPost(u user.User) control.UserActionResponse {
	channel, err := u.Store().CurrentChannel()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := sendTypingEventIfEnabled(u, channel.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// Select the post characteristics
	isReply := rand.Float64() < c.config.PercentReplies
	isUrgent := !isReply && (rand.Float64() < c.config.PercentUrgentPosts)
	hasFilesAttached := rand.Float64() < 0.02

	message, err := createMessage(u, channel, isReply)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	post := &model.Post{
		Message:   message,
		ChannelId: channel.Id,
		CreateAt:  time.Now().Unix() * 1000,
	}

	if isReply {
		var rootId string
		randomPost, err := u.Store().RandomPostForChannel(channel.Id)
		if errors.Is(err, memstore.ErrPostNotFound) {
			return control.UserActionResponse{Info: fmt.Sprintf("no posts found in channel %v", channel.Id)}
		} else if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}

		// Get the ID of the post to which the randomPost replies to,
		// or the ID of the randomPost itself if it's a root post
		if randomPost.RootId != "" {
			rootId = randomPost.RootId
		} else {
			rootId = randomPost.Id
		}

		post.RootId = rootId
	}

	if hasFilesAttached {
		if err := control.AttachFilesToPost(u, post); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	if isUrgent {
		post.Metadata = &model.PostMetadata{}
		post.Metadata.Priority = &model.PostPriority{
			Priority:                model.NewString("urgent"),
			RequestedAck:            model.NewBool(false),
			PersistentNotifications: model.NewBool(false),
		}
	}

	postId, err := u.CreatePost(post)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("post created, id %v", postId)}
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

	if reactionLimit := u.Store().Config().ServiceSettings.UniqueEmojiReactionLimitPerPost; reactionLimit != nil {
		uniqueEmojiNames := map[string]bool{reaction.EmojiName: true}
		for _, r := range reactions {
			uniqueEmojiNames[r.EmojiName] = true
		}

		if len(uniqueEmojiNames) >= *reactionLimit {
			return control.UserActionResponse{Info: "reaction limit reached"}
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
		lastPictureUpdate, err := u.Store().ProfileImageLastUpdated(userId)
		if err != nil {
			return err
		}
		if err := u.GetProfileImageForUser(userId, lastPictureUpdate); err != nil {
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
		if err := emulateMention(u, channel.TeamId, channel.Id, user.Username, u.AutocompleteUsersInChannel); err != nil && !errors.Is(err, errNoMatch) {
			return "", err
		}
		message += "@" + user.Username + " "
	}

	// 10% of messages will contain a link.
	if rand.Float64() < 0.10 {
		message = control.AddLink(message)
	}

	// 1% of messages will contain a permalink
	if rand.Float64() < 0.01 {
		// We want this to be any post from any channel.
		post, err := u.Store().RandomPost(store.SelectAny)
		if err != nil && !errors.Is(err, memstore.ErrPostNotFound) {
			return "", err
		}
		// We ignore in case a post is not found.
		if err == nil {
			siteURL := u.Store().ClientConfig()["SiteURL"]
			team, err := u.Store().CurrentTeam()
			if err != nil {
				return "", err
			}
			pl := siteURL + "/" + team.Name + "/pl/" + post.Id

			message += " " + pl + " "
		}
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

	collapsedThreads, resp := control.CollapsedThreadsEnabled(u)
	if resp.Err != nil {
		return resp
	}

	if _, err := u.ViewChannel(&model.ChannelView{ChannelId: channel.Id, CollapsedThreadsSupported: collapsedThreads}); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: "unread check done"}
}

func (c *SimulController) searchChannels(u user.User) control.UserActionResponse {
	ok, err := control.IsVersionSupported("6.4.0", c.serverVersion)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	var team model.Team
	if ok {
		// Selecting any random team if >=6.4 version.
		team, err = u.Store().RandomTeam(store.SelectMemberOf)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	} else {
		// Selecting only current team otherwise.
		teamPtr, err2 := u.Store().CurrentTeam()
		if err2 != nil {
			return control.UserActionResponse{Err: control.NewUserError(err2)}
		} else if teamPtr == nil {
			return control.UserActionResponse{Err: control.NewUserError(errors.New("current team should be set"))}
		}
		team = *teamPtr
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
		// Searching channels from all teams if >= 6.4 version.
		if ok {
			channels, err := u.SearchChannels(&model.ChannelSearch{
				Term: term,
			})
			if err != nil {
				return control.UserActionResponse{Err: control.NewUserError(err)}
			}
			return control.UserActionResponse{Info: fmt.Sprintf("found %d channels", len(channels))}
		}
		channels, err := u.SearchChannelsForTeam(team.Id, &model.ChannelSearch{
			Term: term,
		})
		// Duplicating the else part because the channels types are different.
		// One is []*model.Channel, other is model.ChannelListWithTeamData
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
			if err := getProfileImageForUsers(u, keys(users)); err != nil {
				return control.UserActionResponse{Err: control.NewUserError(err)}
			}
			if len(users) == 1 {
				return control.UserActionResponse{Err: errors.New("found")}
			}
			return control.UserActionResponse{Info: "emulated user typing users"}
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
			return control.UserActionResponse{Info: "emulated user typing channels"}
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
	numChars := 4
	if numChars > len(user.Username) {
		// rand.Intn returns a number exclusive of the max limit.
		// So there's no need to subtract 1.
		numChars = len(user.Username)
	}
	return control.EmulateUserTyping(user.Username[:1+rand.Intn(numChars)], func(term string) control.UserActionResponse {
		channels, err := u.SearchGroupChannels(&model.ChannelSearch{
			Term: user.Username,
		})
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		return control.UserActionResponse{Info: fmt.Sprintf("found %d channels", len(channels))}
	})
}

func openAddMembersDialog(u user.User, teamId, channelId string) ([]string, control.UserActionResponse) {
	if err := u.GetTeamStats(teamId); err != nil {
		return nil, control.UserActionResponse{Err: control.NewUserError(err)}
	}

	inChannelIds, err := u.GetUsersInChannel(channelId, 0, 50)
	if err != nil {
		return nil, control.UserActionResponse{Err: control.NewUserError(err)}
	}

	notInChannelIds, err := u.GetUsersNotInChannel(teamId, channelId, 0, 100)
	if err != nil {
		return nil, control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.GetUsersStatusesByIds(inChannelIds); err != nil {
		return nil, control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.GetUsersStatusesByIds(notInChannelIds); err != nil {
		return nil, control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// NOTE: the call to ViewChannel is really a side effect and not properly part of this action.
	// Its trigger is the handleBlur() function in the team controller.
	collapsedThreads, resp := control.CollapsedThreadsEnabled(u)
	if resp.Err != nil {
		return nil, resp
	}
	if _, err := u.ViewChannel(&model.ChannelView{ChannelId: channelId, CollapsedThreadsSupported: collapsedThreads}); err != nil {
		return nil, control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return notInChannelIds, control.UserActionResponse{}
}

func createPublicChannel(u user.User) control.UserActionResponse {
	team, err := u.Store().RandomTeam(store.SelectMemberOf)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channelName := model.NewId()
	channelId, err := u.CreateChannel(&model.Channel{
		Name:        channelName,
		DisplayName: "Channel " + channelName,
		TeamId:      team.Id,
		Type:        model.ChannelTypeOpen,
	})
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if _, resp := openAddMembersDialog(u, team.Id, channelId); resp.Err != nil {
		return resp
	}

	// TODO: figure out if it makes sense to add users to the newly created
	// channel following some heuristics.

	return control.UserActionResponse{Info: fmt.Sprintf("public channel created, id %v", channelId)}
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
		Type:        model.ChannelTypePrivate,
	})
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	ids, resp := openAddMembersDialog(u, team.Id, channelId)
	if resp.Err != nil {
		return resp
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

func (c *SimulController) viewGlobalThreads(u user.User) control.UserActionResponse {
	collapsedThreads, resp := control.CollapsedThreadsEnabled(u)
	if resp.Err != nil || !collapsedThreads {
		return resp
	}
	team, err := u.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("viewGlobalThreads: current team should be set"))}
	}

	// View "All your threads" in the Global Threads Screen
	threads, err := u.Store().ThreadsSorted(false, false)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if len(threads) == 0 {
		threads, err = u.GetUserThreads(team.Id, &model.GetUserThreadsOpts{
			PageSize:    25,
			Extended:    false,
			Deleted:     false,
			Unread:      false,
			Since:       0,
			TotalsOnly:  false,
			ThreadsOnly: true,
		})
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		if len(threads) == 0 {
			return control.UserActionResponse{Info: "Visited Global Threads Screen, user has no threads"}
		}
	}

	oldestThreadId := threads[len(threads)-1].PostId
	// scrolling between 1 and 3 times
	numScrolls := rand.Intn(3) + 1
	for i := 0; i < numScrolls; i++ {
		threads, err = u.GetUserThreads(team.Id, &model.GetUserThreadsOpts{
			PageSize:    25,
			Extended:    false,
			Deleted:     false,
			Unread:      false,
			Since:       0,
			TotalsOnly:  false,
			ThreadsOnly: true,
			Before:      oldestThreadId,
		})
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		if len(threads) == 0 {
			break
		}
		oldestThreadId = threads[len(threads)-1].PostId
		// idle time between scrolls, between 1 and 10 seconds.
		idleTime := time.Duration(1+rand.Intn(10)) * time.Second
		select {
		case <-c.stopChan:
			return control.UserActionResponse{Info: "action canceled"}
		case <-time.After(idleTime):
		}
	}

	// Switch to "Unread" tabs
	unreadThreads, err := u.Store().ThreadsSorted(true, false)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if len(unreadThreads) == 0 {
		unreadThreads, err = u.GetUserThreads(team.Id, &model.GetUserThreadsOpts{
			PageSize:    25,
			Extended:    false,
			Deleted:     false,
			Unread:      true,
			Since:       0,
			TotalsOnly:  false,
			ThreadsOnly: true,
		})
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		if len(unreadThreads) == 0 {
			return control.UserActionResponse{Info: "Visited Global Threads Screen, user has no unread threads"}
		}
	}

	oldestUnreadThreadId := unreadThreads[len(unreadThreads)-1].PostId
	// scrolling between 1 and 3 times
	numScrolls = rand.Intn(3) + 1
	for i := 0; i < numScrolls; i++ {
		unreadThreads, err = u.GetUserThreads(team.Id, &model.GetUserThreadsOpts{
			PageSize:    25,
			Extended:    false,
			Deleted:     false,
			Unread:      true,
			Since:       0,
			TotalsOnly:  false,
			ThreadsOnly: true,
			Before:      oldestUnreadThreadId,
		})
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		if len(unreadThreads) == 0 {
			break
		}
		oldestUnreadThreadId = unreadThreads[len(unreadThreads)-1].PostId
		// idle time between scrolls, between 1 and 10 seconds.
		idleTime := time.Duration(1+rand.Intn(10)) * time.Second
		select {
		case <-c.stopChan:
			return control.UserActionResponse{Info: "action canceled"}
		case <-time.After(idleTime):
		}
	}

	return control.UserActionResponse{Info: "Visited Global Threads Screen"}
}

func (c *SimulController) followThread(u user.User) control.UserActionResponse {
	collapsedThreads, resp := control.CollapsedThreadsEnabled(u)
	if resp.Err != nil || !collapsedThreads {
		return resp
	}
	channel, err := u.Store().CurrentChannel()
	if err != nil {
		if errors.Is(err, memstore.ErrChannelNotFound) {
			return control.UserActionResponse{Info: "followThread: current channel not set"}
		}
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	post, err := u.Store().RandomReplyPostForChannel(channel.Id)
	if err != nil && !errors.Is(err, memstore.ErrPostNotFound) {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if errors.Is(err, memstore.ErrPostNotFound) {
		post, err = u.Store().RandomPostForChannel(channel.Id)
		if err != nil {
			if errors.Is(err, memstore.ErrPostNotFound) {
				return control.UserActionResponse{Info: "followThread: no posts in store to follow"}
			}
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}
	id := post.RootId
	if id == "" {
		id = post.Id
	}
	err = u.UpdateThreadFollow(channel.TeamId, id, true)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("followed thread %s", id)}
}

func (c *SimulController) unfollowThread(u user.User) control.UserActionResponse {
	collapsedThreads, resp := control.CollapsedThreadsEnabled(u)
	if resp.Err != nil || !collapsedThreads {
		return resp
	}
	thread, err := u.Store().RandomThread()
	if err != nil {
		if errors.Is(err, memstore.ErrThreadNotFound) {
			return control.UserActionResponse{Info: "unfollow thread: no thread to unfollow"}
		}
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	channel, err := u.Store().Channel(thread.Post.ChannelId)
	if err != nil || channel == nil {
		err = u.GetChannel(thread.Post.ChannelId)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		channel, err = u.Store().Channel(thread.Post.ChannelId)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		if channel == nil {
			return control.UserActionResponse{Err: control.NewUserError(errors.New("unfollow thread: can't get channel for thread"))}
		}
	}
	err = u.UpdateThreadFollow(channel.TeamId, thread.PostId, false)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("unfollowed thread %s", thread.PostId)}
}

func (c *SimulController) viewThread(u user.User) control.UserActionResponse {
	collapsedThreads, resp := control.CollapsedThreadsEnabled(u)
	if resp.Err != nil || !collapsedThreads {
		return resp
	}
	// get a random thread
	thread, err := u.Store().RandomThread()
	if err != nil && !errors.Is(err, memstore.ErrThreadNotFound) {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	// we don't have threads in store lets get some
	if errors.Is(err, memstore.ErrThreadNotFound) {
		team, err := u.Store().CurrentTeam()
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		} else if team == nil {
			return control.UserActionResponse{Err: control.NewUserError(errors.New("viewthread: current team should be set"))}
		}
		threads, err := u.GetUserThreads(team.Id, &model.GetUserThreadsOpts{
			PageSize:    25,
			Extended:    false,
			Deleted:     false,
			Unread:      false,
			Since:       0,
			TotalsOnly:  false,
			ThreadsOnly: true,
		})
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		if len(threads) == 0 {
			return control.UserActionResponse{Info: "viewthread: no threads available to view"}
		}
		thread = *threads[0]
	}

	postIds, hasNext, err := u.GetPostThreadWithOpts(thread.PostId, "", model.GetPostsOptions{
		CollapsedThreads: true,
		Direction:        "down",
		PerPage:          25,
	})
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if len(postIds) == 0 {
		return control.UserActionResponse{Info: "viewthread: no posts available to view in thread"}
	}
	newestPostId := postIds[len(postIds)-1]
	newestPost, err := u.Store().Post(newestPostId)
	if err != nil && !errors.Is(err, memstore.ErrPostNotFound) {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	var newestCreateAt int64
	if errors.Is(err, memstore.ErrPostNotFound) {
		newestCreateAt = thread.Post.CreateAt
	} else {
		newestCreateAt = newestPost.CreateAt
	}

	// scrolling between 1 and 3 times
	numScrolls := rand.Intn(3) + 1
	for i := 0; i < numScrolls && hasNext; i++ {
		postIds, hasNext, err = u.GetPostThreadWithOpts(thread.PostId, "", model.GetPostsOptions{
			CollapsedThreads: true,
			Direction:        "down",
			PerPage:          25,
			FromPost:         newestPostId,
			FromCreateAt:     newestCreateAt,
		})
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		if !hasNext {
			break
		}
		newestPostId = postIds[len(postIds)-1]
		newestPost, err = u.Store().Post(newestPostId)
		if err != nil && !errors.Is(err, memstore.ErrPostNotFound) {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		if errors.Is(err, memstore.ErrPostNotFound) {
			newestCreateAt = thread.Post.CreateAt
		} else {
			newestCreateAt = newestPost.CreateAt
		}

		// idle time between scrolls, between 1 and 10 seconds.
		idleTime := time.Duration(1+rand.Intn(10)) * time.Second
		select {
		case <-c.stopChan:
			return control.UserActionResponse{Info: "action canceled"}
		case <-time.After(idleTime):
		}
	}
	return control.UserActionResponse{Info: fmt.Sprintf("viewedthread %s", thread.PostId)}
}

func (c *SimulController) markAllThreadsInTeamAsRead(u user.User) control.UserActionResponse {
	collapsedThreads, resp := control.CollapsedThreadsEnabled(u)
	if resp.Err != nil || !collapsedThreads {
		return resp
	}
	team, err := u.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if team == nil {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("markAllThreadsInTeamAsRead: current team should be set"))}
	}
	err = u.MarkAllThreadsInTeamAsRead(team.Id)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	return control.UserActionResponse{Info: fmt.Sprintf("marked all threads in team %s as read", team.Id)}
}

func (c *SimulController) updateThreadRead(u user.User) control.UserActionResponse {
	collapsedThreads, resp := control.CollapsedThreadsEnabled(u)
	if resp.Err != nil || !collapsedThreads {
		return resp
	}
	// get a random thread
	thread, err := u.Store().RandomThread()
	if err != nil && !errors.Is(err, memstore.ErrThreadNotFound) {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	// we don't have threads in store lets get some
	if errors.Is(err, memstore.ErrThreadNotFound) {
		team, err := u.Store().CurrentTeam()
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		} else if team == nil {
			return control.UserActionResponse{Err: control.NewUserError(errors.New("updateThreadRead: current team should be set"))}
		}
		threads, err := u.GetUserThreads(team.Id, &model.GetUserThreadsOpts{
			PageSize:    25,
			Extended:    false,
			Deleted:     false,
			Unread:      false,
			Since:       0,
			TotalsOnly:  false,
			ThreadsOnly: false,
		})
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		if len(threads) == 0 {
			return control.UserActionResponse{Info: "updateThreadRead: no threads available"}
		}
		thread, err = u.Store().RandomThread()
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}
	channel, err := u.Store().Channel(thread.Post.ChannelId)
	if err != nil || channel == nil {
		err = u.GetChannel(thread.Post.ChannelId)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		channel, err = u.Store().Channel(thread.Post.ChannelId)
		if err != nil || channel == nil {
			return control.UserActionResponse{Err: control.NewUserError(errors.New("updateThreadRead: can't get channel for thread"))}
		}
	}

	// We set thread read time to the createat of the root post.
	// This is an easy, valid timestamp and causes the server to
	// recalculate all mentions in the thread.
	err = u.UpdateThreadRead(channel.TeamId, thread.PostId, thread.Post.CreateAt)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("updated read state of thread %s", thread.PostId)}
}

func (c *SimulController) createPostReminder(u user.User) control.UserActionResponse {
	ch, err := u.Store().CurrentChannel()
	if errors.Is(err, memstore.ErrChannelNotFound) {
		return control.UserActionResponse{Info: "current channel is not set"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	post, err := u.Store().RandomPostForChannel(ch.Id)
	if errors.Is(err, memstore.ErrPostNotFound) {
		return control.UserActionResponse{Info: fmt.Sprintf("no post in channel: %s", ch.Id)}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// Going with a hardcoded 10 minute addition for now.
	// Probably there's no need to randomize this yet.
	err = u.CreatePostReminder(u.Store().Id(), post.Id, time.Now().Add(10*time.Minute).Unix())
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("created post reminder, id %s", post.Id)}
}

func (c *SimulController) reconnectWebSocket(u user.User) control.UserActionResponse {
	team, err := c.user.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if team != nil {
		// The webapp makes these requests only if the team
		// is set in the state.

		if _, err := u.GetChannelsForTeamForUser(team.Id, u.Store().Id(), false); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}

		if err := u.GetChannelMembersForUser(u.Store().Id(), team.Id); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}

		// getPostsForChannel(?since=mostRecentPost.CreateAt in CurrentChannel)
		crtEnabled, resp := control.CollapsedThreadsEnabled(u)
		if resp.Err != nil {
			return resp
		}
		currentCh, err := u.Store().CurrentChannel()
		if errors.Is(err, memstore.ErrChannelNotFound) {
			return control.UserActionResponse{Info: "current channel is not set"}
		} else if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		posts, err := u.Store().ChannelPostsSorted(currentCh.Id, true)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		// If there are posts in the channel, retrieve the statuses of their authors
		if len(posts) > 0 {
			latestCreateAt := posts[len(posts)-1].CreateAt
			if _, err := u.GetPostsSince(currentCh.Id, latestCreateAt, crtEnabled); err != nil {
				return control.UserActionResponse{Err: control.NewUserError(err)}
			}

			// unique user ids from all posts in current Channel + users from DM prefs
			userIds := usersForPosts(posts)
			prefs, err := u.Store().Preferences()
			if err != nil {
				return control.UserActionResponse{Err: control.NewUserError(err)}
			}

			for _, p := range prefs {
				switch {
				case p.Category == model.PreferenceCategoryDirectChannelShow:
					userIds[p.Name] = true
				}
			}

			if err := u.GetUsersStatusesByIds(keys(userIds)); err != nil {
				return control.UserActionResponse{Err: control.NewUserError(err)}
			}
		}

		if _, err := u.GetTeamsUnread("", crtEnabled); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	if err := u.GetWebappPlugins(); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// All users in state.
	users, err := u.Store().Users()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	userIds := make([]string, len(users))
	for i, u := range users {
		userIds[i] = u.Id
	}
	// Assuming reconnect happened just now.
	if _, err := u.GetUsersByIds(userIds, time.Now().UnixMilli()); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: "reconnected ws"}
}

func (c *SimulController) openUserProfile(u user.User) control.UserActionResponse {
	ch, err := u.Store().CurrentChannel()
	if errors.Is(err, memstore.ErrChannelNotFound) {
		return control.UserActionResponse{Info: "current channel is not set"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	post, err := u.Store().RandomPostForChannel(ch.Id)
	if errors.Is(err, memstore.ErrPostNotFound) {
		return control.UserActionResponse{Info: fmt.Sprintf("no post in channel: %s", ch.Id)}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// We simulate a click on the user profile who wrote the post.
	// The webapp will also do an additional viewChannel request once per-channel.
	// But we avoid that for now because viewChannel is already handled separately.
	if err := u.GetChannelMember(ch.Id, post.UserId); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	// If it's a DM/GM channel, the webapp still sends the current team
	// the user is part of
	if ch.TeamId == "" {
		team, err := u.Store().CurrentTeam()
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		} else if team == nil {
			return control.UserActionResponse{Err: control.NewUserError(errors.New("current team should be set"))}
		}

		ch.TeamId = team.Id
	}
	if err := u.GetTeamMember(ch.TeamId, post.UserId); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("clicked user profile %s", post.UserId)}
}

// openPermalink emulates a user clicking a permalink,
// rather than finding a post with a permalink in the current channel.
// That would require some parsing to find out a permalink.
func (c *SimulController) openPermalink(u user.User) control.UserActionResponse {
	currentCh, err := u.Store().CurrentChannel()
	if errors.Is(err, memstore.ErrChannelNotFound) {
		return control.UserActionResponse{Info: "current channel is not set"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	posts, err := u.Store().ChannelPosts(currentCh.Id)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	var postID string
	for _, p := range posts {
		// If there are multiple permalinks found, we will click on the last one
		// in the channel. This naturally leads to an effect of having clicked all
		// permalinks which emulates organic behavior.
		postID = getPermalinkPostIDFromMessage(p.Message)
	}
	if postID == "" {
		return control.UserActionResponse{Info: "no permalink found"}
	}

	crtEnabled, resp := control.CollapsedThreadsEnabled(u)
	if resp.Err != nil {
		return resp
	}

	// We fetch the post thread.
	postIds, _, err := u.GetPostThreadWithOpts(postID, "", model.GetPostsOptions{
		CollapsedThreads: crtEnabled,
		Direction:        "down",
		PerPage:          25,
	})
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	oldestPost, err := u.Store().Post(postIds[0])
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	ch, err := u.Store().Channel(oldestPost.ChannelId)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	// We fetch the channel if it's not present in state.
	// This is similar to what the webapp does.
	if ch == nil {
		if err := u.GetChannel(oldestPost.ChannelId); err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		// We fetch the channel data from store again.
		ch, err = u.Store().Channel(oldestPost.ChannelId)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
	}

	// We check if membership data is already there. If not,
	// we call getChannelMember, and add the user to the channel
	// if it's not a private channel.
	cm, err := u.Store().ChannelMember(ch.Id, u.Store().Id())
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if cm.UserId == "" {
		err := u.GetChannelMember(ch.Id, u.Store().Id())
		if err != nil {
			// Resorting to string matching, because the API doesn't return the
			// actual Response object.
			if !strings.Contains(err.Error(), "You do not have the appropriate permissions") {
				return control.UserActionResponse{Err: control.NewUserError(err)}
			}

			if ch.Type != model.ChannelTypePrivate {
				if err := u.AddChannelMember(ch.Id, u.Store().Id()); err != nil {
					return control.UserActionResponse{Err: control.NewUserError(err)}
				}
			}
		}
	}

	currentTeam, err := u.Store().CurrentTeam()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	} else if currentTeam == nil {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("current team should be set"))}
	}

	if _, err := u.GetChannelsForTeamForUser(currentTeam.Id, u.Store().Id(), false); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	if err := u.GetChannelMembersForUser(u.Store().Id(), currentTeam.Id); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	// Switch to channel.
	if resp := viewChannel(u, ch); resp.Err != nil {
		return control.UserActionResponse{Err: control.NewUserError(resp.Err)}
	}

	if err := u.SetCurrentChannel(ch); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("clicked permalink on post %s", postID)}
}

func (c *SimulController) generateUserReport(u user.User) control.UserActionResponse {
	// Simulate scrolling through the entire list of users
	// (should be similar to generating the complete report and exporting it)

	lastColumnValue := ""
	lastId := ""
	totalUsers := 0

	for {
		report, err := u.GetUsersForReporting(&model.UserReportOptionsAPI{
			UserReportOptionsWithoutDateRange: model.UserReportOptionsWithoutDateRange{
				SortColumn:          "Username",
				PageSize:            50,
				LastSortColumnValue: lastColumnValue,
				LastUserId:          lastId,
			},
		})

		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}

		totalUsers += len(report)
		if len(report) < 50 {
			break
		}

		lastColumnValue = report[len(report) - 1].Username
		lastId = report[len(report) - 1].Id
	}

	return control.UserActionResponse{Info: fmt.Sprintf("generated user report for %d users", totalUsers)}
}
