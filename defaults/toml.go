package defaults

import (
	"errors"
	"fmt"
	"io"

	"github.com/pelletier/go-toml/v2"
)

type TOMLDecoder struct {
	*toml.Decoder
}

// DisallowUnknownFields will disallow unknown fields in the TOML file.
// Override this method to ensure Decoder interface is met
func (d *TOMLDecoder) DisallowUnknownFields() {
	d.Decoder.DisallowUnknownFields()
}

func (d *TOMLDecoder) Decode(value interface{}) error {
	err := d.Decoder.Decode(value)
	var details *toml.StrictMissingError
	if errors.As(err, &details) {
		return fmt.Errorf("unkown configuration options in file: %w: %s", err, details.String())
	}

	return err
}

func NewTOMLDecoder(r io.Reader) Decoder {
	return &TOMLDecoder{
		toml.NewDecoder(r),
	}
}
