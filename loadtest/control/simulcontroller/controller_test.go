// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package simulcontroller

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/plugins"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/stretchr/testify/require"
)

// pinAgentsLoadtestEnv writes a deterministic Agents-only config so tests do not depend on an
// ignored local ../config/mattermost-ai-loadtest.json.
func pinAgentsLoadtestEnv(t *testing.T) {
	t.Helper()

	const agentsJSON = `{
  "triggerFrequencyChannelMention": 0.001,
  "triggerFrequencyDM": 0.001,
  "agentUsername": "ai",
  "triggerMode": "both",
  "promptProfile": "mixed"
}
`
	path := filepath.Join(t.TempDir(), "mattermost-ai-loadtest.json")
	require.NoError(t, os.WriteFile(path, []byte(agentsJSON), 0600))
	t.Setenv("MM_AGENTS_LOADTEST_CONFIG", path)
}

func newController(t *testing.T) (*SimulController, chan control.UserStatus) {
	t.Helper()

	pinAgentsLoadtestEnv(t)

	config, err := ReadConfig("../../../config/simulcontroller.sample.json")
	require.NoError(t, err)
	require.NotNil(t, config)

	store, err := memstore.New(nil)
	require.NotNil(t, store)
	require.NoError(t, err)

	user := userentity.New(userentity.Setup{Store: store}, userentity.Config{
		ServerURL:    "http://localhost:8065",
		WebSocketURL: "ws://localhost:8065",
	})

	statusChan := make(chan control.UserStatus)

	c, err := New(1, user, config, statusChan)
	require.NoError(t, err)

	return c, statusChan
}

func TestNew(t *testing.T) {
	c, statusChan := newController(t)
	close(statusChan) // not used

	require.Equal(t, len(c.actionList), len(c.actionMap))
}

func TestSetRate(t *testing.T) {
	c, statusChan := newController(t)
	close(statusChan) // not used
	require.Equal(t, 1.0, c.rate)

	err := c.SetRate(-1.0)
	require.NotNil(t, err)
	require.Equal(t, 1.0, c.rate)

	err = c.SetRate(0.0)
	require.Nil(t, err)
	require.Equal(t, 0.0, c.rate)

	err = c.SetRate(1.5)
	require.Nil(t, err)
	require.Equal(t, 1.5, c.rate)
}

func TestRunStop(t *testing.T) {
	c, statusChan := newController(t)

	doneRunning := make(chan struct{})
	go func() {
		c.Run()
		close(doneRunning)
	}()

	status := <-statusChan
	require.NoError(t, status.Err)
	require.Equal(t, "user started", status.Info)

	doneHandlingStatus := make(chan struct{})
	go func() {
		var last control.UserStatus
		for {
			status, ok := <-statusChan
			if !ok {
				require.Equal(t, "user stopped", last.Info)
				break
			}
			last = status
		}
		close(doneHandlingStatus)
	}()

	c.Stop()
	<-doneRunning
	close(statusChan)
	<-doneHandlingStatus
}

func TestGetActionList(t *testing.T) {
	c, statusChan := newController(t)
	close(statusChan) // not used
	for _, action := range getActionList(c) {
		require.NotZero(t, action.minServerVersion, "All actions must have minServerVersion set")
	}
}

func TestAgentsPluginRegistersSimulController(t *testing.T) {
	found := false
	plugins.SpawnPluginControllers(plugins.TypeSimulController, func(p plugins.Controller) {
		if p.PluginId() != "mattermost-ai" {
			return
		}
		_, ok := p.(plugins.SimulController)
		require.True(t, ok, "mattermost-ai must implement plugins.SimulController")
		found = true
	})
	require.True(t, found, "mattermost-ai SimulController should be registered via blank import wiring")
}

func TestAgentsActionsInActionMapWhenEnabled(t *testing.T) {
	c, statusChan := newController(t)
	close(statusChan)

	var ids []string
	for _, p := range c.plugins {
		ids = append(ids, p.PluginId())
	}
	require.Contains(t, ids, "mattermost-ai", "sample config should enable mattermost-ai among other plugins")

	const chAction = "mattermost-ai.AskAgentChannelMention"
	const dmAction = "mattermost-ai.AskAgentDM"
	ch, okCH := c.actionMap[chAction]
	require.True(t, okCH, "prefixed Agents channel action should appear in actionMap")
	require.InDelta(t, 0.001, ch.frequency, 1e-9)
	dm, okDM := c.actionMap[dmAction]
	require.True(t, okDM, "prefixed Agents DM action should appear in actionMap")
	require.InDelta(t, 0.001, dm.frequency, 1e-9)

	_, hasBare := c.actionMap["AskAgentChannelMention"]
	require.False(t, hasBare, "Agents actions must be prefixed by plugin ID in maps and logs")

	_, hasBareDM := c.actionMap["AskAgentDM"]
	require.False(t, hasBareDM, "Agents actions must be prefixed by plugin ID in maps and logs")
}
