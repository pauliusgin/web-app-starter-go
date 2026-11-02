package handlers

import (
	"bytes"
	"encoding/json"
)

var jsonNull = []byte("null")

// PatchField is a PATCH body field that tells an absent key, an explicit null and a value apart.
type PatchField[T any] struct {
	Present bool
	Null    bool
	Value   T
}

// UnmarshalJSON marks the key present and decodes a non-null value, rejecting unknown nested fields.
// encoding/json calls it only for keys the body holds, which is what leaves Present false otherwise.
func (f *PatchField[T]) UnmarshalJSON(data []byte) error {
	f.Present = true
	if bytes.Equal(data, jsonNull) {
		f.Null = true
		return nil
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	return decoder.Decode(&f.Value)
}
