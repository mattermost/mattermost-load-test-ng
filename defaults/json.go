package defaults

import (
	"encoding/json"
	"fmt"
	"os"
)

// ReadFromJSON reads a json file to the value.
// This function will try to read from given path, if it is empty will try
// fallback path. If it fails on fallback, it will set value to it's defaults
func ReadFromJSON(path, fallbackPath string, value interface{}) error {
	if err := Set(value); err != nil {
		return err
	}

	if path != "" {
		if err := readJSON(path, &value); err != nil {
			return fmt.Errorf("failed to read from path %s: %w", path, err)
		}
		return nil
	}

	// If the fallback path doesn't exist, return.
	if _, err := os.Stat(fallbackPath); err != nil && os.IsNotExist(err) {
		return nil
	}

	if err := readJSON(fallbackPath, value); err != nil {
		return fmt.Errorf("failed to read from path %s: %w", fallbackPath, err)
	}

	return nil

}

func readJSON(path string, value interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	dec.DisallowUnknownFields()
	err = dec.Decode(&value)
	if err != nil {
		return fmt.Errorf("could not decode file: %w", err)
	}

	return nil
}
