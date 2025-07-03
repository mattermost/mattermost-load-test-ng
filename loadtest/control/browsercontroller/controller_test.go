// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package browsercontroller

import (
	"os"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/loadtest/control"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/store/memstore"
	"github.com/mattermost/mattermost-load-test-ng/loadtest/user/userentity"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	// Set up environment variable
	originalEnv := os.Getenv("BROWSER_AGENT_API_URL")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("BROWSER_AGENT_API_URL")
		} else {
			os.Setenv("BROWSER_AGENT_API_URL", originalEnv)
		}
	}()

	// Create a test user
	store, err := memstore.New(nil)
	require.NoError(t, err)

	user := userentity.New(userentity.Setup{Store: store}, userentity.Config{
		ServerURL:    "http://localhost:8065",
		WebSocketURL: "ws://localhost:8065",
		Username:     "testuser",
		Email:        "test@example.com",
		Password:     "testpass",
	})

	statusChan := make(chan control.UserStatus, 10)

	// Test successful creation with environment variable set
	os.Setenv("BROWSER_AGENT_API_URL", "http://localhost:5000")
	controller, err := New(1, user, statusChan)
	require.NoError(t, err)
	require.NotNil(t, controller)
	require.Equal(t, 1, controller.id)
	require.Equal(t, user, controller.user)
	require.Equal(t, "http://localhost:5000", controller.browserAPIURL)
	require.Equal(t, 1.0, controller.rate)

	// Test with nil user
	controller, err = New(1, nil, statusChan)
	require.Error(t, err)
	require.Nil(t, controller)
	require.Contains(t, err.Error(), "user cannot be nil")

	// Test with missing environment variable
	os.Unsetenv("BROWSER_AGENT_API_URL")
	controller, err = New(1, user, statusChan)
	require.Error(t, err)
	require.Nil(t, controller)
	require.Contains(t, err.Error(), "BROWSER_AGENT_API_URL environment variable is required")

	// Test with nil status channel
	os.Setenv("BROWSER_AGENT_API_URL", "http://localhost:5000")
	controller, err = New(1, user, nil)
	require.Error(t, err)
	require.Nil(t, controller)
	require.Contains(t, err.Error(), "status channel cannot be nil")

	close(statusChan)
}

func TestSetRate(t *testing.T) {
	// Set up environment variable
	originalEnv := os.Getenv("BROWSER_AGENT_API_URL")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("BROWSER_AGENT_API_URL")
		} else {
			os.Setenv("BROWSER_AGENT_API_URL", originalEnv)
		}
	}()
	os.Setenv("BROWSER_AGENT_API_URL", "http://localhost:5000")

	// Create a test user
	store, err := memstore.New(nil)
	require.NoError(t, err)

	user := userentity.New(userentity.Setup{Store: store}, userentity.Config{
		ServerURL: "http://localhost:8065",
		Username:  "testuser",
		Password:  "testpass",
	})

	statusChan := make(chan control.UserStatus, 10)
	defer close(statusChan)

	controller, err := New(1, user, statusChan)
	require.NoError(t, err)

	// Test setting any rate (should always succeed as it's a no-op)
	err = controller.SetRate(2.5)
	require.NoError(t, err)

	err = controller.SetRate(0.0)
	require.NoError(t, err)

	err = controller.SetRate(-1.0)
	require.NoError(t, err)
}

func TestInjectAction(t *testing.T) {
	// Set up environment variable
	originalEnv := os.Getenv("BROWSER_AGENT_API_URL")
	defer func() {
		if originalEnv == "" {
			os.Unsetenv("BROWSER_AGENT_API_URL")
		} else {
			os.Setenv("BROWSER_AGENT_API_URL", originalEnv)
		}
	}()
	os.Setenv("BROWSER_AGENT_API_URL", "http://localhost:5000")

	// Create a test user
	store, err := memstore.New(nil)
	require.NoError(t, err)

	user := userentity.New(userentity.Setup{Store: store}, userentity.Config{
		ServerURL: "http://localhost:8065",
		Username:  "testuser",
		Password:  "testpass",
	})

	statusChan := make(chan control.UserStatus, 10)
	defer close(statusChan)

	controller, err := New(1, user, statusChan)
	require.NoError(t, err)

	// Test injecting action (should always succeed as it's a no-op)
	err = controller.InjectAction("test-action")
	require.NoError(t, err)
}

func TestBrowserControllerImplementsInterface(t *testing.T) {
	// This test ensures that BrowserController implements the UserController interface
	var _ control.UserController = (*BrowserController)(nil)
}
