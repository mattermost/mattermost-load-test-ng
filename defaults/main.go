package defaults

import (
	"fmt"
	"os"
	"strings"
)

type Decoder interface {
	Decode(value any) error
	DisallowUnknownFields()
}

// ReadFrom reads a configuration file to the value.
// This function will try to read from given path, if it is empty will try
// fallback path. If it fails on fallback, it will set value to its defaults
func ReadFrom(path, fallbackPath string, value any) error {
	if err := Set(value); err != nil {
		return err
	}

	if path != "" {
		if err := read(path, value); err != nil {
			return fmt.Errorf("failed to read from path %s: %w", path, err)
		}
		return nil
	}

	// If the fallback path doesn't exist, return.
	if _, err := os.Stat(fallbackPath); err != nil && os.IsNotExist(err) {
		return nil
	}

	if err := read(fallbackPath, value); err != nil {
		return fmt.Errorf("failed to read from fallback path %s: %w", fallbackPath, err)
	}

	return nil
}

// read reads a file to the value using the provided decoder
func read(path string, value any) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	var dec Decoder
	if strings.Contains(path, ".toml") {
		dec = NewTOMLDecoder(file)
	} else {
		dec = NewJSONDecoder(file)
	}

	dec.DisallowUnknownFields()
	err = dec.Decode(value)
	if err != nil {
		return fmt.Errorf("could not decode file: %w", err)
	}

	return nil
}
