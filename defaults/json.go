package defaults

import (
	"encoding/json"
	"io"
)

type JSONDecoder struct {
	*json.Decoder
}

func NewJSONDecoder(r io.Reader) Decoder {
	return &JSONDecoder{
		json.NewDecoder(r),
	}
}
