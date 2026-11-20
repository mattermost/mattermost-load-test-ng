// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"github.com/mattermost/mattermost-load-test-ng/defaults"
)

// Config holds information needed to run a SimulController.
type Config struct {
	// The minium amount of time (in milliseconds) the controlled users
	// will wait between actions.
	MinIdleTimeMs int `default:"1000" validate:"range:[0,]"`
	// The average amount of time (in milliseconds) the controlled users
	// will wait between actions.
	AvgIdleTimeMs int `default:"20000" validate:"range:($MinIdleTimeMs,]"`

	// The percentage of root posts that are marked as urgent
	PercentUrgentPosts float64 `default:"0.001" validate:"range:[0,1]"`
	// The percentage of all posts that are replies
	PercentReplies float64 `default:"0.18" validate:"range:[0,1]"`

	// The IDs of the enabled plugins.
	EnabledPlugins []string

	ScopedConfig ScopedConfig
}

// ScopedConfig holds the settings to load test Scoped Configuration
// (system setting policies).
type ScopedConfig struct {
	// Whether users assign themselves a random value for each attribute field
	// they don't have a value for yet when they first log in.
	AssignAttributes bool `default:"false"`
	// Whether the first controller of the agent logs in as the admin user
	// (ConnectionConfiguration.AdminEmail), creates the attribute fields and
	// churns system setting policies and user attribute values.
	// When running through the coordinator, only the first agent keeps this enabled.
	EnablePolicyChurn bool `default:"false"`
	// The time (in milliseconds) between two system setting policy changes.
	// A value of 0 disables them.
	PolicyChurnIntervalMs int `default:"60000" validate:"range:[0,]"`
	// The time (in milliseconds) between two admin updates of a random user's
	// attribute value. A value of 0 disables them.
	AttributeChurnIntervalMs int `default:"30000" validate:"range:[0,]"`
	// The client config settings to create policies for. Defaults to
	// MobileEnableBiometrics when empty.
	SettingKeys []string
	// The session attribute referenced by session expressions. It is enabled
	// by the churning admin if needed. An empty value disables session expressions.
	SessionAttribute string `default:"user_agent_browser_name"`
	// The select attribute fields referenced by policy expressions. They are
	// created by the churning admin if missing. Defaults to a Department and
	// a Location field when empty.
	AttributeFields []ScopedConfigAttributeField
}

// ScopedConfigAttributeField describes a select custom profile attribute field.
type ScopedConfigAttributeField struct {
	Name    string
	Options []string
}

func (c ScopedConfig) settingKeys() []string {
	if len(c.SettingKeys) == 0 {
		return []string{"MobileEnableBiometrics"}
	}
	return c.SettingKeys
}

func (c ScopedConfig) attributeFields() []ScopedConfigAttributeField {
	if len(c.AttributeFields) == 0 {
		return []ScopedConfigAttributeField{
			{Name: "Department", Options: []string{"Engineering", "Sales", "Support", "Marketing", "Finance", "Operations"}},
			{Name: "Location", Options: []string{"AMER", "EMEA", "APAC"}},
		}
	}
	return c.AttributeFields
}

// ReadConfig reads the configuration file from the given string. If the string
// is empty, it will return a config with default values.
func ReadConfig(configFilePath string) (*Config, error) {
	var cfg Config

	if err := defaults.ReadFrom(configFilePath, "./config/simulcontroller.json", &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
