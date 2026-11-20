// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package userentity

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user"
	"github.com/mattermost/mattermost-load-test-ng/performance"

	"github.com/mattermost/mattermost/server/public/model"

	"github.com/stretchr/testify/require"
)

func scrapeMetrics(t *testing.T, metrics *performance.Metrics) string {
	t.Helper()
	rec := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	return rec.Body.String()
}

func newScopedConfigTestUser(t *testing.T, mux *http.ServeMux) (*UserEntity, *performance.Metrics) {
	t.Helper()

	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	var cfg config
	err := defaults.ReadFrom("", "../../../config/config.sample.json", &cfg)
	require.NoError(t, err)

	s, err := memstore.New(nil)
	require.NoError(t, err)

	metrics := performance.NewMetrics()
	u := New(Setup{Store: s, Metrics: metrics.UserEntityMetrics()}, Config{
		ServerURL:    ts.URL,
		WebSocketURL: cfg.ConnectionConfiguration.WebSocketURL,
		Username:     "testuser",
		Email:        "testuser@example.com",
		Password:     "testpassword",
	})
	return u, metrics
}

func TestGetClientConfigOverrides(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /api/v4/config/client/overrides", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"MobileEnableBiometrics":"true"}`))
		})
		u, metrics := newScopedConfigTestUser(t, mux)

		require.NoError(t, u.GetClientConfigOverrides())
		require.Equal(t, map[string]string{"MobileEnableBiometrics": "true"}, u.Store().ClientConfigOverrides())
		require.Contains(t, scrapeMetrics(t, metrics), `loadtest_http_scoped_config_request_time_count{request="get_client_config_overrides"} 1`)
	})

	t.Run("error", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.HandleFunc("GET /api/v4/config/client/overrides", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"id":"app.error","message":"error","status_code":500}`))
		})
		u, metrics := newScopedConfigTestUser(t, mux)
		u.store.SetClientConfigOverrides(map[string]string{"MobileEnableBiometrics": "true"})

		require.Error(t, u.GetClientConfigOverrides())
		require.Equal(t, map[string]string{"MobileEnableBiometrics": "true"}, u.Store().ClientConfigOverrides())
		require.Contains(t, scrapeMetrics(t, metrics), `loadtest_http_errors_total{method="GET",path="/api/v4/config/client/overrides",status_code="500"} 1`)
	})
}

func TestSaveAndSearchSystemSettingPolicies(t *testing.T) {
	policy := &user.SystemSettingPolicy{
		Name:  "policy",
		Type:  user.SystemSettingPolicyType,
		Roles: []string{model.SystemUserRoleId},
		Rules: []user.SystemSettingPolicyRule{{
			Actions:    []string{"MobileEnableBiometrics"},
			Expression: `user.attributes.Department == "Sales"`,
			Value:      "true",
		}},
	}

	mux := http.NewServeMux()
	mux.HandleFunc("PUT /api/v4/access_control_policies", func(w http.ResponseWriter, r *http.Request) {
		var received user.SystemSettingPolicy
		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		require.Equal(t, *policy, received)
		received.ID = "policyid"
		received.Revision = 1
		require.NoError(t, json.NewEncoder(w).Encode(received))
	})
	mux.HandleFunc("POST /api/v4/access_control_policies/search", func(w http.ResponseWriter, r *http.Request) {
		var search map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&search))
		require.Equal(t, user.SystemSettingPolicyType, search["type"])
		_, _ = w.Write([]byte(`{"policies":[{"id":"policyid","name":"policy","type":"system_setting","revision":1,"roles":["system_user"],"rules":[{"actions":["MobileEnableBiometrics"],"expression":"true","value":"false"}]}],"total":1}`))
	})
	mux.HandleFunc("GET /api/v4/access_control_policies/{id}", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "policyid", r.PathValue("id"))
		_, _ = w.Write([]byte(`{"id":"policyid","name":"policy","type":"system_setting","revision":1,"roles":["system_user"],"rules":[{"actions":["MobileEnableBiometrics"],"expression":"user.attributes.Department == \"Sales\"","value":"true"}]}`))
	})
	u, _ := newScopedConfigTestUser(t, mux)

	saved, err := u.SaveSystemSettingPolicy(policy)
	require.NoError(t, err)
	require.Equal(t, "policyid", saved.ID)
	require.Equal(t, "MobileEnableBiometrics", saved.SettingKey())

	policies, err := u.SearchSystemSettingPolicies()
	require.NoError(t, err)
	require.Len(t, policies, 1)
	require.Equal(t, "policyid", policies[0].ID)
	require.Equal(t, "false", policies[0].Rules[0].Value)

	fetched, err := u.GetSystemSettingPolicy("policyid")
	require.NoError(t, err)
	require.Equal(t, policy.Rules, fetched.Rules)
}

func TestHandleSystemSettingPolicyUpdatedEvent(t *testing.T) {
	u, metrics := newScopedConfigTestUser(t, http.NewServeMux())

	ev := model.NewWebSocketEvent(model.WebsocketEventType(user.WebsocketEventSystemSettingPolicyUpdated), "", "", "", nil, "")
	ev.Add("overrides", map[string]any{"MobileEnableBiometrics": "true"})
	ev = ev.SetSequence(0)
	require.NoError(t, u.wsEventHandler(ev))
	require.Equal(t, map[string]string{"MobileEnableBiometrics": "true"}, u.Store().ClientConfigOverrides())

	ev = model.NewWebSocketEvent(model.WebsocketEventType(user.WebsocketEventSystemSettingPolicyUpdated), "", "", "", nil, "")
	ev.Add("overrides", map[string]any{})
	ev = ev.SetSequence(1)
	require.NoError(t, u.wsEventHandler(ev))
	require.Empty(t, u.Store().ClientConfigOverrides())

	ev = model.NewWebSocketEvent(model.WebsocketEventType(user.WebsocketEventSystemSettingPolicyUpdated), "", "", "", nil, "")
	ev = ev.SetSequence(2)
	require.Error(t, u.wsEventHandler(ev))

	require.Contains(t, scrapeMetrics(t, metrics), "loadtest_websocket_system_setting_policy_updated_total 3")
}
