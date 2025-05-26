package terraform

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/mattermost/mattermost-load-test-ng/deployment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDir creates a temporary directory for testing and returns a config with it
func setupTestDir(t *testing.T) deployment.Config {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "generated_values_test")
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	cfg := deployment.Config{
		TerraformStateDir: tempDir,
	}

	return cfg
}

func TestGetValuesPath(t *testing.T) {
	cfg := deployment.Config{
		TerraformStateDir: "/test/path",
	}

	for _, id := range []string{"", "someid"} {
		t.Run(fmt.Sprintf("Get values path, id %q", id), func(t *testing.T) {
			path := getValuesPath(id, cfg)

			expectedFileName := genValuesFileName
			if id != "" {
				expectedFileName = id + "_" + genValuesFileName
			}

			expected := filepath.Join("/test/path", expectedFileName)
			assert.Equal(t, expected, path)
		})
	}
}

func TestSanitize(t *testing.T) {
	t.Run("Sanitize values", func(t *testing.T) {
		// Create test values with sensitive data
		values := GeneratedValues{
			GrafanaAdminPassword: "secret-password",
		}

		// Sanitize the values
		sanitized := values.Sanitize()

		// Verify the original values are unchanged
		assert.Equal(t, "secret-password", values.GrafanaAdminPassword)

		// Verify the sensitive fields are masked in the sanitized copy
		assert.Equal(t, "********", sanitized.GrafanaAdminPassword)
	})
}

func TestOpenValuesFile(t *testing.T) {
	cfg := setupTestDir(t)

	for _, id := range []string{"", "someid"} {
		// Test successful file opening
		t.Run(fmt.Sprintf("Open values file successfully, id %q", id), func(t *testing.T) {
			file, err := openValuesFile(id, cfg)
			require.NoError(t, err)
			defer file.Close()
			assert.NotNil(t, file)
		})

		// Test with invalid directory
		t.Run(fmt.Sprintf("Invalid directory, id %q", id), func(t *testing.T) {
			invalidCfg := deployment.Config{
				TerraformStateDir: "/path/that/does/not/exist",
			}
			_, err := openValuesFile(id, invalidCfg)
			assert.Error(t, err)
		})
	}
}

func TestReadGenValues(t *testing.T) {
	cfg := setupTestDir(t)

	for _, id := range []string{"", "someid"} {
		// Test reading from an empty/non-existent file
		t.Run(fmt.Sprintf("Read empty file, id %q", id), func(t *testing.T) {
			values, err := readGenValues(id, cfg)
			require.NoError(t, err)
			assert.NotNil(t, values)
			assert.Empty(t, values.GrafanaAdminPassword)
		})

		// Test reading malformed JSON
		t.Run(fmt.Sprintf("Read malformed JSON, id %q", id), func(t *testing.T) {
			// Write malformed JSON to the file
			filePath := getValuesPath(id, cfg)
			err := os.WriteFile(filePath, []byte("this is not valid json"), 0644)
			require.NoError(t, err)

			// Try to read the values
			_, err = readGenValues(id, cfg)
			assert.Error(t, err)
		})

		// Test reading valid JSON
		t.Run(fmt.Sprintf("Read valid JSON, id %q", id), func(t *testing.T) {
			// Write valid JSON to the file
			filePath := getValuesPath(id, cfg)
			testValues := &GeneratedValues{
				GrafanaAdminPassword: "test-password",
			}
			jsonData, err := json.Marshal(testValues)
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(filePath, jsonData, 0644))

			// Read the values
			readValues, err := readGenValues(id, cfg)
			require.NoError(t, err)
			assert.Equal(t, testValues.GrafanaAdminPassword, readValues.GrafanaAdminPassword)
		})
	}
}

func TestPersistGeneratedValues(t *testing.T) {
	cfg := setupTestDir(t)

	for _, id := range []string{"", "someid"} {
		// Test persisting values to a new file
		t.Run(fmt.Sprintf("Persist to new file, id %q", id), func(t *testing.T) {
			// Create test values
			testValues := &GeneratedValues{
				GrafanaAdminPassword: "test-password",
			}

			// Persist the values
			err := persistGeneratedValues(id, cfg, testValues)
			require.NoError(t, err)

			// Verify the file exists
			filePath := getValuesPath(id, cfg)
			_, err = os.Stat(filePath)
			require.NoError(t, err)

			// Read the file directly and verify JSON content
			fileContent, err := os.ReadFile(filePath)
			require.NoError(t, err)

			var parsedValues GeneratedValues
			err = json.Unmarshal(fileContent, &parsedValues)
			require.NoError(t, err)
			assert.Equal(t, testValues.GrafanaAdminPassword, parsedValues.GrafanaAdminPassword)
		})

		// Test updating existing values
		t.Run(fmt.Sprintf("Update existing values, id %q", id), func(t *testing.T) {
			// Create initial values
			initialValues := &GeneratedValues{
				GrafanaAdminPassword: "initial-password",
			}

			// Persist the initial values
			err := persistGeneratedValues(id, cfg, initialValues)
			require.NoError(t, err)

			// Update with new values
			updatedValues := &GeneratedValues{
				GrafanaAdminPassword: "updated-password",
			}

			// Persist the updated values
			err = persistGeneratedValues(id, cfg, updatedValues)
			require.NoError(t, err)

			// Read the values back
			readValues, err := readGenValues(id, cfg)
			require.NoError(t, err)
			assert.Equal(t, updatedValues.GrafanaAdminPassword, readValues.GrafanaAdminPassword)
		})
	}
}

func TestIntegration(t *testing.T) {
	cfg := setupTestDir(t)

	for _, id := range []string{"", "someid"} {
		// Test the full workflow: persist, read, update
		t.Run(fmt.Sprintf("Full workflow, id %q", id), func(t *testing.T) {
			// Create and persist initial values
			initialValues := &GeneratedValues{
				GrafanaAdminPassword: "initial-password",
			}
			err := persistGeneratedValues(id, cfg, initialValues)
			require.NoError(t, err)

			// Read the values back
			readValues, err := readGenValues(id, cfg)
			require.NoError(t, err)
			assert.Equal(t, initialValues.GrafanaAdminPassword, readValues.GrafanaAdminPassword)

			// Update the values
			updatedValues := &GeneratedValues{
				GrafanaAdminPassword: "updated-password",
			}
			err = persistGeneratedValues(id, cfg, updatedValues)
			require.NoError(t, err)

			// Read the updated values
			readUpdatedValues, err := readGenValues(id, cfg)
			require.NoError(t, err)
			assert.Equal(t, updatedValues.GrafanaAdminPassword, readUpdatedValues.GrafanaAdminPassword)
		})
	}
}
