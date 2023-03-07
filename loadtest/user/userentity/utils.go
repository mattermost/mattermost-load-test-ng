// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"io"
	"net/http"

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
	teams := []*model.Team{}
	tms := []*model.TeamMember{}
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
