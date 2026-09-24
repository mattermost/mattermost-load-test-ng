// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package accesscontrol

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/shared/mlog"
)

const (
	sessionAttributeFieldsRoute = "/properties/groups/session_attributes/session/fields"
	accessControlPoliciesRoute  = "/access_control_policies"

	policyTypePermission = "permission"
	policyVersion        = "v0.3"
	// All the policies created by the load-test have IDs starting with this
	// prefix. Server generated IDs never contain the letter "l", so they
	// cannot collide.
	policyIDPrefix = "lt"
)

// The pinned server model predates permission policies and the property
// fields API, so we use our own types for these requests.

type propertyField struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Type  string         `json:"type"`
	Attrs map[string]any `json:"attrs"`
}

type fieldOption struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
}

type permissionPolicy struct {
	ID      string       `json:"id"`
	Name    string       `json:"name"`
	Type    string       `json:"type"`
	Version string       `json:"version"`
	Active  bool         `json:"active"`
	Roles   []string     `json:"roles"`
	Rules   []policyRule `json:"rules"`
}

type policyRule struct {
	Name       string   `json:"name"`
	Actions    []string `json:"actions"`
	Expression string   `json:"expression"`
}

// userField is a user attribute field as created on the server.
type userField struct {
	id   string
	typ  string
	opts map[string]string // option name -> option ID
}

// Manager sets up ABAC on the target instance and assigns the user attribute
// values to the simulated users. It is safe for concurrent use.
type Manager struct {
	cfg      Config
	client   *model.Client4
	fields   map[string]userField
	assigned sync.Map
}

// NewManager returns a new Manager using the given client, which must be
// logged in as a system admin.
func NewManager(cfg Config, client *model.Client4) *Manager {
	return &Manager{
		cfg:    cfg,
		client: client,
		fields: make(map[string]userField),
	}
}

// Setup enables the session attributes, creates the user attributes and
// creates (or updates) the policies. It is idempotent, so that it can run
// every time an agent is created.
func (m *Manager) Setup() error {
	ctx := context.Background()
	if err := m.setupSessionAttributes(ctx); err != nil {
		return fmt.Errorf("accesscontrol: failed to set up session attributes: %w", err)
	}
	if err := m.setupUserAttributes(ctx); err != nil {
		return fmt.Errorf("accesscontrol: failed to set up user attributes: %w", err)
	}
	if err := m.setupPolicies(ctx); err != nil {
		return fmt.Errorf("accesscontrol: failed to set up policies: %w", err)
	}

	rates := m.cfg.ExpectedPermitRates()
	mlog.Info("accesscontrol: setup done",
		mlog.Int("user_attributes", len(m.cfg.UserAttributes)),
		mlog.Int("session_attributes", len(m.cfg.SessionAttributes)),
		mlog.Int("policies", len(m.cfg.Policies)),
		mlog.Float("expected_read_permit_rate", rates.Read),
		mlog.Float("expected_write_permit_rate", rates.Write),
	)
	return nil
}

func (m *Manager) setupSessionAttributes(ctx context.Context) error {
	if len(m.cfg.SessionAttributes) == 0 {
		return nil
	}

	var fields []propertyField
	if err := m.doJSON(ctx, http.MethodGet, sessionAttributeFieldsRoute+"?per_page=100", nil, &fields); err != nil {
		return fmt.Errorf("failed to get session attribute fields (is the SessionAttributes feature flag enabled?): %w", err)
	}

	for _, attr := range m.cfg.SessionAttributes {
		idx := slices.IndexFunc(fields, func(f propertyField) bool { return f.Name == attr.Name })
		if idx == -1 {
			return fmt.Errorf("session attribute %q not found on the server", attr.Name)
		}
		attrs := map[string]any{"enabled": true}
		if attr.TTLSeconds > 0 {
			attrs["ttl_seconds"] = attr.TTLSeconds
		}
		if attr.GracePeriodSeconds > 0 {
			attrs["grace_period_seconds"] = attr.GracePeriodSeconds
		}
		patch := map[string]any{"attrs": attrs}
		if err := m.doJSON(ctx, http.MethodPatch, sessionAttributeFieldsRoute+"/"+fields[idx].ID, patch, nil); err != nil {
			return fmt.Errorf("failed to enable session attribute %q: %w", attr.Name, err)
		}
	}

	return nil
}

func (m *Manager) setupUserAttributes(ctx context.Context) error {
	existing, _, err := m.client.ListCPAFields(ctx)
	if err != nil {
		return fmt.Errorf("failed to list user attribute fields: %w", err)
	}

	for _, attr := range m.cfg.UserAttributes {
		var field *model.PropertyField
		if idx := slices.IndexFunc(existing, func(f *model.PropertyField) bool { return f.Name == attr.Name }); idx != -1 {
			field, err = m.updateUserField(ctx, existing[idx], attr)
		} else {
			field, err = m.createUserField(ctx, attr)
		}
		if err != nil {
			return fmt.Errorf("user attribute %q: %w", attr.Name, err)
		}

		uf := userField{id: field.ID, typ: attr.attrType()}
		if uf.typ == AttributeTypeSelect {
			opts, err := fieldOptions(field.Attrs)
			if err != nil {
				return fmt.Errorf("user attribute %q: %w", attr.Name, err)
			}
			uf.opts = make(map[string]string, len(opts))
			for _, opt := range opts {
				uf.opts[opt.Name] = opt.ID
			}
			for _, v := range attr.Values {
				if uf.opts[v.Value] == "" {
					return fmt.Errorf("user attribute %q: option %q not found on the server", attr.Name, v.Value)
				}
			}
		}
		m.fields[attr.Name] = uf
	}

	return nil
}

func (m *Manager) createUserField(ctx context.Context, attr UserAttribute) (*model.PropertyField, error) {
	attrs := model.StringInterface{
		// Only admin managed attributes can be used in policies, unless
		// AccessControlSettings.EnableUserManagedAttributes is set.
		model.CustomProfileAttributesPropertyAttrsManaged:    "admin",
		model.CustomProfileAttributesPropertyAttrsVisibility: model.CustomProfileAttributesVisibilityWhenSet,
	}
	if attr.attrType() == AttributeTypeSelect {
		opts := make([]fieldOption, len(attr.Values))
		for i, v := range attr.Values {
			opts[i] = fieldOption{Name: v.Value}
		}
		attrs[model.PropertyFieldAttributeOptions] = opts
	}

	field, _, err := m.client.CreateCPAField(ctx, &model.PropertyField{
		Name:  attr.Name,
		Type:  model.PropertyFieldType(attr.attrType()),
		Attrs: attrs,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create field: %w", err)
	}
	return field, nil
}

// updateUserField makes sure an existing field has the configured options and
// is admin managed.
func (m *Manager) updateUserField(ctx context.Context, field *model.PropertyField, attr UserAttribute) (*model.PropertyField, error) {
	if string(field.Type) != attr.attrType() {
		return nil, fmt.Errorf("field already exists with type %q", field.Type)
	}

	attrs := model.StringInterface{}
	if field.Attrs[model.CustomProfileAttributesPropertyAttrsManaged] != "admin" {
		attrs[model.CustomProfileAttributesPropertyAttrsManaged] = "admin"
	}
	if attr.attrType() == AttributeTypeSelect {
		opts, err := fieldOptions(field.Attrs)
		if err != nil {
			return nil, err
		}
		missing := false
		for _, v := range attr.Values {
			if !slices.ContainsFunc(opts, func(o fieldOption) bool { return o.Name == v.Value }) {
				opts = append(opts, fieldOption{Name: v.Value})
				missing = true
			}
		}
		if missing {
			attrs[model.PropertyFieldAttributeOptions] = opts
		}
	}

	if len(attrs) == 0 {
		return field, nil
	}

	updated, _, err := m.client.PatchCPAField(ctx, field.ID, &model.PropertyFieldPatch{Attrs: &attrs})
	if err != nil {
		return nil, fmt.Errorf("failed to update field: %w", err)
	}
	return updated, nil
}

func fieldOptions(attrs model.StringInterface) ([]fieldOption, error) {
	raw, ok := attrs[model.PropertyFieldAttributeOptions]
	if !ok || raw == nil {
		return nil, nil
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal field options: %w", err)
	}
	var opts []fieldOption
	if err := json.Unmarshal(data, &opts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal field options: %w", err)
	}
	return opts, nil
}

// policyID returns a stable ID for the policy with the given name, so that
// the policy gets updated instead of duplicated when the setup runs again.
func policyID(name string) string {
	sum := sha256.Sum256([]byte(name))
	return policyIDPrefix + hex.EncodeToString(sum[:])[:26-len(policyIDPrefix)]
}

func (m *Manager) setupPolicies(ctx context.Context) error {
	wanted := make(map[string]bool, len(m.cfg.Policies))
	for _, p := range m.cfg.Policies {
		policy := permissionPolicy{
			ID:      policyID(p.Name),
			Name:    p.Name,
			Type:    policyTypePermission,
			Version: policyVersion,
			Active:  true,
			Roles:   []string{p.role()},
			Rules: []policyRule{{
				Name:       p.Name,
				Actions:    []string{p.Action},
				Expression: p.Expression(),
			}},
		}
		if err := m.doJSON(ctx, http.MethodPut, accessControlPoliciesRoute, policy, nil); err != nil {
			return fmt.Errorf("failed to save policy %q: %w", p.Name, err)
		}
		wanted[policy.ID] = true
	}

	// Delete the policies left over by previous runs with a different
	// configuration, as the server only evaluates a limited number of them.
	res, _, err := m.client.SearchAccessControlPolicies(ctx, model.AccessControlPolicySearch{
		Type:  policyTypePermission,
		Limit: 1000,
	})
	if err != nil {
		return fmt.Errorf("failed to search policies: %w", err)
	}
	for _, p := range res.Policies {
		if !strings.HasPrefix(p.ID, policyIDPrefix) || wanted[p.ID] {
			continue
		}
		if _, err := m.client.DeleteAccessControlPolicy(ctx, p.ID); err != nil {
			return fmt.Errorf("failed to delete stale policy %q: %w", p.Name, err)
		}
		mlog.Info("accesscontrol: deleted stale policy", mlog.String("name", p.Name))
	}

	return nil
}

// AssignUserAttributes sets the user attribute values of the given user. It
// is a no-op if the values were already set by this Manager.
func (m *Manager) AssignUserAttributes(user *model.User) error {
	if user == nil || user.Id == "" {
		return errors.New("accesscontrol: user ID is required to assign attributes")
	}
	if len(m.fields) == 0 {
		return nil
	}
	if _, ok := m.assigned.Load(user.Id); ok {
		return nil
	}

	names := m.cfg.UserAttributeValues(user.Email)
	values := make(map[string]json.RawMessage, len(names))
	for attrName, value := range names {
		field := m.fields[attrName]
		if field.typ == AttributeTypeSelect {
			value = field.opts[value]
		}
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("accesscontrol: failed to marshal value: %w", err)
		}
		values[field.id] = data
	}

	if _, _, err := m.client.PatchCPAValuesForUser(context.Background(), user.Id, values); err != nil {
		return fmt.Errorf("accesscontrol: failed to assign attributes to user %q: %w", user.Id, err)
	}
	m.assigned.Store(user.Id, struct{}{})
	return nil
}

func (m *Manager) doJSON(ctx context.Context, method, route string, body, out any) error {
	var (
		resp *http.Response
		err  error
	)
	switch method {
	case http.MethodGet:
		resp, err = m.client.DoAPIGet(ctx, route, "")
	case http.MethodPut:
		resp, err = m.client.DoAPIPutJSON(ctx, route, body)
	case http.MethodPatch:
		resp, err = m.client.DoAPIPatchJSON(ctx, route, body)
	default:
		return fmt.Errorf("unsupported method %q", method)
	}
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}
	return nil
}
