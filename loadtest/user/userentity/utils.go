// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"io"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost-server/v6/model"
)

const (
	graphQLEndpoint = "/graphql"
)

func getGQLURL(baseURL string) string {
	return baseURL + model.APIURLSuffixV5 + graphQLEndpoint
}

func postsMapToSlice(postsMap map[string]*model.Post) []*model.Post {
	posts := make([]*model.Post, len(postsMap))
	i := 0
	for _, v := range postsMap {
		posts[i] = v
		i++
	}
	return posts
}

func postListToSlice(list *model.PostList) []*model.Post {
	posts := make([]*model.Post, len(list.Order))
	for i, id := range list.Order {
		posts[i] = list.Posts[id]
	}
	return posts
}

func closeBody(r *http.Response) {
	if r.Body != nil {
		_, _ = io.Copy(io.Discard, r.Body)
		_ = r.Body.Close()
	}
}

func convertToTypedPrefs(input []gqlPreference) model.Preferences {
	prefs := model.Preferences{}
	for _, p := range input {
		prefs = append(prefs, model.Preference{
			UserId:   p.UserID,
			Category: p.Category,
			Name:     p.Name,
			Value:    p.Value,
		})
	}
	return prefs
}

func convertToTypedRoles(input []gqlRole) []*model.Role {
	roles := []*model.Role{}
	for _, r := range input {
		roles = append(roles, &model.Role{
			Id:            r.ID,
			Name:          r.Name,
			Permissions:   r.Permissions,
			SchemeManaged: r.SchemeManaged,
			BuiltIn:       r.BuiltIn,
			CreateAt:      int64(r.CreateAt),
			DeleteAt:      int64(r.DeleteAt),
			UpdateAt:      int64(r.UpdateAt),
		})
	}
	return roles
}

func convertToRoleString(input []gqlRole) string {
	roleNames := make([]string, 0, len(input))
	for _, r := range input {
		roleNames = append(roleNames, r.Name)
	}
	return strings.Join(roleNames, " ")
}

func convertToTypedUser(input gqlUser) (*model.User, model.Preferences, []*model.Role) {
	prefs := convertToTypedPrefs(input.Preferences)
	roles := convertToTypedRoles(input.Roles)
	user := &model.User{
		Id:            input.ID,
		Username:      input.Username,
		Email:         input.Email,
		FirstName:     input.FirstName,
		LastName:      input.LastName,
		CreateAt:      int64(input.CreateAt),
		UpdateAt:      int64(input.UpdateAt),
		DeleteAt:      int64(input.DeleteAt),
		EmailVerified: input.EmailVerified,
		IsBot:         input.IsBot,
		Timezone:      input.Timezone,
		Props:         input.Props,
		NotifyProps:   input.NotifyProps,
	}

	return user, prefs, roles
}

func convertToTypedTeams(userID string, input []gqlTeamMember) ([]*model.Team, []*model.TeamMember) {
	teams := make([]*model.Team, 0, len(input))
	tms := make([]*model.TeamMember, 0, len(input))
	for _, t := range input {
		team := t.Team
		mTeam := &model.Team{
			Id:                 team.ID,
			DisplayName:        team.DisplayName,
			Name:               team.Name,
			CreateAt:           int64(team.CreateAt),
			UpdateAt:           int64(team.UpdateAt),
			DeleteAt:           int64(team.DeleteAt),
			Description:        team.Description,
			Email:              team.Email,
			Type:               team.Type,
			CompanyName:        team.CompanyName,
			AllowedDomains:     team.AllowedDomains,
			InviteId:           team.InviteId,
			LastTeamIconUpdate: team.LastTeamIconUpdate,
			AllowOpenInvite:    team.AllowOpenInvite,
		}
		if team.GroupConstrained != nil {
			mTeam.GroupConstrained = model.NewBool(*team.GroupConstrained)
		}
		if team.SchemeId != nil {
			mTeam.SchemeId = model.NewString(*team.SchemeId)
		}
		if team.PolicyId != nil {
			mTeam.PolicyID = model.NewString(*team.PolicyId)
		}
		teams = append(teams, mTeam)

		tms = append(tms, &model.TeamMember{
			UserId:      userID,
			TeamId:      team.ID,
			DeleteAt:    int64(t.DeleteAt),
			SchemeGuest: t.SchemeGuest,
			SchemeUser:  t.SchemeUser,
			SchemeAdmin: t.SchemeAdmin,
		})
	}

	return teams, tms
}

func convertToTypedChannels(input []gqlChannel) ([]*model.Channel, string) {
	channels := make([]*model.Channel, 0, len(input))
	cursor := ""
	for _, c := range input {
		ch := &model.Channel{
			Id:                c.ID,
			CreateAt:          int64(c.CreateAt),
			UpdateAt:          int64(c.UpdateAt),
			DeleteAt:          int64(c.DeleteAt),
			TeamId:            c.Team.ID,
			Type:              model.ChannelType(c.Type),
			DisplayName:       c.DisplayName,
			Name:              c.Name,
			Header:            c.Header,
			Purpose:           c.Purpose,
			LastPostAt:        c.LastPostAt,
			LastRootPostAt:    c.LastRootPostAt,
			TotalMsgCount:     c.TotalMsgCount,
			TotalMsgCountRoot: c.TotalMsgCountRoot,
			CreatorId:         c.CreatorID,
			Props:             c.Props,
		}
		if c.SchemeID != nil {
			ch.SchemeId = model.NewString(*c.SchemeID)
		}
		if c.GroupConstrained != nil {
			ch.GroupConstrained = model.NewBool(*c.GroupConstrained)
		}
		if c.Shared != nil {
			ch.Shared = model.NewBool(*c.Shared)
		}
		if c.PolicyID != nil {
			ch.PolicyID = model.NewString(*c.PolicyID)
		}
		cursor = c.Cursor
		channels = append(channels, ch)
	}
	return channels, cursor
}

func convertToTypedChannelMembers(input []gqlChannelMember) (model.ChannelMembers, string) {
	cms := make(model.ChannelMembers, 0, len(input))
	cursor := ""
	for _, cm := range input {
		cms = append(cms, model.ChannelMember{
			ChannelId:          cm.Channel.ID,
			UserId:             cm.User.ID,
			Roles:              convertToRoleString(cm.Roles),
			LastViewedAt:       int64(cm.LastViewedAt),
			MsgCount:           int64(cm.MsgCount),
			MentionCount:       int64(cm.MentionCount),
			MentionCountRoot:   int64(cm.MentionCountRoot),
			UrgentMentionCount: int64(cm.UrgentMentionCount),
			MsgCountRoot:       int64(cm.MsgCountRoot),
			NotifyProps:        cm.NotifyProps,
			LastUpdateAt:       int64(cm.LastUpdateAt),
			SchemeGuest:        cm.SchemeGuest,
			SchemeUser:         cm.SchemeUser,
			SchemeAdmin:        cm.SchemeAdmin,
		})
		cursor = cm.Cursor
	}
	return cms, cursor
}
