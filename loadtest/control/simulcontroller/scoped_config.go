// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License.txt for license information.

package simulcontroller

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math/rand"
	"slices"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"

	"github.com/mattermost/mattermost/server/public/model"
)

const (
	// Controller ids start at 1, so the first controller of an agent leads.
	scopedConfigLeaderID         = 1
	scopedConfigPolicyNamePrefix = "loadtest-"
	// An option toggled on and off by the churning admin to exercise attribute
	// field changes. It is never assigned to users.
	scopedConfigSpareOption = "LoadTestSpare"
	// The value of user_agent_browser_name for sessions created by load test agents.
	scopedConfigAgentBrowserName = "Unknown"
)

type scopedConfigState struct {
	ready       bool
	fields      []*model.PropertyField
	sessionAttr string
}

type cpaOption struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// IsScopedConfigLeader returns whether the controller with the given id
// churns system setting policies. That controller logs in as the admin user.
func IsScopedConfigLeader(config *Config, id int) bool {
	return config != nil && config.ScopedConfig.EnablePolicyChurn && id == scopedConfigLeaderID
}

func (c *SimulController) isScopedConfigLeader() bool {
	return IsScopedConfigLeader(c.config, c.id)
}

func cpaFieldOptions(field *model.PropertyField) []cpaOption {
	if field == nil || field.Attrs == nil {
		return nil
	}
	data, err := json.Marshal(field.Attrs["options"])
	if err != nil {
		return nil
	}
	var options []cpaOption
	if err := json.Unmarshal(data, &options); err != nil {
		return nil
	}
	return options
}

func assignableCPAOptions(field *model.PropertyField) []cpaOption {
	return slices.DeleteFunc(cpaFieldOptions(field), func(o cpaOption) bool {
		return o.Name == scopedConfigSpareOption
	})
}

func isAdminManagedCPAField(field *model.PropertyField) bool {
	managed, _ := field.Attrs[model.CustomProfileAttributesPropertyAttrsManaged].(string)
	return managed == "admin"
}

func findCPAField(fields []*model.PropertyField, name string) *model.PropertyField {
	for _, field := range fields {
		if field.Name == name && field.DeleteAt == 0 {
			return field
		}
	}
	return nil
}

func hasCPAValue(values map[string]json.RawMessage, fieldID string) bool {
	value, ok := values[fieldID]
	return ok && len(value) > 0 && string(value) != "null" && string(value) != `""`
}

// missingScopedConfigAttributes returns a random option for each configured
// select field the user can set and doesn't have a value for.
func missingScopedConfigAttributes(fields []*model.PropertyField, configured []ScopedConfigAttributeField, values map[string]json.RawMessage) (map[string]json.RawMessage, error) {
	patch := map[string]json.RawMessage{}
	for _, cfgField := range configured {
		field := findCPAField(fields, cfgField.Name)
		if field == nil || field.Type != model.PropertyFieldTypeSelect || isAdminManagedCPAField(field) || hasCPAValue(values, field.ID) {
			continue
		}
		options := assignableCPAOptions(field)
		if len(options) == 0 {
			continue
		}
		value, err := json.Marshal(options[rand.Intn(len(options))].ID)
		if err != nil {
			return nil, err
		}
		patch[field.ID] = value
	}
	return patch, nil
}

func assignScopedConfigAttributes(u user.User, configured []ScopedConfigAttributeField) control.UserActionResponse {
	if !control.SystemSettingPoliciesEnabled(u) {
		return control.UserActionResponse{}
	}
	if cpaEnabled, resp := control.CustomProfileAttributesEnabled(u); resp.Err != nil || !cpaEnabled {
		return resp
	}

	values, err := u.GetCPAValues(u.Store().Id())
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	patch, err := missingScopedConfigAttributes(u.Store().GetCPAFields(), configured, values)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if len(patch) == 0 {
		return control.UserActionResponse{}
	}

	if err := u.PatchCPAValues(patch); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("assigned %d scoped config attributes", len(patch))}
}

func ensureSelectCPAField(u user.User, cfgField ScopedConfigAttributeField) (*model.PropertyField, error) {
	if field := findCPAField(u.Store().GetCPAFields(), cfgField.Name); field != nil {
		if field.Type != model.PropertyFieldTypeSelect {
			return nil, fmt.Errorf("attribute field %q has type %q, expected %q", field.Name, field.Type, model.PropertyFieldTypeSelect)
		}
		return field, nil
	}

	if len(cfgField.Options) == 0 {
		return nil, fmt.Errorf("attribute field %q has no options", cfgField.Name)
	}

	options := make([]cpaOption, 0, len(cfgField.Options))
	for _, name := range cfgField.Options {
		options = append(options, cpaOption{Name: name})
	}

	return u.CreateCPAField(&model.PropertyField{
		Name: cfgField.Name,
		Type: model.PropertyFieldTypeSelect,
		Attrs: model.StringInterface{
			model.CustomProfileAttributesPropertyAttrsVisibility: model.CustomProfileAttributesVisibilityWhenSet,
			"options": options,
		},
	})
}

func ensureSessionAttributeEnabled(u user.User, name string) error {
	fields, err := u.GetSessionAttributeFields()
	if err != nil {
		return err
	}

	for _, field := range fields {
		if field.Name != name {
			continue
		}
		if enabled, _ := field.Attrs["enabled"].(bool); enabled {
			return nil
		}
		attrs := maps.Clone(field.Attrs)
		if attrs == nil {
			attrs = model.StringInterface{}
		}
		attrs["enabled"] = true
		_, err := u.PatchSessionAttributeField(field.ID, &model.PropertyFieldPatch{Attrs: &attrs})
		return err
	}

	return fmt.Errorf("session attribute %q not found", name)
}

// setupScopedConfig makes sure the attribute fields exist and the session
// attribute is enabled before churning policies.
func (c *SimulController) setupScopedConfig(u user.User) control.UserActionResponse {
	if !control.SystemSettingPoliciesEnabled(u) {
		return control.UserActionResponse{Info: "system setting policies are not enabled"}
	}

	isAdmin, err := u.IsSysAdmin()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if !isAdmin {
		return control.UserActionResponse{Err: control.NewUserError(errors.New("scoped config policy churn requires a system admin user"))}
	}

	if err := u.GetCPAFields(); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	var fields []*model.PropertyField
	for _, cfgField := range c.config.ScopedConfig.attributeFields() {
		field, err := ensureSelectCPAField(u, cfgField)
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		fields = append(fields, field)
	}

	if err := u.GetCPAFields(); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	var sessionAttr string
	if name := c.config.ScopedConfig.SessionAttribute; name != "" {
		if err := ensureSessionAttributeEnabled(u, name); err != nil {
			c.status <- c.newErrorStatus(control.NewUserError(fmt.Errorf("session expressions disabled: %w", err)))
		} else {
			sessionAttr = name
		}
	}

	c.scopedConfig = scopedConfigState{
		ready:       true,
		fields:      fields,
		sessionAttr: sessionAttr,
	}

	return control.UserActionResponse{Info: fmt.Sprintf("scoped config ready with %d attribute fields and session attribute %q", len(fields), sessionAttr)}
}

func quoteCELStrings(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, strconv.Quote(v))
	}
	return strings.Join(quoted, ", ")
}

func attributeCondition(field *model.PropertyField) string {
	options := assignableCPAOptions(field)
	if len(options) == 0 {
		return ""
	}
	rand.Shuffle(len(options), func(i, j int) { options[i], options[j] = options[j], options[i] })

	names := make([]string, 0, 2)
	for _, o := range options[:1+rand.Intn(min(2, len(options)))] {
		names = append(names, o.Name)
	}

	if len(names) == 1 {
		return fmt.Sprintf("user.attributes.%s == %s", field.Name, strconv.Quote(names[0]))
	}
	return fmt.Sprintf("user.attributes.%s in [%s]", field.Name, quoteCELStrings(names))
}

// userAttributeExpression returns an expression matching a fraction of the
// users based on one or two attribute fields.
func userAttributeExpression(fields []*model.PropertyField) string {
	if len(fields) == 0 {
		return "true"
	}

	perm := rand.Perm(len(fields))
	conditions := []string{attributeCondition(fields[perm[0]])}
	if len(fields) > 1 && rand.Float64() < 0.3 {
		conditions = append(conditions, attributeCondition(fields[perm[1]]))
	}

	expr := strings.Join(slices.DeleteFunc(conditions, func(s string) bool { return s == "" }), " && ")
	if expr == "" {
		return "true"
	}
	return expr
}

// sessionAttributeExpression returns an expression referencing the given
// session attribute. Most of them also match on user attributes so that only
// a fraction of the agent sessions is targeted.
func sessionAttributeExpression(sessionAttr string, fields []*model.PropertyField) string {
	if rand.Float64() < 0.2 {
		return fmt.Sprintf("user.session.%s != %s", sessionAttr, strconv.Quote(scopedConfigAgentBrowserName))
	}
	expr := fmt.Sprintf("user.session.%s == %s", sessionAttr, strconv.Quote(scopedConfigAgentBrowserName))
	if userExpr := userAttributeExpression(fields); userExpr != "true" {
		expr += " && " + userExpr
	}
	return expr
}

// nextPolicyValue returns a value for the setting that differs from the
// current policy value or, for new policies, from the server default.
func nextPolicyValue(defaultValue, currentValue string) string {
	reference := currentValue
	if reference == "" {
		reference = defaultValue
	}
	if b, err := strconv.ParseBool(reference); err == nil {
		return strconv.FormatBool(!b)
	}
	if _, err := strconv.Atoi(reference); err == nil {
		return strconv.Itoa(1 + rand.Intn(100))
	}
	return control.PickRandomWord()
}

func otherSystemRole(roles []string) []string {
	if slices.Contains(roles, model.SystemAdminRoleId) {
		return []string{model.SystemUserRoleId}
	}
	return []string{model.SystemAdminRoleId}
}

func (c *SimulController) newSystemSettingPolicy(u user.User, key string) *user.SystemSettingPolicy {
	expression := userAttributeExpression(c.scopedConfig.fields)
	if c.scopedConfig.sessionAttr != "" && rand.Float64() < 0.3 {
		expression = sessionAttributeExpression(c.scopedConfig.sessionAttr, c.scopedConfig.fields)
	}

	return &user.SystemSettingPolicy{
		Name:  scopedConfigPolicyNamePrefix + key,
		Type:  user.SystemSettingPolicyType,
		Roles: []string{model.SystemUserRoleId},
		Rules: []user.SystemSettingPolicyRule{{
			Actions:    []string{key},
			Expression: expression,
			Value:      nextPolicyValue(u.Store().ClientConfig()[key], ""),
		}},
	}
}

type policyChurnOp struct {
	weight float64
	run    func() (string, error)
}

func pickPolicyChurnOp(ops []policyChurnOp) policyChurnOp {
	var total float64
	for _, op := range ops {
		total += op.weight
	}
	r := rand.Float64() * total
	for _, op := range ops {
		if r < op.weight {
			return op
		}
		r -= op.weight
	}
	return ops[len(ops)-1]
}

// churnSystemSettingPolicy creates, updates or deletes the system setting
// policy of a random setting key.
func (c *SimulController) churnSystemSettingPolicy(u user.User) control.UserActionResponse {
	if !c.scopedConfig.ready {
		if resp := c.setupScopedConfig(u); resp.Err != nil || !c.scopedConfig.ready {
			return resp
		}
	}

	policies, err := u.SearchSystemSettingPolicies()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	keys := c.config.ScopedConfig.settingKeys()
	key := keys[rand.Intn(len(keys))]

	var policy *user.SystemSettingPolicy
	for _, p := range policies {
		if p.SettingKey() == key {
			policy = p
			break
		}
	}

	if policy == nil {
		saved, err := u.SaveSystemSettingPolicy(c.newSystemSettingPolicy(u, key))
		if err != nil {
			return control.UserActionResponse{Err: control.NewUserError(err)}
		}
		return control.UserActionResponse{Info: fmt.Sprintf("created system setting policy %s for %s: %q", saved.ID, key, saved.Rules[0].Expression)}
	}

	if !strings.HasPrefix(policy.Name, scopedConfigPolicyNamePrefix) {
		return control.UserActionResponse{Info: fmt.Sprintf("skipping %s: it is managed by policy %q", key, policy.Name)}
	}

	// Search results reference attributes by ID, which can't be saved back.
	policy, err = u.GetSystemSettingPolicy(policy.ID)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	save := func(desc string) (string, error) {
		saved, err := u.SaveSystemSettingPolicy(policy)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("updated %s of system setting policy %s for %s", desc, saved.ID, key), nil
	}

	ops := []policyChurnOp{
		{weight: 0.3, run: func() (string, error) {
			policy.Rules[0].Expression = userAttributeExpression(c.scopedConfig.fields)
			return save("user attribute expression")
		}},
		{weight: 0.2, run: func() (string, error) {
			policy.Rules[0].Value = nextPolicyValue(u.Store().ClientConfig()[key], policy.Rules[0].Value)
			return save("value")
		}},
		{weight: 0.1, run: func() (string, error) {
			policy.Roles = otherSystemRole(policy.Roles)
			return save("role")
		}},
		{weight: 0.15, run: func() (string, error) {
			if err := u.DeleteSystemSettingPolicy(policy.ID); err != nil {
				return "", err
			}
			return fmt.Sprintf("deleted system setting policy %s for %s", policy.ID, key), nil
		}},
		{weight: 0.1, run: func() (string, error) {
			return c.toggleSpareAttributeOption(u)
		}},
	}
	if c.scopedConfig.sessionAttr != "" {
		ops = append(ops, policyChurnOp{weight: 0.15, run: func() (string, error) {
			policy.Rules[0].Expression = sessionAttributeExpression(c.scopedConfig.sessionAttr, c.scopedConfig.fields)
			return save("session attribute expression")
		}})
	}

	info, err := pickPolicyChurnOp(ops).run()
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: info}
}

// toggleSpareAttributeOption adds or removes the spare option of a random
// attribute field, which makes the server re-evaluate the policies reading it.
func (c *SimulController) toggleSpareAttributeOption(u user.User) (string, error) {
	if len(c.scopedConfig.fields) == 0 {
		return "no attribute fields to update", nil
	}

	idx := rand.Intn(len(c.scopedConfig.fields))
	field := c.scopedConfig.fields[idx]
	if isAdminManagedCPAField(field) {
		return fmt.Sprintf("skipping admin managed attribute field %s", field.Name), nil
	}

	options := assignableCPAOptions(field)
	action := "removed"
	if len(options) == len(cpaFieldOptions(field)) {
		options = append(options, cpaOption{Name: scopedConfigSpareOption})
		action = "added"
	}

	attrs := model.StringInterface{"options": options}
	updated, err := u.PatchCPAField(field.ID, &model.PropertyFieldPatch{Attrs: &attrs})
	if err != nil {
		return "", err
	}
	c.scopedConfig.fields[idx] = updated

	return fmt.Sprintf("%s spare option of attribute field %s", action, field.Name), nil
}

// churnUserAttribute sets a random attribute field of a random user to a
// random option, which pushes overrides to that user's sessions.
func (c *SimulController) churnUserAttribute(u user.User) control.UserActionResponse {
	if !c.scopedConfig.ready {
		if resp := c.setupScopedConfig(u); resp.Err != nil || !c.scopedConfig.ready {
			return resp
		}
	}

	if len(c.scopedConfig.fields) == 0 {
		return control.UserActionResponse{Info: "no attribute fields to update"}
	}

	target, err := u.Store().RandomUser()
	if errors.Is(err, memstore.ErrLenMismatch) {
		return control.UserActionResponse{Info: "no users to update"}
	} else if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if target.IsBot || target.DeleteAt != 0 {
		return control.UserActionResponse{Info: fmt.Sprintf("skipping attribute update for user %s", target.Id)}
	}

	field := c.scopedConfig.fields[rand.Intn(len(c.scopedConfig.fields))]
	options := assignableCPAOptions(field)
	if len(options) == 0 {
		return control.UserActionResponse{Info: fmt.Sprintf("attribute field %s has no options", field.Name)}
	}
	option := options[rand.Intn(len(options))]

	value, err := json.Marshal(option.ID)
	if err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}
	if err := u.PatchCPAValuesForUser(target.Id, map[string]json.RawMessage{field.ID: value}); err != nil {
		return control.UserActionResponse{Err: control.NewUserError(err)}
	}

	return control.UserActionResponse{Info: fmt.Sprintf("set attribute %s=%s for user %s", field.Name, option.Name, target.Id)}
}
