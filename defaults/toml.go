package defaults

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

func ReadFromTOML(path string, value any) error {
	if err := Set(value); err != nil {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	dec := toml.NewDecoder(file)
	dec.DisallowUnknownFields()

	err = dec.Decode(&value)
	if err != nil {
		return fmt.Errorf("could not decode file: %w", err)
	}

	return nil
}
