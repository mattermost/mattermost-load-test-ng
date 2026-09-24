// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package accesscontrol

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeServer implements the subset of the API used by the Manager.
type fakeServer struct {
	t  *testing.T
	mu sync.Mutex

	sessionFields  []propertyField
	sessionPatches map[string]map[string]any
	cpaFields      []*model.PropertyField
	policies       map[string]permissionPolicy
	deleted        []string
	userValues     map[string]map[string]json.RawMessage
}

func newFakeServer(t *testing.T) *fakeServer {
	return &fakeServer{
		t: t,
		sessionFields: []propertyField{
			{ID: "sf_vpn", Name: "vpn_active", Type: "select"},
			{ID: "sf_os", Name: "os_platform", Type: "select"},
		},
		sessionPatches: map[string]map[string]any{},
		cpaFields: []*model.PropertyField{
			// An existing field missing one of the configured options.
			{ID: "f_clearance", Name: "lt_clearance", Type: model.PropertyFieldTypeSelect, Attrs: model.StringInterface{
				"managed": "admin",
				"options": []any{map[string]any{"id": "o_high", "name": "high"}},
			}},
		},
		policies: map[string]permissionPolicy{
			// A stale policy from a previous run, and one not created by the
			// load-test.
			policyID("lt-old"):           {ID: policyID("lt-old"), Name: "lt-old", Type: policyTypePermission},
			"otherpolicyid0000000000000": {ID: "otherpolicyid0000000000000", Name: "other", Type: policyTypePermission},
		},
		userValues: map[string]map[string]json.RawMessage{},
	}
}

func (s *fakeServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := strings.TrimPrefix(r.URL.Path, "/api/v4")
	writeJSON := func(v any) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(s.t, json.NewEncoder(w).Encode(v))
	}
	decode := func(v any) {
		require.NoError(s.t, json.NewDecoder(r.Body).Decode(v))
	}

	switch {
	case r.Method == http.MethodGet && path == sessionAttributeFieldsRoute:
		writeJSON(s.sessionFields)
	case r.Method == http.MethodPatch && strings.HasPrefix(path, sessionAttributeFieldsRoute+"/"):
		var patch struct {
			Attrs map[string]any `json:"attrs"`
		}
		decode(&patch)
		s.sessionPatches[strings.TrimPrefix(path, sessionAttributeFieldsRoute+"/")] = patch.Attrs
		writeJSON(map[string]any{})
	case r.Method == http.MethodGet && path == "/custom_profile_attributes/fields":
		writeJSON(s.cpaFields)
	case r.Method == http.MethodPost && path == "/custom_profile_attributes/fields":
		var field model.PropertyField
		decode(&field)
		field.ID = "f_" + field.Name
		field.Attrs = withOptionIDs(s.t, field.Attrs)
		s.cpaFields = append(s.cpaFields, &field)
		writeJSON(field)
	case r.Method == http.MethodPatch && strings.HasPrefix(path, "/custom_profile_attributes/fields/"):
		var patch model.PropertyFieldPatch
		decode(&patch)
		id := strings.TrimPrefix(path, "/custom_profile_attributes/fields/")
		for _, f := range s.cpaFields {
			if f.ID == id {
				for k, v := range *patch.Attrs {
					f.Attrs[k] = v
				}
				f.Attrs = withOptionIDs(s.t, f.Attrs)
				writeJSON(f)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	case r.Method == http.MethodPut && path == accessControlPoliciesRoute:
		var p permissionPolicy
		decode(&p)
		s.policies[p.ID] = p
		writeJSON(p)
	case r.Method == http.MethodPost && path == accessControlPoliciesRoute+"/search":
		var policies []permissionPolicy
		for _, p := range s.policies {
			policies = append(policies, p)
		}
		writeJSON(map[string]any{"policies": policies, "total": len(policies)})
	case r.Method == http.MethodDelete && strings.HasPrefix(path, accessControlPoliciesRoute+"/"):
		id := strings.TrimPrefix(path, accessControlPoliciesRoute+"/")
		delete(s.policies, id)
		s.deleted = append(s.deleted, id)
		writeJSON(map[string]any{"status": "OK"})
	case r.Method == http.MethodPatch && strings.HasPrefix(path, "/users/") && strings.HasSuffix(path, "/custom_profile_attributes"):
		userID := strings.Split(path, "/")[2]
		var values map[string]json.RawMessage
		decode(&values)
		s.userValues[userID] = values
		writeJSON(values)
	default:
		s.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}
}

// withOptionIDs assigns IDs to the options that miss one, like the server.
func withOptionIDs(t *testing.T, attrs model.StringInterface) model.StringInterface {
	opts, err := fieldOptions(attrs)
	require.NoError(t, err)
	if opts == nil {
		return attrs
	}
	for i := range opts {
		if opts[i].ID == "" {
			opts[i].ID = "o_" + opts[i].Name
		}
	}
	attrs["options"] = opts
	return attrs
}

func TestManager(t *testing.T) {
	srv := newFakeServer(t)
	ts := httptest.NewServer(srv)
	defer ts.Close()

	cfg := Config{
		Enable:    true,
		UserAgent: "Mattermost/5.13.0 Electron/36.4.0",
		UserAttributes: []UserAttribute{
			{Name: "lt_clearance", Values: []AttributeValue{{Value: "high", Weight: 0.5}, {Value: "low", Weight: 0.5}}},
			{Name: "lt_team", Type: AttributeTypeText, Values: []AttributeValue{{Value: "a", Weight: 1}}},
		},
		SessionAttributes: []SessionAttribute{
			{Name: "vpn_active", TTLSeconds: 600, Values: []AttributeValue{{Value: "true", Weight: 1}}},
		},
		Policies: []Policy{
			{
				Name:                   "lt-read",
				Action:                 ActionChannelRead,
				UserAttribute:          "lt_clearance",
				UserAttributeValues:    []string{"high"},
				Operator:               OperatorOr,
				SessionAttribute:       "vpn_active",
				SessionAttributeValues: []string{"true"},
			},
		},
	}
	require.NoError(t, cfg.IsValid())

	client := model.NewAPIv4Client(ts.URL)
	m := NewManager(cfg, client)
	require.NoError(t, m.Setup())

	t.Run("session attributes are enabled", func(t *testing.T) {
		require.Len(t, srv.sessionPatches, 1)
		assert.Equal(t, map[string]any{"enabled": true, "ttl_seconds": float64(600)}, srv.sessionPatches["sf_vpn"])
	})

	t.Run("user attributes are created or updated", func(t *testing.T) {
		require.Len(t, srv.cpaFields, 2)
		opts, err := fieldOptions(srv.cpaFields[0].Attrs)
		require.NoError(t, err)
		assert.Equal(t, []fieldOption{{ID: "o_high", Name: "high"}, {ID: "o_low", Name: "low"}}, opts)

		created := srv.cpaFields[1]
		assert.Equal(t, "lt_team", created.Name)
		assert.Equal(t, model.PropertyFieldTypeText, created.Type)
		assert.Equal(t, "admin", created.Attrs["managed"])
	})

	t.Run("policies are saved and stale ones deleted", func(t *testing.T) {
		assert.Equal(t, []string{policyID("lt-old")}, srv.deleted)
		assert.Contains(t, srv.policies, "otherpolicyid0000000000000")

		p, ok := srv.policies[policyID("lt-read")]
		require.True(t, ok)
		assert.Equal(t, policyTypePermission, p.Type)
		assert.Equal(t, []string{RoleSystemUser}, p.Roles)
		require.Len(t, p.Rules, 1)
		assert.Equal(t, "lt-read", p.Rules[0].Name)
		assert.Equal(t, []string{ActionChannelRead}, p.Rules[0].Actions)
		assert.Equal(t, cfg.Policies[0].Expression(), p.Rules[0].Expression)
	})

	t.Run("setup is idempotent", func(t *testing.T) {
		require.NoError(t, NewManager(cfg, client).Setup())
		assert.Len(t, srv.cpaFields, 2)
		assert.Len(t, srv.policies, 2)
	})

	t.Run("user attributes are assigned", func(t *testing.T) {
		user := &model.User{Id: "user1", Email: "user-1@example.com"}
		require.NoError(t, m.AssignUserAttributes(user))

		expected := cfg.UserAttributeValues(user.Email)
		values := srv.userValues["user1"]
		require.Len(t, values, 2)
		assert.JSONEq(t, `"o_`+expected["lt_clearance"]+`"`, string(values["f_clearance"]))
		assert.JSONEq(t, `"a"`, string(values["f_lt_team"]))

		// Assigning again is a no-op.
		delete(srv.userValues, "user1")
		require.NoError(t, m.AssignUserAttributes(user))
		assert.NotContains(t, srv.userValues, "user1")
	})

	t.Run("user ID is required", func(t *testing.T) {
		require.Error(t, m.AssignUserAttributes(&model.User{Email: "user-2@example.com"}))
	})
}
