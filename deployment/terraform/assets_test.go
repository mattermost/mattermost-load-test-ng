// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Test Asset function
func TestAsset(t *testing.T) {
	// Test reading a valid file
	data, err := Asset("outputs.tf")
	require.NoError(t, err)
	require.NotEmpty(t, data)

	// Convert to string and verify it contains expected content
	content := string(data)
	require.Contains(t, content, "output")
}

func TestAsset_NonExistentFile(t *testing.T) {
	// Test reading a non-existent file
	_, err := Asset("non_existent_file.txt")
	require.Error(t, err)
}

func TestAsset_ValidTerraformFile(t *testing.T) {
	// Test reading specific terraform files
	terraformFiles := []string{
		"outputs.tf",
		"variables.tf",
		"cluster.tf",
		"elasticsearch.tf",
	}

	for _, fileName := range terraformFiles {
		data, err := Asset(fileName)
		require.NoError(t, err, "Failed to read %s", fileName)
		require.NotEmpty(t, data, "File %s should not be empty", fileName)

		// Verify it's a terraform file by checking for terraform syntax
		content := string(data)
		require.True(t, strings.Contains(content, "resource") ||
			strings.Contains(content, "variable") ||
			strings.Contains(content, "output") ||
			strings.Contains(content, "data"),
			"File %s should contain terraform syntax", fileName)
	}
}

// Test MustAsset function
func TestMustAsset(t *testing.T) {
	// Test reading a valid file
	data := MustAsset("outputs.tf")
	require.NotEmpty(t, data)

	// Convert to string and verify it contains expected content
	content := string(data)
	require.Contains(t, content, "output")
}

func TestMustAsset_NonExistentFile_Panics(t *testing.T) {
	// Test that MustAsset panics on non-existent file
	require.Panics(t, func() {
		MustAsset("non_existent_file.txt")
	})
}

func TestMustAsset_PanicMessage(t *testing.T) {
	// Test that MustAsset panic message contains the file name
	defer func() {
		if r := recover(); r != nil {
			panicMsg := r.(string)
			require.Contains(t, panicMsg, "asset: Asset(non_existent_file.txt):")
		}
	}()

	MustAsset("non_existent_file.txt")
	t.Error("Expected panic did not occur")
}

// Test AssetString function
func TestAssetString(t *testing.T) {
	// Test reading a valid file as string
	content, err := AssetString("outputs.tf")
	require.NoError(t, err)
	require.NotEmpty(t, content)
	require.Contains(t, content, "output")

	// Compare with Asset function result
	data, err := Asset("outputs.tf")
	require.NoError(t, err)
	require.Equal(t, string(data), content)
}

func TestAssetString_NonExistentFile(t *testing.T) {
	// Test reading a non-existent file as string
	_, err := AssetString("non_existent_file.txt")
	require.Error(t, err)
}

func TestAssetString_ValidYamlFile(t *testing.T) {
	// Test reading YAML files
	yamlFiles := []string{
		"datasource.yaml",
		"dashboard.yaml",
	}

	for _, fileName := range yamlFiles {
		content, err := AssetString(fileName)
		require.NoError(t, err, "Failed to read %s", fileName)
		require.NotEmpty(t, content, "File %s should not be empty", fileName)

		// Verify it's a YAML file by checking for YAML syntax
		require.True(t, strings.Contains(content, ":") ||
			strings.Contains(content, "---") ||
			strings.Contains(content, "name:") ||
			strings.Contains(content, "type:"),
			"File %s should contain YAML syntax", fileName)
	}
}

// Test MustAssetString function
func TestMustAssetString(t *testing.T) {
	// Test reading a valid file as string
	content := MustAssetString("outputs.tf")
	require.NotEmpty(t, content)
	require.Contains(t, content, "output")

	// Compare with MustAsset function result
	data := MustAsset("outputs.tf")
	require.Equal(t, string(data), content)
}

func TestMustAssetString_NonExistentFile_Panics(t *testing.T) {
	// Test that MustAssetString panics on non-existent file
	require.Panics(t, func() {
		MustAssetString("non_existent_file.txt")
	})
}

func TestMustAssetString_PanicMessage(t *testing.T) {
	// Test that MustAssetString panic message contains the file name
	defer func() {
		if r := recover(); r != nil {
			panicMsg := r.(string)
			require.Contains(t, panicMsg, "asset: Asset(non_existent_file.txt):")
		}
	}()

	MustAssetString("non_existent_file.txt")
	t.Error("Expected panic did not occur")
}

// Test consistency between functions
func TestAssetFunctions_Consistency(t *testing.T) {
	fileName := "outputs.tf"

	// Get data using Asset
	data, err := Asset(fileName)
	require.NoError(t, err)

	// Get data using MustAsset
	mustData := MustAsset(fileName)

	// Get string using AssetString
	str, err := AssetString(fileName)
	require.NoError(t, err)

	// Get string using MustAssetString
	mustStr := MustAssetString(fileName)

	// All should be consistent
	require.Equal(t, data, mustData)
	require.Equal(t, string(data), str)
	require.Equal(t, string(mustData), mustStr)
	require.Equal(t, str, mustStr)
}

// Test with different file types
func TestAssetFunctions_DifferentFileTypes(t *testing.T) {
	testCases := []struct {
		fileName        string
		expectedContent string
	}{
		{"outputs.tf", "output"},
		{"variables.tf", "variable"},
		{"cluster.tf", "resource"},
		{"datasource.yaml", ":"},
		{"dashboard.yaml", ":"},
	}

	for _, tc := range testCases {
		t.Run(tc.fileName, func(t *testing.T) {
			// Test Asset
			data, err := Asset(tc.fileName)
			require.NoError(t, err)
			require.Contains(t, string(data), tc.expectedContent)

			// Test MustAsset
			mustData := MustAsset(tc.fileName)
			require.Equal(t, data, mustData)

			// Test AssetString
			str, err := AssetString(tc.fileName)
			require.NoError(t, err)
			require.Contains(t, str, tc.expectedContent)

			// Test MustAssetString
			mustStr := MustAssetString(tc.fileName)
			require.Equal(t, str, mustStr)
		})
	}
}

func TestRestoreAssets_SingleFile(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "restore_assets_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test restoring a single file
	err = RestoreAssets(tempDir, "outputs.tf")
	require.NoError(t, err)

	// Verify the file was created
	outputPath := filepath.Join(tempDir, "outputs.tf")
	_, err = os.Stat(outputPath)
	require.NoError(t, err)

	// Verify the file content is not empty
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)
	require.NotEmpty(t, content)
}

func TestRestoreAssets_Directory(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "restore_assets_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test restoring a directory
	err = RestoreAssets(tempDir, "provisioners")
	require.NoError(t, err)

	// Verify the directory was created
	provisionersPath := filepath.Join(tempDir, "provisioners")
	info, err := os.Stat(provisionersPath)
	require.NoError(t, err)
	require.True(t, info.IsDir())

	// Verify subdirectories exist
	debianPath := filepath.Join(provisionersPath, "debian")
	info, err = os.Stat(debianPath)
	require.NoError(t, err)
	require.True(t, info.IsDir())

	rhelPath := filepath.Join(provisionersPath, "rhel")
	info, err = os.Stat(rhelPath)
	require.NoError(t, err)
	require.True(t, info.IsDir())

	// Verify files exist in subdirectories
	expectedFiles := []string{
		"agent.sh",
		"app.sh",
		"common.sh",
		"job.sh",
		"keycloak.sh",
		"metrics.sh",
		"proxy.sh",
	}

	for _, fileName := range expectedFiles {
		debianFilePath := filepath.Join(debianPath, fileName)
		_, err = os.Stat(debianFilePath)
		require.NoError(t, err, "File %s should exist in debian directory", fileName)

		rhelFilePath := filepath.Join(rhelPath, fileName)
		_, err = os.Stat(rhelFilePath)
		require.NoError(t, err, "File %s should exist in rhel directory", fileName)

		// Verify files are not empty
		content, err := os.ReadFile(debianFilePath)
		require.NoError(t, err)
		require.NotEmpty(t, content, "File %s should not be empty", fileName)
	}
}

func TestRestoreAssets_NonExistentAsset(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "restore_assets_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test restoring a non-existent asset
	err = RestoreAssets(tempDir, "non_existent_file.txt")
	require.Error(t, err)
}

func TestRestoreAssets_MultipleFiles(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "restore_assets_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test restoring multiple files
	files := []string{
		"outputs.tf",
		"variables.tf",
		"cluster.tf",
		"elasticsearch.tf",
	}

	for _, fileName := range files {
		err = RestoreAssets(tempDir, fileName)
		require.NoError(t, err, "Failed to restore %s", fileName)

		// Verify the file was created
		outputPath := filepath.Join(tempDir, fileName)
		_, err = os.Stat(outputPath)
		require.NoError(t, err, "File %s should exist", fileName)

		// Verify the file content is not empty
		content, err := os.ReadFile(outputPath)
		require.NoError(t, err)
		require.NotEmpty(t, content, "File %s should not be empty", fileName)
	}
}

func TestRestoreAssets_FilePermissions(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "restore_assets_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test restoring a file and check permissions
	err = RestoreAssets(tempDir, "outputs.tf")
	require.NoError(t, err)

	// Verify the file permissions
	outputPath := filepath.Join(tempDir, "outputs.tf")
	info, err := os.Stat(outputPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0644), info.Mode().Perm())
}

func TestRestoreAssets_DirectoryPermissions(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "restore_assets_test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Test restoring a directory and check permissions
	err = RestoreAssets(tempDir, "provisioners")
	require.NoError(t, err)

	// Verify the directory permissions
	provisionersPath := filepath.Join(tempDir, "provisioners")
	info, err := os.Stat(provisionersPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0755), info.Mode().Perm())

	// Verify subdirectory permissions
	debianPath := filepath.Join(provisionersPath, "debian")
	info, err = os.Stat(debianPath)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0755), info.Mode().Perm())
}
