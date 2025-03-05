// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

//go:build integration
// +build integration

package version_test

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/mattermost/mattermost-load-test-ng/version"
	"github.com/stretchr/testify/require"
)

func TestVersionInfoIntegration(t *testing.T) {
	// Check if go is available
	_, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go not found in PATH, skipping integration test")
	}

	// Create a temporary directory for our test binary
	tempDir, err := os.MkdirTemp("", "ltapi-test")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	// Build the ltapi binary in the temp directory
	apiServerPath := filepath.Join(tempDir, "ltapi")

	// Build the binary directly with go build
	buildCmd := exec.Command("go", "build", "-o", apiServerPath, "../cmd/ltapi")
	buildOutput, err := buildCmd.CombinedOutput()
	require.NoError(t, err, "Failed to build binary: %s", string(buildOutput))

	// Start the API server
	apiCmd := exec.Command(apiServerPath)
	err = apiCmd.Start()
	require.NoError(t, err, "Failed to start API server")

	// Ensure we clean up the API server process when the test finishes
	defer func() {
		if apiCmd.Process != nil {
			apiCmd.Process.Kill()
		}
	}()

	// Wait for the server to start
	time.Sleep(2 * time.Second)

	// Make a request to the API server's version endpoint
	resp, err := http.Get("http://localhost:4000/version")
	require.NoError(t, err, "Failed to make request to API server")
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")

	// Unmarshal the response into a VersionInfo struct
	var info version.VersionInfo
	err = json.Unmarshal(body, &info)
	require.NoError(t, err, "Failed to unmarshal response body")

	// Verify the commit is a valid SHA (40 hex characters)
	require.NotEqual(t, "unknown", info.Commit, "Commit should not be 'unknown'")
	require.Regexp(t, regexp.MustCompile(`^[0-9a-f]{40}$`), info.Commit, "Commit should be a 40-character hex SHA")

	// Verify the build time is not zero
	require.False(t, info.BuildTime.IsZero(), "BuildTime should not be zero")
}
