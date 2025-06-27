// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package terraform

import (
	"embed"
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
