// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package memstore

import (
	"errors"
	"sort"
	"sync"

	"github.com/mattermost/mattermost-server/v5/model"
)

// MemStore is a simple implementation of MutableUserStore
// which holds all data in memory.
type MemStore struct {
	lock           sync.RWMutex
	user           *model.User
	preferences    *model.Preferences
	config         *model.Config
	emojis         []*model.Emoji
	posts          map[string]*model.Post
	teams          map[string]*model.Team
	channels       map[string]*model.Channel
	channelMembers map[string]map[string]*model.ChannelMember
	teamMembers    map[string]map[string]*model.TeamMember
	users          map[string]*model.User
	reactions      map[string][]*model.Reaction
	roles          map[string]*model.Role
	license        map[string]string
}

// New returns a new instance of MemStore.
func New() *MemStore {
	s := &MemStore{}
	s.Clear()
	return s
}

// Clear resets the store and removes all entries
func (s *MemStore) Clear() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.posts = map[string]*model.Post{}
	s.teams = map[string]*model.Team{}
	s.channels = map[string]*model.Channel{}
	s.channelMembers = map[string]map[string]*model.ChannelMember{}
	s.teamMembers = map[string]map[string]*model.TeamMember{}
	s.users = map[string]*model.User{}
	s.reactions = map[string][]*model.Reaction{}
	s.roles = map[string]*model.Role{}
}

func (s *MemStore) Id() string {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.user == nil {
		return ""
	}
	return s.user.Id
}

func (s *MemStore) Username() string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.user == nil {
		return ""
	}
	return s.user.Username
}

func (s *MemStore) Email() string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.user == nil {
		return ""
	}
	return s.user.Email
}

func (s *MemStore) Password() string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.user == nil {
		return ""
	}
	return s.user.Password
}

func (s *MemStore) Config() model.Config {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return *s.config
}

func (s *MemStore) SetConfig(config *model.Config) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.config = config
}

func (s *MemStore) User() (*model.User, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	return s.user, nil
}

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

func (s *MemStore) Preferences() (model.Preferences, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if s.preferences == nil {
		return nil, nil
	}
	newPref := make(model.Preferences, len(*s.preferences))
	copy(newPref, *s.preferences)
	return newPref, nil
}

func (s *MemStore) SetPreferences(preferences *model.Preferences) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.preferences = preferences
	return nil
}

func (s *MemStore) Post(postId string) (*model.Post, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if post, ok := s.posts[postId]; ok {
		return post, nil
	}
	return nil, nil
}

func (s *MemStore) ChannelPosts(channelId string) ([]*model.Post, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var channelPosts []*model.Post
	for _, post := range s.posts {
		if post.ChannelId == channelId {
			channelPosts = append(channelPosts, post)
		}
	}

	return channelPosts, nil
}

func (s *MemStore) ChannelPostsSorted(channelId string, asc bool) ([]*model.Post, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	posts, err := s.ChannelPosts(channelId)
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

func (s *MemStore) PostsSince(ts int64) ([]model.Post, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	var posts []model.Post
	for _, post := range s.posts {
		if post.CreateAt > ts {
			posts = append(posts, *post)
		}
	}
	return posts, nil
}

func (s *MemStore) SetPost(post *model.Post) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if post == nil {
		return errors.New("memstore: post should not be nil")
	}
	s.posts[post.Id] = post
	return nil
}

func (s *MemStore) DeletePost(postId string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.posts, postId)
	return nil
}

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

func (s *MemStore) Channel(channelId string) (*model.Channel, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	if channel, ok := s.channels[channelId]; ok {
		return channel, nil
	}
	return nil, nil
}

func (s *MemStore) SetChannel(channel *model.Channel) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if channel == nil {
		return errors.New("memstore: channel should not be nil")
	}
	s.channels[channel.Id] = channel
	return nil
}

// Channels return all the channels for a team.
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

func (s *MemStore) Team(teamId string) (*model.Team, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if team, ok := s.teams[teamId]; ok {
		return team, nil
	}
	return nil, nil
}

func (s *MemStore) SetTeam(team *model.Team) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.teams[team.Id] = team
	return nil
}

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
func (s *MemStore) SetChannelMembers(channelMembers *model.ChannelMembers) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if channelMembers == nil {
		return errors.New("memstore: channelMembers should not be nil")
	}

	for _, member := range *channelMembers {
		member := member
		// Initialize the maps as necessary.
		if s.channelMembers == nil {
			s.channelMembers = make(map[string]map[string]*model.ChannelMember)
		}
		if s.channelMembers[member.ChannelId] == nil {
			s.channelMembers[member.ChannelId] = make(map[string]*model.ChannelMember)
		}
		// Set value.
		s.channelMembers[member.ChannelId][member.UserId] = &member
	}

	return nil
}

func (s *MemStore) ChannelMembers(channelId string) (*model.ChannelMembers, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	channelMembers := model.ChannelMembers{}
	for key := range s.channelMembers[channelId] {
		channelMembers = append(channelMembers, *s.channelMembers[channelId][key])
	}
	return &channelMembers, nil
}

func (s *MemStore) SetChannelMember(channelId string, channelMember *model.ChannelMember) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if channelMember == nil {
		return errors.New("memstore: channelMember should not be nil")
	}
	if s.channelMembers[channelId] == nil {
		s.channelMembers[channelId] = map[string]*model.ChannelMember{}
	}
	s.channelMembers[channelId][channelMember.UserId] = channelMember
	return nil
}

func (s *MemStore) ChannelMember(channelId, userId string) (model.ChannelMember, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var cm model.ChannelMember
	if s.channelMembers[channelId][userId] != nil {
		cm = *s.channelMembers[channelId][userId]
	}
	return cm, nil
}

func (s *MemStore) RemoveChannelMember(channelId string, userId string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.channelMembers[channelId], userId)
	return nil
}

func (s *MemStore) RemoveTeamMember(teamId string, userId string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.teamMembers[teamId], userId)
	return nil
}

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

func (s *MemStore) SetTeamMembers(teamId string, teamMembers []*model.TeamMember) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.teamMembers[teamId] = map[string]*model.TeamMember{}
	for _, m := range teamMembers {
		s.teamMembers[teamId][m.UserId] = m
	}

	return nil
}

func (s *MemStore) TeamMember(teamId, userId string) (model.TeamMember, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var tm model.TeamMember
	if s.teamMembers[teamId][userId] != nil {
		tm = *s.teamMembers[teamId][userId]
	}
	return tm, nil
}

func (s *MemStore) SetEmojis(emoji []*model.Emoji) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.emojis = emoji
	return nil
}

func (s *MemStore) SetReactions(postId string, reactions []*model.Reaction) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.reactions[postId] = reactions
	return nil
}

func (s *MemStore) SetReaction(reaction *model.Reaction) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.reactions[reaction.PostId] = append(s.reactions[reaction.PostId], reaction)

	return nil
}

func (s *MemStore) Reactions(postId string) ([]model.Reaction, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	var reactions []model.Reaction
	for _, reaction := range s.reactions[postId] {
		reactions = append(reactions, *reaction)
	}

	return reactions, nil
}

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
			s.reactions[reaction.PostId] = reactions[:len(reactions)-1]
			return true, nil
		}
	}
	return false, nil
}

func (s *MemStore) Users() ([]*model.User, error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	users := make([]*model.User, len(s.users))
	i := 0
	for _, user := range s.users {
		users[i] = user
		i++
	}
	return users, nil
}

func (s *MemStore) SetUsers(users []*model.User) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	//s.users = make(map[string]*model.User)
	for _, user := range users {
		s.users[user.Id] = user
	}
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

// Roles return the roles of the user.
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
