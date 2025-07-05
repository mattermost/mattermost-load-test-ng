// Package testdata provides embedded test data files
package testdata

import (
	"embed"
)

//go:embed *
var TestDataFS embed.FS

// MustAsset returns the content of the embedded file and panics on error
func MustAsset(name string) []byte {
	data, err := TestDataFS.ReadFile(name)
	if err != nil {
		panic("testdata: Asset(" + name + "): " + err.Error())
	}
	return data
}

// MustAssetString returns the content of the embedded file as string and panics on error
func MustAssetString(name string) string {
	return string(MustAsset(name))
}