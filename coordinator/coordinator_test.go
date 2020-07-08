// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package coordinator

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/logger"

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

func newLoadTestConfig(t *testing.T) loadtest.Config {
	t.Helper()
	var cfg loadtest.Config
	defaults.Set(&cfg)
	return cfg
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
	c, err := New(nil, newLoadTestConfig(t), logger.New(&logger.Settings{}))
	require.Error(t, err)
	require.Nil(t, c)

	c, err = New(newConfig(t), loadtest.Config{}, logger.New(&logger.Settings{}))
	require.Error(t, err)
	require.Nil(t, c)

	c, err = New(newConfig(t), newLoadTestConfig(t), nil)
	require.Error(t, err)
	require.Nil(t, c)

	srv := setupAPIServer(t)
	defer srv.Close()

	cfg := newConfig(t)
	cfg.ClusterConfig.Agents[0].ApiURL = srv.URL
	c, err = New(cfg, newLoadTestConfig(t), logger.New(&logger.Settings{}))
	require.NoError(t, err)
	require.NotNil(t, c)
}

func TestRun(t *testing.T) {
	srv := setupAPIServer(t)
	defer srv.Close()

	cfg := newConfig(t)
	cfg.ClusterConfig.Agents[0].ApiURL = srv.URL

	c, err := New(cfg, newLoadTestConfig(t), logger.New(&logger.Settings{}))
	require.NoError(t, err)
	require.NotNil(t, c)

	done, err := c.Run()
	require.NoError(t, err)
	require.NotNil(t, done)
	c.mut.RLock()
	require.Equal(t, Running, c.status.State)
	c.mut.RUnlock()

	done, err = c.Run()
	require.Error(t, err)
	require.Equal(t, ErrNotStopped, err)
	require.Nil(t, done)
	c.mut.RLock()
	require.Equal(t, Running, c.status.State)
	c.mut.RUnlock()

	err = c.Stop()
	require.NoError(t, err)
	c.mut.RLock()
	require.Equal(t, Done, c.status.State)
	c.mut.RUnlock()

	done, err = c.Run()
	require.Error(t, err)
	require.Nil(t, done)
	c.mut.RLock()
	require.Equal(t, ErrAlreadyDone, err)
	c.mut.RUnlock()
}

func TestStop(t *testing.T) {
	srv := setupAPIServer(t)
	defer srv.Close()

	cfg := newConfig(t)
	cfg.ClusterConfig.Agents[0].ApiURL = srv.URL

	c, err := New(cfg, newLoadTestConfig(t), logger.New(&logger.Settings{}))
	require.NoError(t, err)
	require.NotNil(t, c)
	c.mut.RLock()
	require.Equal(t, Stopped, c.status.State)
	c.mut.RUnlock()

	err = c.Stop()
	require.Error(t, err)

	done, err := c.Run()
	require.NoError(t, err)
	require.NotNil(t, done)
	c.mut.RLock()
	require.Equal(t, Running, c.status.State)
	c.mut.RUnlock()

	err = c.Stop()
	require.NoError(t, err)
	c.mut.RLock()
	require.Equal(t, Done, c.status.State)
	c.mut.RUnlock()

	err = c.Stop()
	require.Error(t, err)
	require.Equal(t, ErrNotRunning, err)
}

func TestStatus(t *testing.T) {
	srv := setupAPIServer(t)
	defer srv.Close()

	cfg := newConfig(t)
	cfg.ClusterConfig.Agents[0].ApiURL = srv.URL

	c, err := New(cfg, newLoadTestConfig(t), logger.New(&logger.Settings{}))
	require.NoError(t, err)
	require.NotNil(t, c)

	status, err := c.Status()
	require.NoError(t, err)
	require.Equal(t, Stopped, status.State)
	require.Empty(t, status)

	done, err := c.Run()
	require.NoError(t, err)
	require.NotNil(t, done)

	status, err = c.Status()
	require.NoError(t, err)
	require.Equal(t, Running, status.State)

	err = c.Stop()
	require.NoError(t, err)

	status, err = c.Status()
	require.NoError(t, err)
	require.Equal(t, Done, status.State)
	require.NotEmpty(t, status)
}

func TestConcurrency(t *testing.T) {
	srv := setupAPIServer(t)
	defer srv.Close()

	cfg := newConfig(t)
	cfg.ClusterConfig.Agents[0].ApiURL = srv.URL

	c, err := New(cfg, newLoadTestConfig(t), logger.New(&logger.Settings{}))
	require.NoError(t, err)
	require.NotNil(t, c)

	// level of concurrency
	n := 8

	t.Run("status", func(t *testing.T) {
		var wg sync.WaitGroup
		wg.Add(n)
		done, err := c.Run()
		require.NoError(t, err)
		require.NotNil(t, done)
		for i := 0; i < n; i++ {
			go func() {
				c.Status()
				wg.Done()
			}()
		}
		wg.Wait()
		err = c.Stop()
		require.NoError(t, err)
		c.mut.RLock()
		require.Equal(t, c.status.State, Done)
		c.mut.RUnlock()
	})
}
