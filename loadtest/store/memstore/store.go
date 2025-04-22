// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost/server/public/model"
)

// MemStore is a simple implementation of MutableUserStore
// which holds all data in memory.
type MemStore struct {
	lock                  sync.RWMutex
	user                  *model.User
	preferences           model.Preferences
	config                *model.Config
	clientConfig          map[string]string
	emojis                []*model.Emoji
	posts                 map[string]*model.Post
	postsQueue            *CQueue[model.Post]
	teams                 map[string]*model.Team
	channels              map[string]*model.Channel
	channelStats          map[string]*model.ChannelStats
	channelMembers        map[string]map[string]*model.ChannelMember
	channelMembersQueue   *CQueue[model.ChannelMember]
	teamMembers           map[string]map[string]*model.TeamMember
	users                 map[string]*model.User
	usersQueue            *CQueue[model.User]
	statuses              map[string]*model.Status
	statusesQueue         *CQueue[model.Status]
	reactions             map[string][]*model.Reaction
	reactionsQueue        *CQueue[model.Reaction]
	roles                 map[string]*model.Role
	license               map[string]string
	currentChannel        *model.Channel
	currentTeam           *model.Team
	channelViews          map[string]int64
	profileImages         map[string]int
	serverVersion         string
	threads               map[string]*model.ThreadResponse
	threadsQueue          *CQueue[model.ThreadResponse]
	sidebarCategories     map[string]map[string]*model.SidebarCategoryWithChannels
	drafts                map[string]map[string]*model.Draft
	featureFlags          map[string]bool
	report                *model.PerformanceReport
	channelBookmarks      map[string]*model.ChannelBookmarkWithFileInfo
	customAttributeFields []*model.PropertyField
	scheduledPosts        map[string]map[string][]*model.ScheduledPost // map of team ID -> channel/thread ID -> list of scheduled posts
	customAttributeValues map[string]map[string]json.RawMessage
}

// New returns a new instance of MemStore with the given config.
// If config is nil, defaults will be used.
func New(config *Config) (*MemStore, error) {
	if config == nil {
		config = &Config{}
		config.SetDefaults()
	}
	if err := config.IsValid(); err != nil {
		return nil, fmt.Errorf("memstore: config validation failed %w", err)
	}

	s := &MemStore{}

	if err := s.setupQueues(config); err != nil {
		return nil, err
	}

	s.Clear()
	s.profileImages = map[string]int{}

	return s, nil
}

// Clear resets the store and removes all entries with the exception of the
// user object and state information (current team/channel) which are preserved.
func (s *MemStore) Clear() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.preferences = nil
	s.config = nil
	clear(s.emojis)
	s.emojis = []*model.Emoji{}
	clear(s.posts)
	s.posts = map[string]*model.Post{}
	clear(s.clientConfig)
	s.clientConfig = map[string]string{}
	s.postsQueue.Reset()
	clear(s.teams)
	s.teams = map[string]*model.Team{}
	clear(s.channels)
	s.channels = map[string]*model.Channel{}
	channelStats := map[string]*model.ChannelStats{}
	if s.currentChannel != nil && s.channelStats[s.currentChannel.Id] != nil {
		channelStats[s.currentChannel.Id] = s.channelStats[s.currentChannel.Id]
	}
	s.channelStats = channelStats
	clear(s.channelMembers)
	s.channelMembers = map[string]map[string]*model.ChannelMember{}
	s.channelMembersQueue.Reset()
	clear(s.teamMembers)
	s.teamMembers = map[string]map[string]*model.TeamMember{}
	clear(s.users)
	s.users = map[string]*model.User{}
	s.usersQueue.Reset()
	clear(s.statuses)
	s.statuses = map[string]*model.Status{}
	s.statusesQueue.Reset()
	clear(s.reactions)
	s.reactions = map[string][]*model.Reaction{}
	s.reactionsQueue.Reset()
	clear(s.roles)
	s.roles = map[string]*model.Role{}
	clear(s.license)
	s.license = map[string]string{}
	clear(s.channelViews)
	s.channelViews = map[string]int64{}
	clear(s.threads)
	s.threads = map[string]*model.ThreadResponse{}
	s.threadsQueue.Reset()
	clear(s.sidebarCategories)
	s.sidebarCategories = map[string]map[string]*model.SidebarCategoryWithChannels{}
	s.report = &model.PerformanceReport{}
	clear(s.drafts)
	s.drafts = map[string]map[string]*model.Draft{}
	clear(s.channelBookmarks)
	s.channelBookmarks = map[string]*model.ChannelBookmarkWithFileInfo{}
	clear(s.scheduledPosts)
	s.scheduledPosts = map[string]map[string][]*model.ScheduledPost{}
}

func (s *MemStore) setupQueues(config *Config) error {
	var err error
	s.postsQueue, err = NewCQueue[model.Post](config.MaxStoredPosts)
	if err != nil {
		return fmt.Errorf("memstore: post queue creation failed %w", err)
	}

	s.usersQueue, err = NewCQueue[model.User](config.MaxStoredUsers)
	if err != nil {
		return fmt.Errorf("memstore: users queue creation failed %w", err)
	}

	s.channelMembersQueue, err = NewCQueue[model.ChannelMember](config.MaxStoredChannelMembers)
	if err != nil {
		return fmt.Errorf("memstore: channel members queue creation failed %w", err)
	}

	s.statusesQueue, err = NewCQueue[model.Status](config.MaxStoredStatuses)
	if err != nil {
		return fmt.Errorf("memstore: status queue creation failed %w", err)
	}

	s.threadsQueue, err = NewCQueue[model.ThreadResponse](config.MaxStoredThreads)
	if err != nil {
		return fmt.Errorf("memstore: threads queue creation failed %w", err)
	}

	s.reactionsQueue, err = NewCQueue[model.Reaction](config.MaxStoredReactions)
	if err != nil {
		return fmt.Errorf("memstore: reactions queue creation failed %w", err)
	}

	return nil
}

// Id returns the id for the stored user.
func (s *MemStore) Id() string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.user == nil {
		return ""
	}
	return s.user.Id
}

// Username returns the username for the stored user.
func (s *MemStore) Username() string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.user == nil {
		return ""
	}
	return s.user.Username
}

// Email returns the email for the stored user.
func (s *MemStore) Email() string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.user == nil {
		return ""
	}
	return s.user.Email
}

// Password returns the password for the stored user.
func (s *MemStore) Password() string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.user == nil {
		return ""
	}
	return s.user.Password
}

// ClientConfig returns the limited server configuration settings for user.
func (s *MemStore) ClientConfig() map[string]string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.clientConfig
}

// FeatureFlags returns a map of the features flags stored in the client config.
func (s *MemStore) FeatureFlags() map[string]bool {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.featureFlags
}

// Config returns the server configuration settings.
func (s *MemStore) Config() model.Config {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return *s.config
}

// SetConfig stores the given configuration settings.
func (s *MemStore) SetConfig(config *model.Config) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.config = config
}

// Set ClientConfig stores the given  limited configuration settings.
func (s *MemStore) SetClientConfig(config map[string]string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.clientConfig = config

	// Populate FF
	s.featureFlags = map[string]bool{}
	for k, v := range s.clientConfig {
		// We avoid an extra call to strings.HasPrefix by checking the returned length.
		// If the prefix matches then the returned string must be shorter.
		if ffKey := strings.TrimPrefix(k, "FeatureFlag"); len(ffKey) < len(k) {
			v, err := strconv.ParseBool(v)
			if err != nil {
				continue
			}
			s.featureFlags[ffKey] = v
		}
	}
}

// User returns the stored user.
func (s *MemStore) User() (*model.User, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.user, nil
}

// SetUser stores the given user.
func (s *MemStore) SetUser(user *model.User) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if user == nil {
		return errors.New("memstore: user should not be nil")
	}
	restorePrivateData(s.user, user)
	s.user = user
	return nil
}

// Preferences returns the preferences for the stored user.
func (s *MemStore) Preferences() (model.Preferences, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.preferences == nil {
		return nil, nil
	}
	newPref := make(model.Preferences, len(s.preferences))
	copy(newPref, s.preferences)
	return newPref, nil
}

// Preferences stores the preferences for the stored user.
func (s *MemStore) SetPreferences(preferences model.Preferences) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.preferences = preferences
	return nil
}

// Post returns the post for the given postId.
func (s *MemStore) Post(postId string) (*model.Post, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if post, ok := s.posts[postId]; ok {
		p := post.Clone()
		return p, nil
	}
	return nil, ErrPostNotFound
}

// UserForPost returns the userId for the user who created the specified post.
func (s *MemStore) UserForPost(postId string) (string, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if postId == "" {
		return "", errors.New("memstore: postId should not be empty")
	}
	if post, ok := s.posts[postId]; ok {
		return post.UserId, nil
	}
	return "", ErrPostNotFound
}

// FileInfoForPost returns the FileInfo for the specified post, if any.
func (s *MemStore) FileInfoForPost(postId string) ([]*model.FileInfo, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if postId == "" {
		return nil, errors.New("memstore: postId should not be empty")
	}
	if post, ok := s.posts[postId]; ok && post.Metadata != nil {
		return post.Metadata.Files, nil
	}
	return nil, ErrPostNotFound
}

// ChannelPosts returns all posts for the specified channel.
func (s *MemStore) ChannelPosts(channelId string) ([]*model.Post, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.channelPosts(channelId)
}

func (s *MemStore) channelPosts(channelId string) ([]*model.Post, error) {
	var channelPosts []*model.Post
	for _, post := range s.posts {
		if post.ChannelId == channelId {
			p := post.Clone()
			channelPosts = append(channelPosts, p)
		}
	}

	return channelPosts, nil
}

// ChannelPostsSorted returns all posts for specified channel, sorted by CreateAt.
func (s *MemStore) ChannelPostsSorted(channelId string, asc bool) ([]*model.Post, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	posts, err := s.channelPosts(channelId)
	if err != nil {
		return nil, err
	}
	sort.Slice(posts, func(i, j int) bool {
		if asc {
			return posts[i].CreateAt < posts[j].CreateAt
		}
		return posts[i].CreateAt > posts[j].CreateAt
	})
	return posts, nil
}

// PostsIdsSince returns a list of post ids for posts created after a specified timestamp in milliseconds.
func (s *MemStore) PostsIdsSince(ts int64) ([]string, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var postsIds []string
	for _, post := range s.posts {
		if post.CreateAt > ts {
			postsIds = append(postsIds, post.Id)
		}
	}
	return postsIds, nil
}

// UsersIdsForPostsIds returns a list of user ids that created the specified
// posts.
func (s *MemStore) UsersIdsForPostsIds(postIds []string) ([]string, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var users map[string]bool
	for _, postId := range postIds {
		if post, ok := s.posts[postId]; ok && users[post.UserId] {
			users[post.UserId] = true
		}
	}
	var i int
	userIds := make([]string, len(users))
	for id := range users {
		userIds[i] = id
		i++
	}
	return userIds, nil
}

// SetPost stores the given post.
func (s *MemStore) SetPost(post *model.Post) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if post == nil {
		return errors.New("memstore: post should not be nil")
	}

	if post.Id == "" {
		return errors.New("memstore: post id should not be empty")
	}

	// Avoid storing deleted posts.
	if post.DeleteAt > 0 {
		return nil
	}

	// We get an element from the queue and check if we have it in the map and
	// if it points to the same memory location. If so, we delete it since it means the queue is full.
	// This is done to keep the data pointed by the map consistent with the data stored in the queue.
	p := s.postsQueue.Get()
	if pp, ok := s.posts[p.Id]; ok && pp == p {
		delete(s.posts, p.Id)
	}
	post.ShallowCopy(p)
	s.posts[post.Id] = p

	return nil
}

// DeletePost deletes the specified post.
func (s *MemStore) DeletePost(postId string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.posts, postId)
	return nil
}

// SetPosts stores the given posts.
func (s *MemStore) SetPosts(posts []*model.Post) error {
	if len(posts) == 0 {
		return errors.New("memstore: posts should not be nil or empty")
	}
	for _, post := range posts {
		if err := s.SetPost(post); err != nil {
			return err
		}
	}
	return nil
}

// Channel returns the channel for the given channelId.
func (s *MemStore) Channel(channelId string) (*model.Channel, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if channel, ok := s.channels[channelId]; ok {
		channelCopy := *channel
		return &channelCopy, nil
	}
	return nil, nil
}

// SetChannel stores the given channel.
func (s *MemStore) SetChannel(channel *model.Channel) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if channel == nil {
		return errors.New("memstore: channel should not be nil")
	}
	s.channels[channel.Id] = channel
	return nil
}

// GetCurrentChannel returns the channel the user is currently viewing.
func (s *MemStore) CurrentChannel() (*model.Channel, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.currentChannel == nil {
		return nil, ErrChannelNotFound
	}
	chanCopy := *s.currentChannel
	return &chanCopy, nil
}

// SetCurrentChannel stores the channel the user is currently viewing.
func (s *MemStore) SetCurrentChannel(channel *model.Channel) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if channel == nil {
		return errors.New("memstore: channel should not be nil")
	}
	s.currentChannel = channel
	return nil
}

// Channels returns all the channels for a team.
// This means no DM/GM channels are returned.
func (s *MemStore) Channels(teamId string) ([]model.Channel, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var channels []model.Channel
	for _, channel := range s.channels {
		if channel.TeamId == teamId {
			channels = append(channels, *channel)
		}
	}
	return channels, nil
}

// SetChannels adds the given channels to the store.
func (s *MemStore) SetChannels(channels []*model.Channel) error {
	if channels == nil {
		return errors.New("memstore: channels should not be nil")
	}
	for _, channel := range channels {
		if err := s.SetChannel(channel); err != nil {
			return err
		}
	}
	return nil
}

// SetChannelView marks the given channel as viewed and updates the store with the
// current timestamp.
func (s *MemStore) SetChannelView(channelId string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if len(channelId) == 0 {
		return errors.New("memstore: channelId should not be empty")
	}

	s.channelViews[channelId] = time.Now().Unix() * 1000

	return nil
}

// ChannelView returns the timestamp of the last view for the given channelId.
func (s *MemStore) ChannelView(channelId string) (int64, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if len(channelId) == 0 {
		return 0, errors.New("memstore: channelId should not be empty")
	}

	return s.channelViews[channelId], nil
}

// ChannelStats returns statistics for the given channelId.
func (s *MemStore) ChannelStats(channelId string) (*model.ChannelStats, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if channelId == "" {
		return nil, errors.New("memstore: channelId should not be empty")
	}

	return s.channelStats[channelId], nil
}

// SetChannelStats stores statistics for the given channelId.
func (s *MemStore) SetChannelStats(channelId string, stats *model.ChannelStats) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if channelId == "" {
		return errors.New("memstore: channelId should not be empty")
	}

	s.channelStats[channelId] = stats

	return nil
}

// Team returns the team for the given teamId.
func (s *MemStore) Team(teamId string) (*model.Team, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if team, ok := s.teams[teamId]; ok {
		return team, nil
	}
	return nil, nil
}

// SetTeam stores the given team.
func (s *MemStore) SetTeam(team *model.Team) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.teams[team.Id] = team
	return nil
}

// GetCurrentTeam returns the currently selected team for the user.
func (s *MemStore) CurrentTeam() (*model.Team, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.currentTeam == nil {
		return nil, nil
	}
	teamCopy := *s.currentTeam
	return &teamCopy, nil
}

// SetCurrentTeam sets the currently selected team for the user.
func (s *MemStore) SetCurrentTeam(team *model.Team) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if team == nil {
		return errors.New("memstore: team should not be nil")
	}
	s.currentTeam = team
	return nil
}

// Teams returns the teams a user belong to.
func (s *MemStore) Teams() ([]model.Team, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	teams := make([]model.Team, len(s.teams))
	i := 0
	for _, team := range s.teams {
		teams[i] = *team
		i++
	}
	return teams, nil
}

// SetTeams stores the given teams.
func (s *MemStore) SetTeams(teams []*model.Team) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.teams = make(map[string]*model.Team)
	for _, team := range teams {
		s.teams[team.Id] = team
	}
	return nil
}

// SetChannelMembers stores the given channel members in the store.
func (s *MemStore) SetChannelMembers(channelMembers model.ChannelMembers) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if channelMembers == nil {
		return errors.New("memstore: channelMembers should not be nil")
	}

	cms := channelMembers
	for i := range cms {
		cm := &cms[i]
		if s.channelMembers == nil {
			s.channelMembers = make(map[string]map[string]*model.ChannelMember)
		}
		if s.channelMembers[cm.ChannelId] == nil {
			s.channelMembers[cm.ChannelId] = make(map[string]*model.ChannelMember)
		}

		// We get an element from the queue and check if we have it in the map and
		// if it points to the same memory location. If so, we delete it since it means the queue is full.
		// This is done to keep the data pointed by the map consistent with the data stored in the queue.
		c := s.channelMembersQueue.Get()
		if s.channelMembers[c.ChannelId] != nil {
			if cc, ok := s.channelMembers[c.ChannelId][c.UserId]; ok && cc == c {
				delete(s.channelMembers[c.ChannelId], c.UserId)
			}
		}
		*c = *cm
		s.channelMembers[cm.ChannelId][cm.UserId] = c
	}

	return nil
}

// ChannelMembers returns a list of members for the specified channel.
func (s *MemStore) ChannelMembers(channelId string) (model.ChannelMembers, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	channelMembers := model.ChannelMembers{}
	for key := range s.channelMembers[channelId] {
		channelMembers = append(channelMembers, *s.channelMembers[channelId][key])
	}
	return channelMembers, nil
}

// SetChannelMember stores the given channel member.
func (s *MemStore) SetChannelMember(channelId string, channelMember *model.ChannelMember) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if channelMember == nil {
		return errors.New("memstore: channelMember should not be nil")
	}

	if channelId != channelMember.ChannelId {
		return errors.New("memstore: channelMember is not valid")
	}

	if s.channelMembers[channelId] == nil {
		s.channelMembers[channelId] = map[string]*model.ChannelMember{}
	}

	// We get an element from the queue and check if we have it in the map and
	// if it points to the same memory location. If so, we delete it since it means the queue is full.
	// This is done to keep the data pointed by the map consistent with the data stored in the queue.
	cm := s.channelMembersQueue.Get()
	if s.channelMembers[cm.ChannelId] != nil {
		if cc, ok := s.channelMembers[cm.ChannelId][cm.UserId]; ok && cc == cm {
			delete(s.channelMembers[cm.ChannelId], cm.UserId)
		}
	}

	*cm = *channelMember
	s.channelMembers[channelId][channelMember.UserId] = cm

	return nil
}

// ChannelMember returns the channel member for the given channelId and userId.
func (s *MemStore) ChannelMember(channelId, userId string) (model.ChannelMember, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var cm model.ChannelMember
	if s.channelMembers[channelId][userId] != nil {
		cm = *s.channelMembers[channelId][userId]
	}
	return cm, nil
}

// RemoveChannelMember removes the channel member for the specified channel and user.
func (s *MemStore) RemoveChannelMember(channelId string, userId string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.channelMembers[channelId], userId)
	return nil
}

// RemoveTeamMember removes the team member for the specified team and user..
func (s *MemStore) RemoveTeamMember(teamId string, userId string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.teamMembers[teamId], userId)
	return nil
}

// SetTeamMember stores the given team member.
func (s *MemStore) SetTeamMember(teamId string, teamMember *model.TeamMember) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if teamMember == nil {
		return errors.New("memstore: teamMember should not be nil")
	}
	if s.teamMembers[teamId] == nil {
		s.teamMembers[teamId] = map[string]*model.TeamMember{}
	}
	s.teamMembers[teamId][teamMember.UserId] = teamMember
	return nil
}

// SetTeamMembers stores the given team members.
func (s *MemStore) SetTeamMembers(teamId string, teamMembers []*model.TeamMember) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.teamMembers[teamId] = map[string]*model.TeamMember{}
	for _, m := range teamMembers {
		s.teamMembers[teamId][m.UserId] = m
	}

	return nil
}

// IsTeamMember returns whether the user is part of the team.
func (s *MemStore) IsTeamMember(teamId, userId string) bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	_, isMember := s.teamMembers[teamId][userId]
	return isMember
}

// TeamMember returns the team member for the given teamId and userId.
func (s *MemStore) TeamMember(teamId, userId string) (model.TeamMember, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var tm model.TeamMember
	if s.teamMembers[teamId][userId] != nil {
		tm = *s.teamMembers[teamId][userId]
	}
	return tm, nil
}

// SetEmojis stores the given emojis.
func (s *MemStore) SetEmojis(emoji []*model.Emoji) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.emojis = emoji
	return nil
}

// SetReaction stores the given reaction.
func (s *MemStore) SetReaction(reaction *model.Reaction) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// We get an element from the queue and check if we have it in the map and
	// if it points to the same memory location. If so, we delete it since it means the queue is full.
	// This is done to keep the data pointed by the map consistent with the data stored in the queue.
	r := s.reactionsQueue.Get()
	if rs, ok := s.reactions[r.PostId]; ok && rs[len(s.reactions[r.PostId])-1] == r {
		rs[len(s.reactions[r.PostId])-1] = nil
		s.reactions[r.PostId] = rs[:len(rs)-1]
	}

	*r = *reaction
	s.reactions[r.PostId] = append(s.reactions[r.PostId], r)

	return nil
}

// Reactions returns the reactions for the specified post.
func (s *MemStore) Reactions(postId string) ([]model.Reaction, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	reactions := make([]model.Reaction, 0, len(s.reactions))
	for _, reaction := range s.reactions[postId] {
		reactions = append(reactions, *reaction)
	}

	return reactions, nil
}

// DeleteReaction deletes the given reaction.
// It returns whether or not the reaction was deleted.
func (s *MemStore) DeleteReaction(reaction *model.Reaction) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if reaction == nil {
		return false, errors.New("memstore: reaction should not be nil")
	}
	reactions := s.reactions[reaction.PostId]
	for i, r := range reactions {
		if *r == *reaction {
			reactions[i] = reactions[len(reactions)-1]
			reactions[len(reactions)-1] = nil // Allow element to be garbage collected.
			s.reactions[reaction.PostId] = reactions[:len(reactions)-1]
			return true, nil
		}
	}
	return false, nil
}

// GetUser returns the user for the given userId.
func (s *MemStore) GetUser(userId string) (model.User, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var user model.User

	if len(userId) == 0 {
		return user, errors.New("memstore: userId should not be empty")
	}

	if u, ok := s.users[userId]; ok {
		user = *u
	}

	return user, nil
}

// Users returns all users in the store.
func (s *MemStore) Users() ([]model.User, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	users := make([]model.User, 0, len(s.users))
	for _, u := range s.users {
		users = append(users, *u)
	}

	return users, nil
}

// SetUsers stores the given users.
func (s *MemStore) SetUsers(users []*model.User) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	for _, user := range users {
		// We get an element from the queue and check if we have it in the map and
		// if it points to the same memory location. If so, we delete it since it means the queue is full.
		// This is done to keep the data pointed by the map consistent with the data stored in the queue.
		u := s.usersQueue.Get()
		if uu, ok := s.users[u.Id]; ok && uu == u {
			delete(s.users, u.Id)
		}

		*u = *user
		s.users[user.Id] = u
	}
	return nil
}

// Status returns the status for the given userId.
func (s *MemStore) Status(userId string) (model.Status, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var status model.Status

	if len(userId) == 0 {
		return status, errors.New("memstore: userId should not be empty")
	}

	if st, ok := s.statuses[userId]; ok {
		status = *st
	}
	return status, nil
}

// SetStatus stores the status for the given userId.
func (s *MemStore) SetStatus(userId string, status *model.Status) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if len(userId) == 0 {
		return errors.New("memstore: userId should not be empty")
	}

	if status == nil {
		return errors.New("memstore: status should not be nil")
	}

	if userId != status.UserId {
		return errors.New("memstore: status is not valid")
	}

	// We get an element from the queue and check if we have it in the map and
	// if it points to the same memory location. If so, we delete it since it means the queue is full.
	// This is done to keep the data pointed by the map consistent with the data stored in the queue.
	st := s.statusesQueue.Get()
	if ss, ok := s.statuses[st.UserId]; ok && ss == st {
		delete(s.statuses, st.UserId)
	}

	*st = *status
	s.statuses[userId] = st

	return nil
}

// SetRoles stores the given roles.
func (s *MemStore) SetRoles(roles []*model.Role) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.roles = make(map[string]*model.Role)
	for _, role := range roles {
		s.roles[role.Id] = role
	}
	return nil
}

// Roles returns the roles of the user.
func (s *MemStore) Roles() ([]model.Role, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	roles := make([]model.Role, len(s.roles))
	i := 0
	for _, role := range s.roles {
		roles[i] = *role
		i++
	}
	return roles, nil
}

// SetLicense stores the given license in the store.
func (s *MemStore) SetLicense(license map[string]string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.license = license
	return nil
}

// ProfileImageLastUpdated returns the etag returned by the server when first
// fetched, which is the last time the picture was updated, or zero if the
// image is not stored.
func (s *MemStore) ProfileImageLastUpdated(userId string) (int, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if userId == "" {
		return 0, errors.New("memstore: userId should not be empty")
	}

	return s.profileImages[userId], nil
}

// SetProfileImage sets as stored the profile image for the given user.
func (s *MemStore) SetProfileImage(userId string, lastPictureUpdate int) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if userId == "" {
		return errors.New("memstore: userId should not be empty")
	}

	s.profileImages[userId] = lastPictureUpdate
	return nil
}

// ServerVersion returns the server version string.
func (s *MemStore) ServerVersion() (string, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.serverVersion, nil
}

// SetServerVersion stores the given server version.
func (s *MemStore) SetServerVersion(version string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.serverVersion = version
	return nil
}

// SetThread stores the given thread reponse.
func (s *MemStore) SetThread(thread *model.ThreadResponse) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if thread == nil {
		return errors.New("memstore: thread should not be nil")
	}

	// We get an element from the queue and check if we have it in the map and
	// if it points to the same memory location. If so, we delete it since it means the queue is full.
	// This is done to keep the data pointed by the map consistent with the data stored in the queue.
	t := s.threadsQueue.Get()
	if tt, ok := s.threads[t.PostId]; ok && tt == t {
		delete(s.threads, t.PostId)
	}
	cloneThreadResponse(thread, t)
	s.threads[thread.PostId] = t

	return nil
}

// SetThreads stores the given thread response as a thread.
func (s *MemStore) SetThreads(trs []*model.ThreadResponse) error {
	if len(trs) == 0 {
		return nil
	}
	for _, tr := range trs {
		if err := s.SetThread(tr); err != nil {
			return err
		}
	}
	return nil
}

func (s *MemStore) SetCategories(teamID string, sidebarCategories *model.OrderedSidebarCategories) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	teamCat := make(map[string]*model.SidebarCategoryWithChannels)
	for _, cat := range sidebarCategories.Categories {
		teamCat[cat.SidebarCategory.Id] = cat
	}
	s.sidebarCategories[teamID] = teamCat
	return nil
}

func (s *MemStore) getThreads(unreadOnly bool) ([]*model.ThreadResponse, error) {
	var threads []*model.ThreadResponse
	for _, thread := range s.threads {
		if unreadOnly && thread.UnreadReplies == 0 {
			continue
		}
		threads = append(threads, cloneThreadResponse(thread, &model.ThreadResponse{}))
	}
	return threads, nil
}

// ThreadsSorted returns all threads, sorted by LastReplyAt
func (s *MemStore) ThreadsSorted(unreadOnly, asc bool) ([]*model.ThreadResponse, error) {
	s.lock.RLock()
	threads, err := s.getThreads(unreadOnly)
	s.lock.RUnlock()
	if err != nil {
		return nil, err
	}
	sort.Slice(threads, func(i, j int) bool {
		if asc {
			return threads[i].LastReplyAt < threads[j].LastReplyAt
		}
		return threads[i].LastReplyAt > threads[j].LastReplyAt
	})
	return threads, nil
}

// cloneThreadResponse copies the given source threadResponse into the given destination
// ThreadResponse.
// It also returns the destination ThreadReponse. This lets us:
// 1. directly call this function in the parent's return statement, i.e. return cloneThreadResponse(...)
// 2. use inplace, e.g. append(threads, cloneThreadResponse(thread, &model.ThreadResponse{}))
// 3. pass the threadResponse object in the case where we need to update an existing object
func cloneThreadResponse(src *model.ThreadResponse, dst *model.ThreadResponse) *model.ThreadResponse {
	dst.PostId = src.PostId
	dst.ReplyCount = src.ReplyCount
	dst.LastReplyAt = src.LastReplyAt
	dst.LastViewedAt = src.LastViewedAt
	dst.Participants = src.Participants
	if src.Post != nil {
		if dst.Post == nil {
			dst.Post = &model.Post{}
		}
		src.Post.ShallowCopy(dst.Post)
	}
	dst.UnreadReplies = src.UnreadReplies
	dst.UnreadMentions = src.UnreadMentions
	return dst
}

// MarkAllThreadsInTeamAsRead marks all threads in the given team as read
func (s *MemStore) MarkAllThreadsInTeamAsRead(teamId string) error {
	s.lock.RLock()
	threads, err := s.getThreads(false)
	s.lock.RUnlock()
	if err != nil {
		return err
	}
	now := model.GetMillis()
	for _, thread := range threads {
		ch, _ := s.Channel(thread.Post.ChannelId)
		if ch == nil || ch.TeamId != teamId {
			// We do our best to keep the local store threads in sync
			// If we don't have data in local store, we skip it
			// instead of making API calls or failing the whole
			// operation
			continue
		}
		thread.UnreadMentions = 0
		thread.UnreadReplies = 0
		thread.LastViewedAt = now
	}
	return s.SetThreads(threads)
}

// Thread returns the thread for the given the threadId.
func (s *MemStore) Thread(threadId string) (*model.ThreadResponse, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if thread, ok := s.threads[threadId]; ok {
		return cloneThreadResponse(thread, &model.ThreadResponse{}), nil
	}
	return nil, ErrThreadNotFound
}

// PostsWithAckRequests returns IDs of the posts that asked for acknowledgment.
func (s *MemStore) PostsWithAckRequests() ([]string, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var ids []string
	for _, p := range s.posts {
		if p.Metadata != nil && p.Metadata.Priority != nil && p.Metadata.Priority.RequestedAck != nil && *p.Metadata.Priority.RequestedAck {
			ids = append(ids, p.Id)
		}
	}

	return ids, nil
}

func (s *MemStore) SetPerformanceReport(report *model.PerformanceReport) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.report = report
}

func (s *MemStore) PerformanceReport() (*model.PerformanceReport, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.report == nil {
		return nil, nil
	}

	report := &model.PerformanceReport{
		Version:  s.report.Version,
		ClientID: s.report.ClientID,
		Start:    s.report.Start,
		End:      s.report.End,
	}

	if s.report.Labels != nil {
		report.Labels = make(map[string]string)
	}
	for k, v := range s.report.Labels {
		report.Labels[k] = v
	}

	if s.report.Histograms != nil {
		report.Histograms = make([]*model.MetricSample, len(s.report.Histograms))
	}
	for i, h := range s.report.Histograms {
		report.Histograms[i] = &model.MetricSample{
			Metric:    h.Metric,
			Value:     h.Value,
			Timestamp: h.Timestamp,
			Labels:    h.Labels,
		}
	}

	if s.report.Counters != nil {
		report.Counters = make([]*model.MetricSample, len(s.report.Counters))
	}
	for i, h := range s.report.Counters {
		report.Counters[i] = &model.MetricSample{
			Metric:    h.Metric,
			Value:     h.Value,
			Timestamp: h.Timestamp,
			Labels:    h.Labels,
		}
	}

	return report, nil
}

// SetDraft stores the draft for the given teamId, and channelId or rootId.
func (s *MemStore) SetDraft(teamId, id string, draft *model.Draft) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if draft == nil {
		return errors.New("memstore: draft should not be nil")
	}

	if s.drafts[teamId] == nil {
		s.drafts[teamId] = map[string]*model.Draft{}
	}

	s.drafts[teamId][id] = draft
	return nil
}

// SetDrafts stores the given drafts.
func (s *MemStore) SetDrafts(teamId string, drafts []*model.Draft) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.drafts[teamId] = map[string]*model.Draft{}
	for _, d := range drafts {
		rootID := d.RootId
		// Note: rootID should never be empty.
		// Need to verify if this is the right logic.
		if rootID == "" {
			rootID = d.ChannelId
		}
		s.drafts[teamId][rootID] = d
	}

	return nil
}

// ChannelBookmarks returns all bookmarks for the specified channel.
func (s *MemStore) ChannelBookmarks(channelId string) []*model.ChannelBookmarkWithFileInfo {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var bookmarks []*model.ChannelBookmarkWithFileInfo
	for _, b := range s.channelBookmarks {
		if b.ChannelId == channelId {
			bookmarks = append(bookmarks, b)
		}
	}
	return bookmarks
}

// SetChannelBookmarks stores the given bookmarks.
func (s *MemStore) SetChannelBookmarks(bookmarks []*model.ChannelBookmarkWithFileInfo) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, bookmark := range bookmarks {
		if bookmark == nil {
			return errors.New("memstore: bookmark should not be nil")
		}
		s.channelBookmarks[bookmark.Id] = bookmark
	}

	return nil
}

// AddChannelBookmark stores the bookmark.
func (s *MemStore) AddChannelBookmark(bookmark *model.ChannelBookmarkWithFileInfo) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if bookmark == nil {
		return errors.New("memstore: bookmark should not be nil")
	}

	s.channelBookmarks[bookmark.Id] = bookmark
	return nil
}

// UpdateChannelBookmark updates a given bookmark.
func (s *MemStore) UpdateChannelBookmark(bookmark *model.ChannelBookmarkWithFileInfo) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if bookmark == nil {
		return errors.New("memstore: bookmark should not be nil")
	}

	if s.channelBookmarks[bookmark.Id] == nil {
		return errors.New("memstore: bookmark not found")
	}

	s.channelBookmarks[bookmark.Id] = bookmark

	return nil
}

// DeleteChannelBookmark deletes a given bookmark.
func (s *MemStore) DeleteChannelBookmark(bookmarkId string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if bookmarkId == "" {
		return errors.New("memstore: bookmarkId should not be empty")
	}

	if s.channelBookmarks[bookmarkId] == nil {
		return errors.New("memstore: bookmark not found")
	}

	delete(s.channelBookmarks, bookmarkId)
	return nil
}

func (s *MemStore) SetScheduledPost(teamId string, scheduledPost *model.ScheduledPost) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if scheduledPost == nil {
		return errors.New("memstore: scheduled post should not be nil")
	}

	if s.scheduledPosts == nil {
		s.scheduledPosts = map[string]map[string][]*model.ScheduledPost{}
	}

	if s.scheduledPosts[teamId] == nil {
		s.scheduledPosts[teamId] = map[string][]*model.ScheduledPost{}
	}

	channelOrThreadId := scheduledPost.ChannelId
	if scheduledPost.RootId != "" {
		channelOrThreadId = scheduledPost.RootId
	}

	s.scheduledPosts[teamId][channelOrThreadId] = append(s.scheduledPosts[teamId][channelOrThreadId], scheduledPost)
	return nil
}

func (s *MemStore) DeleteScheduledPost(scheduledPost *model.ScheduledPost) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for teamId := range s.scheduledPosts {
		channelOrThreadId := scheduledPost.ChannelId
		if scheduledPost.RootId != "" {
			channelOrThreadId = scheduledPost.RootId
		}

		// find index of scheduledPost in s.scheduledPosts[teamId][channelOrThreadId] and if found, delete it
		for i, sp := range s.scheduledPosts[teamId][channelOrThreadId] {
			if sp.Id == scheduledPost.Id {
				s.scheduledPosts[teamId][channelOrThreadId] = append(s.scheduledPosts[teamId][channelOrThreadId][:i], s.scheduledPosts[teamId][channelOrThreadId][i+1:]...)
				break
			}
		}
	}
}

func (s *MemStore) UpdateScheduledPost(teamId string, scheduledPost *model.ScheduledPost) {
	s.lock.Lock()
	defer s.lock.Unlock()

	channelOrThreadId := scheduledPost.ChannelId
	if scheduledPost.RootId != "" {
		channelOrThreadId = scheduledPost.RootId
	}

	if _, ok := s.scheduledPosts[teamId]; !ok {
		s.scheduledPosts[teamId] = map[string][]*model.ScheduledPost{
			channelOrThreadId: {scheduledPost},
		}
		return
	}

	for i := range s.scheduledPosts[teamId][channelOrThreadId] {
		if s.scheduledPosts[teamId][channelOrThreadId][i].Id == scheduledPost.Id {
			s.scheduledPosts[teamId][channelOrThreadId][i] = scheduledPost
			break
		}
	}
}

func (s *MemStore) SetCPAFields(fields []*model.PropertyField) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.customAttributeFields = fields
	return nil
}

func (s *MemStore) GetCPAFields() []*model.PropertyField {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.customAttributeFields
}

func (s *MemStore) SetCPAValues(userID string, values map[string]json.RawMessage) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.customAttributeValues[userID] = values
	return nil
}

func (s *MemStore) GetCPAValues(userID string) map[string]json.RawMessage {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.customAttributeValues[userID]
}
