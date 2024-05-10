package defaults

import (
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

func NewTOMLDecoder(r io.Reader) Decoder {
	return &TOMLDecoder{
		toml.NewDecoder(r),
	}
}
