package handlers

import (
	"encoding/json"
	"testing"
)

type patchedAddress struct {
	City PatchField[string] `json:"city"`
}

type patchBody struct {
	Phone   PatchField[string]         `json:"phone"`
	Address PatchField[patchedAddress] `json:"address"`
}

func decodePatchBody(t *testing.T, raw string) (patchBody, error) {
	t.Helper()

	body := patchBody{}
	err := json.Unmarshal([]byte(raw), &body)

	return body, err
}

func TestPatchFieldTellsAbsentNullAndValueApart(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want PatchField[string]
	}{
		{name: "absent", raw: `{}`, want: PatchField[string]{}},
		{name: "null", raw: `{"phone":null}`, want: PatchField[string]{Present: true, Null: true}},
		{name: "value", raw: `{"phone":"+370 600 00000"}`, want: PatchField[string]{Present: true, Value: "+370 600 00000"}},
		{name: "empty string", raw: `{"phone":""}`, want: PatchField[string]{Present: true}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			body, err := decodePatchBody(t, testCase.raw)
			if err != nil {
				t.Fatalf("decoding %s: %v", testCase.raw, err)
			}

			if body.Phone != testCase.want {
				t.Errorf("Phone = %+v, want %+v", body.Phone, testCase.want)
			}
		})
	}
}

func TestPatchFieldDecodesNestedPatchFields(t *testing.T) {
	body, err := decodePatchBody(t, `{"address":{"city":null}}`)
	if err != nil {
		t.Fatalf("decoding: %v", err)
	}

	if !body.Address.Present || body.Address.Null {
		t.Fatalf("Address = %+v, want a present non-null object", body.Address)
	}
	if body.Address.Value.City != (PatchField[string]{Present: true, Null: true}) {
		t.Errorf("Address.City = %+v, want present and null", body.Address.Value.City)
	}
}

func TestPatchFieldRejectsAMismatchedValue(t *testing.T) {
	if _, err := decodePatchBody(t, `{"phone":42}`); err == nil {
		t.Error("decoding a number into a string field succeeded, want an error")
	}
}

func TestPatchFieldRejectsUnknownNestedFields(t *testing.T) {
	if _, err := decodePatchBody(t, `{"address":{"citty":"Vilnius"}}`); err == nil {
		t.Error("decoding an unknown nested field succeeded, want an error")
	}
}
