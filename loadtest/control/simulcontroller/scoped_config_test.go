// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"encoding/json"
	"regexp"
	"strconv"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
)

func newSelectField(id, name string, managed string, options ...string) *model.PropertyField {
	opts := make([]any, 0, len(options))
	for i, o := range options {
		opts = append(opts, map[string]any{"id": id + "-opt" + strconv.Itoa(i), "name": o})
	}
	return &model.PropertyField{
		ID:   id,
		Name: name,
		Type: model.PropertyFieldTypeSelect,
		Attrs: model.StringInterface{
			"options": opts,
			"managed": managed,
		},
	}
}

func TestScopedConfigDefaults(t *testing.T) {
	config, err := ReadConfig("../../../config/simulcontroller.sample.json")
	require.NoError(t, err)

	require.False(t, config.ScopedConfig.EnablePolicyChurn)
	require.False(t, config.ScopedConfig.AssignAttributes)
	require.Equal(t, 60000, config.ScopedConfig.PolicyChurnIntervalMs)
	require.Equal(t, 30000, config.ScopedConfig.AttributeChurnIntervalMs)
	require.Equal(t, "user_agent_browser_name", config.ScopedConfig.SessionAttribute)
	require.Equal(t, []string{"MobileEnableBiometrics"}, config.ScopedConfig.settingKeys())
	require.Len(t, config.ScopedConfig.attributeFields(), 2)

	config.ScopedConfig.SettingKeys = []string{"EnableBanner"}
	require.Equal(t, []string{"EnableBanner"}, config.ScopedConfig.settingKeys())
}

func TestIsScopedConfigLeader(t *testing.T) {
	c := &SimulController{id: 0, config: &Config{}}
	require.False(t, c.isScopedConfigLeader())

	c.config.ScopedConfig.EnablePolicyChurn = true
	require.True(t, c.isScopedConfigLeader())

	c.id = 1
	require.False(t, c.isScopedConfigLeader())
}

func TestAssignableCPAOptions(t *testing.T) {
	field := newSelectField("f", "Department", "", "Sales", scopedConfigSpareOption, "Support")
	require.Len(t, cpaFieldOptions(field), 3)

	options := assignableCPAOptions(field)
	require.Equal(t, []cpaOption{{ID: "f-opt0", Name: "Sales"}, {ID: "f-opt2", Name: "Support"}}, options)

	require.Empty(t, cpaFieldOptions(&model.PropertyField{}))
}

func TestMissingScopedConfigAttributes(t *testing.T) {
	department := newSelectField("dep", "Department", "", "Sales", "Support")
	location := newSelectField("loc", "Location", "", "EMEA")
	adminManaged := newSelectField("team", "Team", "admin", "Red")
	text := &model.PropertyField{ID: "title", Name: "Title", Type: model.PropertyFieldTypeText}
	fields := []*model.PropertyField{department, location, adminManaged, text}

	configured := []ScopedConfigAttributeField{
		{Name: "Department"},
		{Name: "Location"},
		{Name: "Team"},
		{Name: "Title"},
		{Name: "Missing"},
	}

	t.Run("no values", func(t *testing.T) {
		patch, err := missingScopedConfigAttributes(fields, configured, nil)
		require.NoError(t, err)
		require.Len(t, patch, 2)
		require.Contains(t, []string{`"dep-opt0"`, `"dep-opt1"`}, string(patch["dep"]))
		require.Equal(t, `"loc-opt0"`, string(patch["loc"]))
	})

	t.Run("existing values", func(t *testing.T) {
		values := map[string]json.RawMessage{
			"dep": json.RawMessage(`"dep-opt1"`),
			"loc": json.RawMessage(`""`),
		}
		patch, err := missingScopedConfigAttributes(fields, configured, values)
		require.NoError(t, err)
		require.Equal(t, map[string]json.RawMessage{"loc": json.RawMessage(`"loc-opt0"`)}, patch)
	})
}

func TestScopedConfigExpressions(t *testing.T) {
	fields := []*model.PropertyField{
		newSelectField("dep", "Department", "", "Engineering", "Sales", "Support"),
		newSelectField("loc", "Location", "", "AMER", "EMEA"),
	}

	condition := `user\.attributes\.(Department|Location) (== "[A-Za-z]+"|in \["[A-Za-z]+", "[A-Za-z]+"\])`
	userExprRe := regexp.MustCompile(`^` + condition + `( && ` + condition + `)?$`)
	sessionExprRe := regexp.MustCompile(`^user\.session\.user_agent_browser_name (!= "Unknown"|== "Unknown" && ` + condition + `( && ` + condition + `)?)$`)

	for range 200 {
		require.Regexp(t, userExprRe, userAttributeExpression(fields))
		require.Regexp(t, sessionExprRe, sessionAttributeExpression("user_agent_browser_name", fields))
	}

	require.Equal(t, "true", userAttributeExpression(nil))
}

func TestNextPolicyValue(t *testing.T) {
	require.Equal(t, "true", nextPolicyValue("false", ""))
	require.Equal(t, "false", nextPolicyValue("false", "true"))
	require.Equal(t, "true", nextPolicyValue("true", "false"))

	n, err := strconv.Atoi(nextPolicyValue("10", ""))
	require.NoError(t, err)
	require.Positive(t, n)

	require.NotEmpty(t, nextPolicyValue("", ""))
}

func TestOtherSystemRole(t *testing.T) {
	require.Equal(t, []string{model.SystemAdminRoleId}, otherSystemRole([]string{model.SystemUserRoleId}))
	require.Equal(t, []string{model.SystemUserRoleId}, otherSystemRole([]string{model.SystemAdminRoleId}))
	require.Equal(t, []string{model.SystemAdminRoleId}, otherSystemRole(nil))
}

func TestPickPolicyChurnOp(t *testing.T) {
	var picked []int
	ops := []policyChurnOp{
		{weight: 0, run: func() (string, error) { picked = append(picked, 0); return "", nil }},
		{weight: 1, run: func() (string, error) { picked = append(picked, 1); return "", nil }},
	}
	for range 50 {
		_, err := pickPolicyChurnOp(ops).run()
		require.NoError(t, err)
	}
	require.NotContains(t, picked, 0)
}
