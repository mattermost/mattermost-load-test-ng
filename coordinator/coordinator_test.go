// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"

	"github.com/stretchr/testify/require"
)

func newConfig(t *testing.T) *Config {
	t.Helper()
	var cfg Config
	defaults.Set(&cfg)
	cfg.MonitorConfig.Queries[0].Description = "Query"
	cfg.MonitorConfig.Queries[0].Query = "query"
	return &cfg
}

func setupAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := struct {
			Id      string           `json:"id,omitempty"`
			Message string           `json:"message,omitempty"`
			Status  *loadtest.Status `json:"status,omitempty"`
			Error   string           `json:"error,omitempty"`
		}{
			"lt0",
			"",
			&loadtest.Status{},
			"",
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

func TestNew(t *testing.T) {
	c, err := New(nil)
	require.Error(t, err)
	require.Nil(t, c)

	c, err = New(newConfig(t))
	require.NoError(t, err)
	require.NotNil(t, c)
}

func TestRun(t *testing.T) {
	srv := setupAPIServer(t)
	defer srv.Close()

	cfg := newConfig(t)
	cfg.ClusterConfig.Agents[0].ApiURL = srv.URL

	c, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, c)

	done, err := c.Run()
	require.NoError(t, err)
	require.NotNil(t, done)

	done, err = c.Run()
	require.Error(t, err)
	require.Nil(t, done)

	err = c.Stop()
	require.NoError(t, err)
}

func TestStop(t *testing.T) {
	srv := setupAPIServer(t)
	defer srv.Close()

	cfg := newConfig(t)
	cfg.ClusterConfig.Agents[0].ApiURL = srv.URL

	c, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, c)

	err = c.Stop()
	require.Error(t, err)

	done, err := c.Run()
	require.NoError(t, err)
	require.NotNil(t, done)

	err = c.Stop()
	require.NoError(t, err)

	err = c.Stop()
	require.Error(t, err)
}

func TestStatus(t *testing.T) {
	srv := setupAPIServer(t)
	defer srv.Close()

	cfg := newConfig(t)
	cfg.ClusterConfig.Agents[0].ApiURL = srv.URL

	c, err := New(cfg)
	require.NoError(t, err)
	require.NotNil(t, c)

	status := c.Status()
	require.Equal(t, Stopped, status.State)
	require.Empty(t, status)

	done, err := c.Run()
	require.NoError(t, err)
	require.NotNil(t, done)

	status = c.Status()
	require.Equal(t, Running, status.State)

	err = c.Stop()
	require.NoError(t, err)

	status = c.Status()
	require.Equal(t, Stopped, status.State)
	require.NotEmpty(t, status)
}
