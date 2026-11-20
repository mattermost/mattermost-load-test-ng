// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package user

const (
	SystemSettingPolicyType = "system_setting"

	WebsocketEventSystemSettingPolicyUpdated = "system_setting_policy_updated"
)

// SystemSettingPolicyRule overrides the client config setting named in Actions
// with Value for the users matching Expression.
type SystemSettingPolicyRule struct {
	Actions    []string `json:"actions"`
	Expression string   `json:"expression"`
	Value      string   `json:"value"`
}

// SystemSettingPolicy is an access control policy of type system_setting.
type SystemSettingPolicy struct {
	ID       string                    `json:"id,omitempty"`
	Name     string                    `json:"name"`
	Type     string                    `json:"type"`
	Revision int                       `json:"revision,omitempty"`
	Roles    []string                  `json:"roles"`
	Rules    []SystemSettingPolicyRule `json:"rules"`
}

// SettingKey returns the client config key the policy overrides.
func (p *SystemSettingPolicy) SettingKey() string {
	if len(p.Rules) == 0 || len(p.Rules[0].Actions) == 0 {
		return ""
	}
	return p.Rules[0].Actions[0]
}
