// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package api

import (
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	agentClient "github.com/mattermost/mattermost-load-test-ng/api/client/agent"
	coordClient "github.com/mattermost/mattermost-load-test-ng/api/client/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/coordinator"
	"github.com/mattermost/mattermost-load-test-ng/defaults"
	"github.com/mattermost/mattermost-load-test-ng/loadtest"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/control/simulcontroller"
	"github.com/mattermost/mattermost-load-test-ng/logger"

	"github.com/stretchr/testify/require"
)

// concurrency level
const n = 8

func TestAgentClientConcurrency(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	var wg sync.WaitGroup

	t.Run("create", func(t *testing.T) {
		id := "test"
		var success int
		var ltConfig loadtest.Config
		var ucConfig simulcontroller.Config
		defaults.Set(&ltConfig)
		defaults.Set(&ucConfig)
		agent, err := agentClient.New(id, server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, agent)
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				if _, err := agent.Create(&ltConfig, &ucConfig); err == nil {
					// Purposely not using atomics here. The race condition would only
					// trigger if the test were to fail.
					success += 1
				}
				wg.Done()
			}()
		}
		wg.Wait()
		// Only the one attempt should have succeeded.
		require.Equal(t, 1, success)

		_, err = agent.Run()
		require.NoError(t, err)
		_, err = agent.AddUsers(10)
		require.NoError(t, err)

		success = 0
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				_, err := agent.Destroy()
				if err == nil {
					// Purposely not using atomics here. The race condition would only
					// trigger if the test were to fail.
					success += 1
				}
				wg.Done()
			}()
		}
		wg.Wait()
		// Only the one attempt should have succeeded.
		require.Equal(t, 1, success)
	})

	t.Run("create/destroy", func(t *testing.T) {
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func(id string) {
				agent := createAgent(t, id, server.URL)
				require.Equal(t, id, agent.Id())
				_, err := agent.Destroy()
				require.NoError(t, err)
				wg.Done()
			}(fmt.Sprintf("agent-%d", i))
		}
		wg.Wait()
	})

	t.Run("run/stop", func(t *testing.T) {
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func(id string) {
				agent := createAgent(t, id, server.URL)
				require.Equal(t, id, agent.Id())
				st, err := agent.Run()
				require.NoError(t, err)
				require.Equal(t, loadtest.Running, st.State)
				time.Sleep(100 * time.Millisecond)
				st, err = agent.Stop()
				require.NoError(t, err)
				require.Equal(t, loadtest.Stopped, st.State)
				st, err = agent.Run()
				require.NoError(t, err)
				require.Equal(t, loadtest.Running, st.State)
				time.Sleep(100 * time.Millisecond)
				st, err = agent.Stop()
				require.NoError(t, err)
				require.Equal(t, loadtest.Stopped, st.State)
				wg.Done()
			}(fmt.Sprintf("agent-%d", i))
		}
		wg.Wait()
	})

	t.Run("status", func(t *testing.T) {
		id := "test"
		agent := createAgent(t, id, server.URL)
		require.Equal(t, id, agent.Id())
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				st, err := agent.Status()
				require.NoError(t, err)
				require.Empty(t, st)
				wg.Done()
			}()
		}
		wg.Wait()
		_, err := agent.Destroy()
		require.NoError(t, err)
	})

	t.Run("add/rm users", func(t *testing.T) {
		id := "test"
		agent := createAgent(t, id, server.URL)
		require.Equal(t, id, agent.Id())
		st, err := agent.Run()
		require.NoError(t, err)
		require.Equal(t, loadtest.Running, st.State)
		st, err = agent.AddUsers(n)
		require.NoError(t, err)
		require.Equal(t, int64(n), st.NumUsers)
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func(add bool) {
				defer wg.Done()
				if add {
					st, err := agent.AddUsers(2)
					require.NoError(t, err)
					require.NotEmpty(t, st)
					return
				}
				st, err := agent.RemoveUsers(2)
				require.NoError(t, err)
				require.NotEmpty(t, st)
			}(i%2 == 0)
		}
		wg.Wait()
		_, err = agent.Destroy()
		require.NoError(t, err)
	})
}

func TestCoordClientConcurrency(t *testing.T) {
	// create http.Handler
	handler := SetupAPIRouter(logger.New(&logger.Settings{}), logger.New(&logger.Settings{}))

	// run server using httptest
	server := httptest.NewServer(handler)
	defer server.Close()

	var wg sync.WaitGroup

	createClient := func(t *testing.T, i int) *coordClient.Coordinator {
		t.Helper()
		id := fmt.Sprintf("coord-%d", i)
		coord, err := coordClient.New(id, server.URL, nil)
		require.NoError(t, err)
		require.NotNil(t, coord)
		return coord
	}

	create := func(t *testing.T, i int) *coordClient.Coordinator {
		t.Helper()
		coord := createClient(t, i)
		var coordConfig coordinator.Config
		var ltConfig loadtest.Config
		defaults.Set(&coordConfig)
		defaults.Set(&ltConfig)
		coordConfig.ClusterConfig.Agents[0].Id = coord.Id() + "-agent"
		coordConfig.ClusterConfig.Agents[0].ApiURL = server.URL
		coordConfig.MonitorConfig.Queries[0].Description = "Query"
		coordConfig.MonitorConfig.Queries[0].Query = "query"
		_, err := coord.Create(&coordConfig, &ltConfig)
		require.NoError(t, err)
		return coord
	}

	t.Run("create/destroy", func(t *testing.T) {
		coord := createClient(t, 0)
		var coordConfig coordinator.Config
		var ltConfig loadtest.Config
		defaults.Set(&coordConfig)
		defaults.Set(&ltConfig)
		coordConfig.ClusterConfig.Agents[0].Id = coord.Id() + "-agent"
		coordConfig.ClusterConfig.Agents[0].ApiURL = server.URL
		coordConfig.MonitorConfig.Queries[0].Description = "Query"
		coordConfig.MonitorConfig.Queries[0].Query = "query"
		var success int
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				_, err := coord.Create(&coordConfig, &ltConfig)
				if err == nil {
					// Purposely not using atomics here. The race condition would only
					// trigger if the test were to fail.
					success += 1
				}
				wg.Done()
			}()
		}
		wg.Wait()
		// Only the one attempt should have succeeded.
		require.Equal(t, 1, success)

		_, err := coord.Run()
		require.NoError(t, err)

		success = 0
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				_, err := coord.Destroy()
				if err == nil {
					// Purposely not using atomics here. The race condition would only
					// trigger if the test were to fail.
					success += 1
				}
				wg.Done()
			}()
		}
		wg.Wait()
		// Only the one attempt should have succeeded.
		require.Equal(t, 1, success)
	})

	t.Run("run/status/stop", func(t *testing.T) {
		wg.Add(n)
		for i := 0; i < n; i++ {
			id := fmt.Sprintf("coord-%d", i)
			coord := create(t, i)
			require.Equal(t, id, coord.Id())
			go func() {
				st, err := coord.Run()
				require.NoError(t, err)
				require.Equal(t, coordinator.Running, st.State)
				time.Sleep(100 * time.Millisecond)
				st, err = coord.Status()
				require.NoError(t, err)
				require.NotEmpty(t, st)
				st, err = coord.Stop()
				require.NoError(t, err)
				require.Equal(t, coordinator.Done, st.State)
				wg.Done()
			}()
		}
		wg.Wait()
	})
}
