// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import "github.com/mattermost/mattermost-server/v6/model"

type gqlRole struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Permissions   []string `json:"permissions"`
	SchemeManaged bool     `json:"schemeManaged"`
	BuiltIn       bool     `json:"builtIn"`
	CreateAt      float64  `json:"createAt"`
	DeleteAt      float64  `json:"deleteAt"`
	UpdateAt      float64  `json:"updateAt"`
}

type gqlPreference struct {
	UserID   string `json:"userId"`
	Category string `json:"category"`
	Name     string `json:"name"`
	Value    string `json:"value"`
}

type gqlUser struct {
	ID            string          `json:"id"`
	Username      string          `json:"username"`
	Email         string          `json:"email"`
	FirstName     string          `json:"firstName"`
	LastName      string          `json:"lastName"`
	CreateAt      float64         `json:"createAt"`
	UpdateAt      float64         `json:"updateAt"`
	DeleteAt      float64         `json:"deleteAt"`
	EmailVerified bool            `json:"emailVerified"`
	IsBot         bool            `json:"isBot"`
	IsGuest       bool            `json:"isGuest"`
	IsSystemAdmin bool            `json:"isSystemAdmin"`
	Timezone      model.StringMap `json:"timezone"`
	Props         model.StringMap `json:"props"`
	NotifyProps   model.StringMap `json:"notifyProps"`
	Roles         []gqlRole       `json:"roles"`
	Preferences   []gqlPreference `json:"preferences"`
}

type gqlTeamMember struct {
	Team struct {
		ID                 string  `json:"id"`
		DisplayName        string  `json:"displayName"`
		Name               string  `json:"name"`
		CreateAt           float64 `json:"createAt"`
		UpdateAt           float64 `json:"updateAt"`
		DeleteAt           float64 `json:"deleteAt"`
		Description        string  `json:"description"`
		Email              string  `json:"email"`
		Type               string  `json:"type"`
		CompanyName        string  `json:"companyName"`
		AllowedDomains     string  `json:"allowedDomains"`
		InviteId           string  `json:"inviteId"`
		LastTeamIconUpdate int64   `json:"lastTeamIconUpdate"`
		GroupConstrained   *bool   `json:"groupConstrained"`
		AllowOpenInvite    bool    `json:"allowOpenInvite"`
		SchemeId           *string `json:"schemeId"`
		PolicyId           *string `json:"policyId"`
	} `json:"team"`
	DeleteAt    float64 `json:"deleteAt"`
	SchemeGuest bool    `json:"schemeGuest"`
	SchemeUser  bool    `json:"schemeUser"`
	SchemeAdmin bool    `json:"schemeAdmin"`
}

type gqlChannel struct {
	Cursor   string  `json:"cursor"`
	ID       string  `json:"id"`
	CreateAt float64 `json:"createAt"`
	UpdateAt float64 `json:"updateAt"`
	DeleteAt float64 `json:"deleteAt"`
	Team     struct {
		ID string `json:"id"`
	} `json:"team"`
	Type              string         `json:"type"`
	DisplayName       string         `json:"displayName"`
	Name              string         `json:"name"`
	Header            string         `json:"header"`
	Purpose           string         `json:"purpose"`
	LastPostAt        int64          `json:"lastPostAt"`
	LastRootPostAt    int64          `json:"lastRootPostAt"`
	TotalMsgCount     int64          `json:"totalMsgCount"`
	TotalMsgCountRoot int64          `json:"totalMsgCountRoot"`
	CreatorID         string         `json:"creatorId"`
	SchemeID          *string        `json:"schemeId"`
	GroupConstrained  *bool          `json:"groupConstrained"`
	Shared            *bool          `json:"shared"`
	Props             map[string]any `json:"props"`
	PolicyID          *string        `json:"policyId"`
}

type gqlChannelMember struct {
	Cursor             string          `json:"cursor"`
	Channel            gqlChannel      `json:"channel"`
	Roles              []gqlRole       `json:"roles"`
	User               gqlUser         `json:"user"`
	LastViewedAt       float64         `json:"lastViewedAt"`
	LastUpdateAt       float64         `json:"lastUpdateAt"`
	MsgCount           float64         `json:"msgCount"`
	MentionCount       float64         `json:"mentionCount"`
	MentionCountRoot   float64         `json:"mentionCountRoot"`
	UrgentMentionCount float64         `json:"urgentMentionCount"`
	MsgCountRoot       float64         `json:"msgCountRoot"`
	NotifyProps        model.StringMap `json:"notifyProps"`
	SchemeGuest        bool            `json:"schemeGuest"`
	SchemeUser         bool            `json:"schemeUser"`
	SchemeAdmin        bool            `json:"schemeAdmin"`
}
