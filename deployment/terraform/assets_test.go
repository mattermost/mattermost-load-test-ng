// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAsset(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		data, err := Asset("outputs.tf")
		require.NoError(t, err)
		require.NotEmpty(t, data)

		content := string(data)
		require.Contains(t, content, "output")
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := Asset("non_existent_file.txt")
		require.Error(t, err)
	})
}

func TestMustAsset(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		data := MustAsset("outputs.tf")
		require.NotEmpty(t, data)

		content := string(data)
		require.Contains(t, content, "output")
	})

	t.Run("non-existent file panics", func(t *testing.T) {
		require.Panics(t, func() {
			MustAsset("non_existent_file.txt")
		})
	})

	t.Run("panic message format", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				panicMsg := r.(string)
				require.Contains(t, panicMsg, "asset: Asset(non_existent_file.txt):")
			}
		}()

		MustAsset("non_existent_file.txt")
		t.Error("Expected panic did not occur")
	})
}

func TestAssetString(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		content, err := AssetString("outputs.tf")
		require.NoError(t, err)
		require.NotEmpty(t, content)
		require.Contains(t, content, "output")

		// Compare with Asset function result
		data, err := Asset("outputs.tf")
		require.NoError(t, err)
		require.Equal(t, string(data), content)
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := AssetString("non_existent_file.txt")
		require.Error(t, err)
	})

}

func TestMustAssetString(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		content := MustAssetString("outputs.tf")
		require.NotEmpty(t, content)
		require.Contains(t, content, "output")

		// Compare with MustAsset function result
		data := MustAsset("outputs.tf")
		require.Equal(t, string(data), content)
	})

	t.Run("non-existent file panics", func(t *testing.T) {
		require.Panics(t, func() {
			MustAssetString("non_existent_file.txt")
		})
	})

	t.Run("panic message format", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				panicMsg := r.(string)
				require.Contains(t, panicMsg, "asset: Asset(non_existent_file.txt):")
			}
		}()

		MustAssetString("non_existent_file.txt")
		t.Error("Expected panic did not occur")
	})
}

func TestRestoreAssetFile(t *testing.T) {
	t.Run("successful file creation", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "restore_asset_file_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		testData := []byte("test file content")
		targetPath := filepath.Join(tempDir, "subdir", "testfile.txt")

		err = restoreAssetFile(testData, targetPath)
		require.NoError(t, err)

		// Verify the file was created
		_, err = os.Stat(targetPath)
		require.NoError(t, err)

		// Verify the file content
		content, err := os.ReadFile(targetPath)
		require.NoError(t, err)
		require.Equal(t, testData, content)

		// Verify file permissions
		info, err := os.Stat(targetPath)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0644), info.Mode().Perm())

		// Verify directory was created with correct permissions
		dirInfo, err := os.Stat(filepath.Dir(targetPath))
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0755), dirInfo.Mode().Perm())
	})

	t.Run("creates nested directories", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "restore_asset_file_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		testData := []byte("test file content")
		targetPath := filepath.Join(tempDir, "level1", "level2", "level3", "testfile.txt")

		err = restoreAssetFile(testData, targetPath)
		require.NoError(t, err)

		// Verify the nested directories were created
		level1Path := filepath.Join(tempDir, "level1")
		info, err := os.Stat(level1Path)
		require.NoError(t, err)
		require.True(t, info.IsDir())
		require.Equal(t, os.FileMode(0755), info.Mode().Perm())

		level2Path := filepath.Join(tempDir, "level1", "level2")
		info, err = os.Stat(level2Path)
		require.NoError(t, err)
		require.True(t, info.IsDir())
		require.Equal(t, os.FileMode(0755), info.Mode().Perm())

		level3Path := filepath.Join(tempDir, "level1", "level2", "level3")
		info, err = os.Stat(level3Path)
		require.NoError(t, err)
		require.True(t, info.IsDir())
		require.Equal(t, os.FileMode(0755), info.Mode().Perm())

		// Verify the file was created
		_, err = os.Stat(targetPath)
		require.NoError(t, err)

		// Verify the file content
		content, err := os.ReadFile(targetPath)
		require.NoError(t, err)
		require.Equal(t, testData, content)
	})

	t.Run("invalid path", func(t *testing.T) {
		testData := []byte("test content")
		invalidPath := "/invalid/readonly/path/file.txt"
		err := restoreAssetFile(testData, invalidPath)
		require.Error(t, err)
	})
}

func TestRestoreAssets(t *testing.T) {
	t.Run("single file", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "restore_assets_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

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

		// Verify file permissions
		info, err := os.Stat(outputPath)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0644), info.Mode().Perm())
	})

	t.Run("directory recursive", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "restore_assets_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		err = RestoreAssets(tempDir, "provisioners")
		require.NoError(t, err)

		// Verify the directory was created
		provisionersPath := filepath.Join(tempDir, "provisioners")
		info, err := os.Stat(provisionersPath)
		require.NoError(t, err)
		require.True(t, info.IsDir())
		require.Equal(t, os.FileMode(0755), info.Mode().Perm())

		// Verify subdirectories exist
		subdirs := []string{"debian", "rhel"}
		for _, subdir := range subdirs {
			subdirPath := filepath.Join(provisionersPath, subdir)
			info, err := os.Stat(subdirPath)
			require.NoError(t, err)
			require.True(t, info.IsDir())
			require.Equal(t, os.FileMode(0755), info.Mode().Perm())
		}

		// Verify files exist in subdirectories
		expectedFiles := []string{
			"agent.sh", "app.sh", "common.sh", "job.sh",
			"keycloak.sh", "metrics.sh", "proxy.sh",
		}

		for _, subdir := range subdirs {
			for _, fileName := range expectedFiles {
				filePath := filepath.Join(provisionersPath, subdir, fileName)
				_, err = os.Stat(filePath)
				require.NoError(t, err, "File %s should exist in %s directory", fileName, subdir)

				// Verify files are not empty
				content, err := os.ReadFile(filePath)
				require.NoError(t, err)
				require.NotEmpty(t, content, "File %s should not be empty", fileName)
			}
		}
	})

	t.Run("multiple files", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "restore_assets_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		files := []string{"outputs.tf", "variables.tf", "cluster.tf", "elasticsearch.tf"}

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
	})

	t.Run("non-existent asset", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "restore_assets_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		err = RestoreAssets(tempDir, "non_existent_file.txt")
		require.Error(t, err)
	})
}
