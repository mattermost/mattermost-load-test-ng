// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package user

import (
	"regexp"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/store"

	"github.com/mattermost/mattermost-server/v6/model"
)

// TestUserSuffixRegexp matches the numerical suffix of test usernames,
// which are assumed to be in this format.
var TestUserSuffixRegexp = regexp.MustCompile(`\d+$`)

type GraphQLInput struct {
	Query         string                 `json:"query"`
	OperationName string                 `json:"operationName"`
	Variables     map[string]interface{} `json:"variables"`
}

// User provides a wrapper interface to interact with the Mattermost server
// through its client APIs. It persists the data to its UserStore for later use.
type User interface {
	// store
	// Store exposes the underlying UserStore.
	Store() store.UserStore
	// ClearUserData calls the Clear method on the underlying UserStore.
	ClearUserData()

	// websocket
	// Connect creates a WebSocket connection to the server and starts listening for messages.
	Connect() (<-chan error, error)
	// Disconnect closes the WebSocket connection.
	Disconnect() error
	// Events returns the WebSocket event chan for the controller
	// to listen and react to events.
	Events() <-chan *model.WebSocketEvent
	// SendTypingEvent will push a user_typing event out to all connected users
	// who are in the specified channel.
	SendTypingEvent(channelId, parentId string) error

	//server
	// GetConfig fetches and stores the server's configuration.
	GetConfig() error
	// GetClientConfig fetches and stores the limited server's configuration for logged in user.
	GetClientConfig() error
	// FetchStaticAssets parses index.html and fetches static assets mentioned in link/script tags.
	FetchStaticAssets() error
	// GetClientLicense fetched and stores the client license.
	// It returns the client license in the old format.
	GetClientLicense() error

	// user
	// SignUp signs up the user with the given credentials.
	SignUp(email, username, password string) error
	// Login logs the user in. It authenticates a user and starts a new session.
	Login() error
	// Logout logs the user out. It terminates the current user's session.
	Logout() error
	// GetMe loads user's information into the store and returns its id.
	GetMe() (string, error)
	// GetPreferences fetches and store the user's preferences.
	GetPreferences() error
	// UpdatePreferences updates the user's preferences.
	UpdatePreferences(pref model.Preferences) error
	// CreateUser creates a new user with the given information.
	CreateUser(user *model.User) (string, error)
	// UpdateUser updates the given user with the given information.
	UpdateUser(user *model.User) error
	// UpdateUserRoles updates the given userId with the given role ids.
	UpdateUserRoles(userId, roles string) error
	// PatchUser patches a given user with the given information.
	PatchUser(userId string, patch *model.UserPatch) error
	// GetUsersByIds fetches and stores the specified users.
	// It returns a list of those users' ids.
	GetUsersByIds(userIds []string) ([]string, error)
	// GetUsersByUsername fetches and stores users for the given usernames.
	// It returns a list of those users' ids.
	GetUsersByUsernames(usernames []string) ([]string, error)
	// GetUserStatus fetches and stores the status for the user.
	GetUserStatus() error
	// GetUsersStatusesByIds fetches and stores statuses for the specified users.
	GetUsersStatusesByIds(userIds []string) error
	// GetUsersInChannel fetches and stores users in the specified channel.
	GetUsersInChannel(channelId string, page, perPage int) error
	// GetUsers fetches and stores all users. It returns a list of those users' ids.
	GetUsers(page, perPage int) ([]string, error)
	// GetUsersNotInChannel returns a list of user ids not in a given channel.
	GetUsersNotInChannel(teamId, channelId string, page, perPage int) ([]string, error)

	// SetProfileImage sets the profile image for the user.
	SetProfileImage(data []byte) error
	// GetProfileImage fetches the profile image for the user.
	GetProfileImage() error
	// GetProfileImageForUser fetches and stores the profile imagine for the
	// specified user.
	GetProfileImageForUser(userId string) error
	// SearchUsers performs a user search. It returns a list of users that matched.
	SearchUsers(search *model.UserSearch) ([]*model.User, error)
	// AutocompleteUsersInChannel performs autocomplete of a username
	// in a specified team and channel.
	// It returns the users in the system based on the given username.
	AutocompleteUsersInChannel(teamId, channelId, username string, limit int) (map[string]bool, error)
	// AutocompleteUsersInTeam performs autocomplete of a username
	// in a specified team.
	// It returns the users in the system based on the given username.
	AutocompleteUsersInTeam(teamId, username string, limit int) (map[string]bool, error)

	// posts
	// CreatePost creates and stores a new post made by the user.
	CreatePost(post *model.Post) (string, error)
	// PatchPost modifies a post for the given postId and stores the updated result.
	PatchPost(postId string, patch *model.PostPatch) (string, error)
	// DeletePost deletes a post for the given postId.
	DeletePost(postId string) error
	// SearchPosts performs a search for posts in the given teamId with the given terms.
	SearchPosts(teamId, terms string, isOrSearch bool) (*model.PostList, error)
	// GetPostsForChannel fetches and stores posts in a given channelId.
	GetPostsForChannel(channelId string, page, perPage int, collapsedThreads bool) error
	// GetPostsBefore fetches and stores posts in a given channelId that were made
	// before a given postId. It returns a list of posts ids.
	GetPostsBefore(channelId, postId string, page, perPage int, collapsedThreads bool) ([]string, error)
	// GetPostsAfter fetches and stores posts in a given channelId that were made
	// after a given postId.
	GetPostsAfter(channelId, postId string, page, perPage int, collapsedThreads bool) error
	// GetPostsSince fetches and stores posts in a given channelId that were made
	// since the given time. It returns a list of posts ids.
	GetPostsSince(channelId string, time int64, collapsedThreads bool) ([]string, error)
	// GetPinnedPosts fetches and returns pinned posts in a given channelId.
	GetPinnedPosts(channelId string) (*model.PostList, error)
	// GetPostsAroundLastUnread fetches and stores the posts made around last
	// unread in a given channelId. It returns a list of posts ids.
	GetPostsAroundLastUnread(channelId string, limitBefore, limitAfter int, collapsedThreads bool) ([]string, error)

	// files
	// UploadFile uploads the given data in the specified channel.
	UploadFile(data []byte, channelId, filename string) (*model.FileUploadResponse, error)
	// GetFileInfosForPost returns file information for the specified post.
	GetFileInfosForPost(postId string) ([]*model.FileInfo, error)
	// GetFileThumbnail fetches the thumbnail for the specified file.
	GetFileThumbnail(fileId string) error
	// GetFilePreview fetches the preview for the specified file.
	GetFilePreview(fileId string) error

	// channels
	// CreateChannel creates and stores a new channel with the given information.
	// It returns the channel's id.
	CreateChannel(channel *model.Channel) (string, error)
	// CreateGroupChannel creates and stores a new group channel with the given
	// members. It returns the channel's id.
	CreateGroupChannel(memberIds []string) (string, error)
	// CreateGroupChannel creates and stores a new direct channel with the given
	// user. It returns the channel's id.
	CreateDirectChannel(otherUserId string) (string, error)
	// GetChannel fetches and stores the specified channel.
	GetChannel(channelId string) error
	// GetChannelsForTeam fetches and stores channels in the specified team.
	GetChannelsForTeam(teamId string, includeDeleted bool) error
	// GetPublicChannelsForTeam fetches and stores public channels in the
	// specified team.
	GetPublicChannelsForTeam(teamId string, page, perPage int) error
	// SearchChannelsForTeam performs a search for channels in the specified team.
	// It returns channels that matches the search.
	SearchChannelsForTeam(teamId string, search *model.ChannelSearch) ([]*model.Channel, error)
	// SearchChannels performs a search for channels in all teams for a user.
	SearchChannels(search *model.ChannelSearch) (model.ChannelListWithTeamData, error)
	// SearchGroupChannels performs a search for group channels.
	// It returns channels whose members' usernames match the search term.
	SearchGroupChannels(search *model.ChannelSearch) ([]*model.Channel, error)
	// RemoveUserFromChannel removes the specified user from the specified channel.
	RemoveUserFromChannel(channelId, userId string) error
	// ViewChannels performs a channel view for the user.
	ViewChannel(view *model.ChannelView) (*model.ChannelViewResponse, error)
	// GetChannelUnread fetches and returns information about the specified channel's unread
	// messages.
	GetChannelUnread(channelId string) (*model.ChannelUnread, error)
	// GetChannelMembers fetches and stores channel members for the specified channel.
	GetChannelMembers(channelId string, page, perPage int) error
	// GetChannelMembersForUser gets the channel members for the specified user in
	// the specified team.
	GetChannelMembersForUser(userId, teamId string) error
	// GetChannelMember fetches and stores the channel member for the specified user in
	// the specified channel.
	GetChannelMember(channelId string, userId string) error
	// GetChannelStats fetches statistics for the specified channel.
	GetChannelStats(channelId string, excludeFileCount bool) error
	// AddChannelMember adds the specified user to the specified channel.
	AddChannelMember(channelId, userId string) error
	// GetChannelsForTeamForUser fetches and stores chanels for the specified user in
	// the specified team. It returns a list of those channels.
	GetChannelsForTeamForUser(teamId, userId string, includeDeleted bool) ([]*model.Channel, error)
	// AutocompleteChannelsForTeam returns an ordered list of channels for a given
	// name in a specified team.
	AutocompleteChannelsForTeam(teamId, name string) error
	// AutocompleteChannelsForTeamForSearch fetches and stores an ordered list of the
	// user's channels autocomplete suggestions. It returns a map of found channel names.
	AutocompleteChannelsForTeamForSearch(teamId, name string) (map[string]bool, error)
	// GetChannelsForUser returns all channels from all teams for a given user.
	GetChannelsForUser(userID string) ([]*model.Channel, error)

	// teams
	// GetAllTeams returns all teams based on permissions.
	// It returns a list of team ids.
	GetAllTeams(page, perPage int) ([]string, error)
	// CreateTeam creates a new team with the given information.
	CreateTeam(team *model.Team) (string, error)
	// GetTeam fetches and returns the specified team.
	GetTeam(teamId string) error
	// GetTeamsForUser fetches and stores the teams for the specified user.
	// It returns a list of team ids.
	GetTeamsForUser(userId string) ([]string, error)
	// AddTeamMember adds the specified user to the specified team.
	AddTeamMember(teamId, userId string) error
	// RemoveTeamMember removes the specified user from the specified team.
	RemoveTeamMember(teamId, userId string) error
	// GetTeamMembers fetches and stores team members for the specified team.
	GetTeamMembers(teamId string, page, perPage int) error
	// GetTeamMembersForUser fetches and stores team members for the specified user.
	GetTeamMembersForUser(userId string) error
	// GetTeamStats fetches statistics for the specified team.
	GetTeamStats(teamId string) error
	// GetTeamsUnread fetches and returns information about unreads messages for
	// the user in the teams it belongs to.
	GetTeamsUnread(teamIdToExclude string, includeCollapsedThreads bool) ([]*model.TeamUnread, error)
	// AddTeamMemberFromInvite adds a user to a team using the given token and
	// inviteId.
	AddTeamMemberFromInvite(token, inviteId string) error
	// UpdateTeam updates and stores the given team.
	UpdateTeam(team *model.Team) error

	// roles
	// GetRolesByName fetches and stores roles for the given names.
	// It returns a list of role ids.
	GetRolesByNames(roleNames []string) ([]string, error)

	// emoji
	// GetEmojiList fetches and stores a list of custom emoji.
	GetEmojiList(page, perPage int) error
	// GetEmojiImage fetches the image for a given emoji.
	GetEmojiImage(emojiId string) error

	// reactions
	// SaveReaction stores the given reaction.
	SaveReaction(reaction *model.Reaction) error
	// DeleteReaction deletes the given reaction.
	DeleteReaction(reaction *model.Reaction) error
	// GetReactions fetches and stores reactions to the specified post.
	GetReactions(postId string) error

	// plugins
	// GetWebappPlugins fetches webapp plugins.
	GetWebappPlugins() error

	// utils
	// IsSysAdmin returns whether the user is a system admin or not.
	IsSysAdmin() (bool, error)
	// IsTeamAdmin returns whether the user is a team admin or not.
	IsTeamAdmin() (bool, error)
	// SetCurrentTeam sets the given team as the current team for the user.
	SetCurrentTeam(team *model.Team) error
	// SetCurrentChannel sets the given channel as the current channel for the user.
	SetCurrentChannel(channel *model.Channel) error

	// System console functionalities
	// GetLogs fetches the server logs.
	GetLogs(page, perPage int) error
	// GetAnalytics fetches the system analytics.
	GetAnalytics() error
	// GetClusterStatus fetches the cluster status.
	GetClusterStatus() error
	// GetPluginStatuses fetches the plugin statuses.
	GetPluginStatuses() error
	// UpdateConfig updates the config with cfg.
	UpdateConfig(cfg *model.Config) error
	// MessageExport triggers a message export
	MessageExport() error

	// Threads
	// GetUserThreads fetches and stores threads. It returns a list of thread ids.
	GetUserThreads(teamId string, options *model.GetUserThreadsOpts) ([]*model.ThreadResponse, error)
	// UpdateThreadFollow updates the follow state of the thread
	UpdateThreadFollow(teamId, threadId string, state bool) error
	// GetPostThread gets a post with all the other posts in the same thread.
	GetPostThreadWithOpts(threadId, etag string, opts model.GetPostsOptions) ([]string, bool, error)
	// MarkAllThreadsInTeamAsRead marks all threads in a team as read
	MarkAllThreadsInTeamAsRead(teamId string) error
	// UpdateThreadRead updates the read timestamp of the thread
	UpdateThreadRead(teamId, threadId string, timestamp int64) error

	// SidebarCategories
	GetSidebarCategories(userID, teamID string) error
	CreateSidebarCategory(userID, teamID string, category *model.SidebarCategoryWithChannels) (*model.SidebarCategoryWithChannels, error)
	UpdateSidebarCategory(userID, teamID string, categories []*model.SidebarCategoryWithChannels) error

	// Insights
	GetTopThreadsForTeamSince(userID, teamID string, duration string, offset int, limit int) (*model.TopThreadList, error)
	GetTopThreadsForUserSince(userID, teamID string, duration string, offset int, limit int) (*model.TopThreadList, error)
	GetTopChannelsForTeamSince(userID, teamID string, duration string, offset int, limit int) (*model.TopChannelList, error)
	GetTopChannelsForUserSince(userID, teamID string, duration string, offset int, limit int) (*model.TopChannelList, error)
	GetTopReactionsForTeamSince(userID, teamID string, duration string, offset int, limit int) (*model.TopReactionList, error)
	GetTopReactionsForUserSince(userID, teamID string, duration string, offset int, limit int) (*model.TopReactionList, error)
	GetTopInactiveChannelsForTeamSince(userID, teamID string, duration string, offset int, limit int) (*model.TopInactiveChannelList, error)
	GetTopInactiveChannelsForUserSince(userID, teamID string, duration string, offset int, limit int) (*model.TopInactiveChannelList, error)
	GetTopDMsForUserSince(duration string, offset int, limit int) (*model.TopDMList, error)
	GetNewTeamMembersSince(teamID string, duration string, offset int, limit int) (*model.NewTeamMembersList, error)
	// Custom Status
	UpdateCustomStatus(userID string, status *model.CustomStatus) error
	RemoveCustomStatus(userID string) error

	// CreatePostReminder creates a post reminder at a given target time.
	CreatePostReminder(userID, postID string, targetTime int64) error

	// GraphQL
	GetInitialDataGQL() error
	GetChannelsAndChannelMembersGQL(teamID string, includeDeleted bool, channelsCursor, channelMembersCursor string) (string, string, error)
}
