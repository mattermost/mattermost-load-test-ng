package defaults

import (
	"encoding/json"
	"io"
)

var _ Decoder = (*JSONDecoder)(nil)

type JSONDecoder struct {
	*json.Decoder
}

func NewJSONDecoder(r io.Reader) *JSONDecoder {
	return &JSONDecoder{
		json.NewDecoder(r),
	}
}
