package terraform

import (
	"encoding/json"
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

func TestOpenValuesFile(t *testing.T) {
	cfg := setupTestDir(t)

	// Test successful file opening
	t.Run("Open values file successfully", func(t *testing.T) {
		file, err := openValuesFile(cfg)
		require.NoError(t, err)
		defer file.Close()
		assert.NotNil(t, file)
	})

	// Test with invalid directory
	t.Run("Invalid directory", func(t *testing.T) {
		invalidCfg := deployment.Config{
			TerraformStateDir: "/path/that/does/not/exist",
		}
		_, err := openValuesFile(invalidCfg)
		assert.Error(t, err)
	})
}

func TestReadGenValues(t *testing.T) {
	cfg := setupTestDir(t)
	tempDir := cfg.TerraformStateDir

	// Test reading from an empty/non-existent file
	t.Run("Read empty file", func(t *testing.T) {
		values, err := readGenValues(cfg)
		require.NoError(t, err)
		assert.NotNil(t, values)
		assert.Empty(t, values.GrafanaAdminPassword)
	})

	// Test reading malformed JSON
	t.Run("Read malformed JSON", func(t *testing.T) {
		// Write malformed JSON to the file
		filePath := filepath.Join(tempDir, genValuesFileName)
		err := os.WriteFile(filePath, []byte("this is not valid json"), 0644)
		require.NoError(t, err)

		// Try to read the values
		_, err = readGenValues(cfg)
		assert.Error(t, err)
	})

	// Test reading valid JSON
	t.Run("Read valid JSON", func(t *testing.T) {
		// Write valid JSON to the file
		filePath := filepath.Join(tempDir, genValuesFileName)
		testValues := &GeneratedValues{
			GrafanaAdminPassword: "test-password",
		}
		jsonData, err := json.Marshal(testValues)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filePath, jsonData, 0644))

		// Read the values
		readValues, err := readGenValues(cfg)
		require.NoError(t, err)
		assert.Equal(t, testValues.GrafanaAdminPassword, readValues.GrafanaAdminPassword)
	})
}

func TestPersistGeneratedValues(t *testing.T) {
	cfg := setupTestDir(t)
	tempDir := cfg.TerraformStateDir

	// Test persisting values to a new file
	t.Run("Persist to new file", func(t *testing.T) {
		// Create test values
		testValues := &GeneratedValues{
			GrafanaAdminPassword: "test-password",
		}

		// Persist the values
		err := persistGeneratedValues(cfg, testValues)
		require.NoError(t, err)

		// Verify the file exists
		filePath := filepath.Join(tempDir, genValuesFileName)
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
	t.Run("Update existing values", func(t *testing.T) {
		// Create initial values
		initialValues := &GeneratedValues{
			GrafanaAdminPassword: "initial-password",
		}

		// Persist the initial values
		err := persistGeneratedValues(cfg, initialValues)
		require.NoError(t, err)

		// Update with new values
		updatedValues := &GeneratedValues{
			GrafanaAdminPassword: "updated-password",
		}

		// Persist the updated values
		err = persistGeneratedValues(cfg, updatedValues)
		require.NoError(t, err)

		// Read the values back
		readValues, err := readGenValues(cfg)
		require.NoError(t, err)
		assert.Equal(t, updatedValues.GrafanaAdminPassword, readValues.GrafanaAdminPassword)
	})
}

func TestIntegration(t *testing.T) {
	cfg := setupTestDir(t)

	// Test the full workflow: persist, read, update
	t.Run("Full workflow", func(t *testing.T) {
		// Create and persist initial values
		initialValues := &GeneratedValues{
			GrafanaAdminPassword: "initial-password",
		}
		err := persistGeneratedValues(cfg, initialValues)
		require.NoError(t, err)

		// Read the values back
		readValues, err := readGenValues(cfg)
		require.NoError(t, err)
		assert.Equal(t, initialValues.GrafanaAdminPassword, readValues.GrafanaAdminPassword)

		// Update the values
		updatedValues := &GeneratedValues{
			GrafanaAdminPassword: "updated-password",
		}
		err = persistGeneratedValues(cfg, updatedValues)
		require.NoError(t, err)

		// Read the updated values
		readUpdatedValues, err := readGenValues(cfg)
		require.NoError(t, err)
		assert.Equal(t, updatedValues.GrafanaAdminPassword, readUpdatedValues.GrafanaAdminPassword)
	})
}
