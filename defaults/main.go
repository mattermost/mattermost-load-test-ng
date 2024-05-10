package defaults

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Decoder interface {
	Decode(value any) error
	DisallowUnknownFields()
}

type DecoderFactory func(r io.Reader) Decoder

// ReadFrom reads a file from the given path and path and decodes it into the given value,
// based on the file extension.
func ReadFrom(path, fallbackPath string, value any) error {
	if strings.HasSuffix(path, ".json") {
		return readFromFile(NewJSONDecoder, path, fallbackPath, value)
	}

	if strings.HasSuffix(path, ".toml") {
		return readFromFile(NewTOMLDecoder, path, fallbackPath, value)
	}

	return fmt.Errorf("unsupported file format: %s", filepath.Ext(path))
}

// readFromFile reads a configuration file to the value.
// This function will try to read from given path, if it is empty will try
// fallback path. If it fails on fallback, it will set value to it's defaults
func readFromFile(dFactory DecoderFactory, path, fallbackPath string, value any) error {
	if err := Set(value); err != nil {
		return err
	}

	if path != "" {
		if err := read(dFactory, path, value); err != nil {
			return fmt.Errorf("failed to read from path %s: %w", path, err)
		}
		return nil
	}

	// If the fallback path doesn't exist, return.
	if _, err := os.Stat(fallbackPath); err != nil && os.IsNotExist(err) {
		return nil
	}

	if err := read(dFactory, fallbackPath, value); err != nil {
		return fmt.Errorf("failed to read from fallback path %s: %w", fallbackPath, err)
	}

	return nil
}

// read reads a file to the value using the provided decoder
func read(decoderFactory DecoderFactory, path string, value any) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	dec := decoderFactory(file)
	dec.DisallowUnknownFields()
	err = dec.Decode(value)
	if err != nil {
		return fmt.Errorf("could not decode file: %w", err)
	}

	return nil
}
