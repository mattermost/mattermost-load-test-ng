// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"embed"
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

// RestoreAssets writes an embedded asset to the given directory
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

				// Create parent directory if it doesn't exist
				if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
					return err
				}

				return os.WriteFile(targetPath, data, 0644)
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

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(outputPath, data, 0644)
}
