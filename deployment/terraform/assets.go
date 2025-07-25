// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed assets/*
var assetsFS embed.FS

// Asset returns the content of the embedded file
func Asset(name string) ([]byte, error) {
	return assetsFS.ReadFile("assets/" + name)
}

// MustAsset returns the content of the embedded file and panics on error
func MustAsset(name string) []byte {
	data, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}
	return data
}

// AssetString returns the content of the embedded file as string
func AssetString(name string) (string, error) {
	data, err := Asset(name)
	return string(data), err
}

// MustAssetString returns the content of the embedded file as string and panics on error
func MustAssetString(name string) string {
	return string(MustAsset(name))
}

// restoreAssetFile writes a single embedded file to the specified path
func restoreAssetFile(data []byte, targetPath string) error {
	// Create parent directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(targetPath, data, 0644)
}

// RestoreAssets writes an embedded asset to the given directory
// If name is a file, it writes the file. If name is a directory, it recursively writes all files in the directory.
func RestoreAssets(dir, name string) error {
	assetPath := filepath.Join("assets", name)

	fileInfo, err := fs.Stat(assetsFS, assetPath)
	if err != nil {
		return err
	}

	// Check if it's a directory
	if fileInfo.IsDir() {
		// It's a directory, copy recursively using WalkDir
		return fs.WalkDir(assetsFS, assetPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Calculate the relative path from the asset root
			relPath, err := filepath.Rel(assetPath, path)
			if err != nil {
				return err
			}

			// Skip the root directory itself
			if relPath == "." {
				return nil
			}

			targetPath := filepath.Join(dir, name, relPath)

			// Check if it's a directory and only proceed for files, since files are the only ones we want to copy
			// and we make sure the target directory exists before copying. This prevents empty folders.
			if !d.IsDir() {
				data, err := assetsFS.ReadFile(path)
				if err != nil {
					return err
				}

				return restoreAssetFile(data, targetPath)
			}

			return nil
		})
	}

	// It's a file, copy it to the destination
	data, err := Asset(name)
	if err != nil {
		return err
	}

	outputPath := filepath.Join(dir, name)
	return restoreAssetFile(data, outputPath)
}

// restoreDir recursively extracts a directory from the embedded filesystem
func restoreDir(baseDir, relPath, assetPath string) error {
	entries, err := fs.ReadDir(assetsFS, assetPath)
	if err != nil {
		return fmt.Errorf("failed to read directory %s: %w", assetPath, err)
	}

	for _, entry := range entries {
		entryRelPath := filepath.Join(relPath, entry.Name())
		entryAssetPath := filepath.Join(assetPath, entry.Name())

		if entry.IsDir() {
			// Recursively handle subdirectories
			if err := restoreDir(baseDir, entryRelPath, entryAssetPath); err != nil {
				return err
			}
		} else {
			// Extract the file
			data, err := assetsFS.ReadFile(entryAssetPath)
			if err != nil {
				return fmt.Errorf("failed to read file %s: %w", entryAssetPath, err)
			}

			outputPath := filepath.Join(baseDir, entryRelPath)

			// Create directory if it doesn't exist
			if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", outputPath, err)
			}

			if err := os.WriteFile(outputPath, data, 0644); err != nil {
				return fmt.Errorf("failed to write file %s: %w", outputPath, err)
			}
		}
	}

	return nil
}
