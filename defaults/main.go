package defaults

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ReadFrom reads a file from the given path and path and decodes it into the given value,
// based on the file extension.
func ReadFrom(path, fallbackPath string, value any) error {
	if path == "" {
		path = fallbackPath
	}

	if strings.HasSuffix(path, ".json") {
		return ReadFromJSON(path, value)
	}

	if strings.HasSuffix(path, ".toml") {
		return ReadFromTOML(path, value)
	}

	return fmt.Errorf("unsupported file format: %s", filepath.Ext(path))
}
