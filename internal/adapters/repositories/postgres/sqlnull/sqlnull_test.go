package sqlnull

import (
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

var storedAt = time.Date(2026, time.March, 4, 10, 30, 0, 0, time.UTC)

var storedID = uuid.MustParse("6f1c2d3e-4b5a-4c6d-8e7f-9a0b1c2d3e4f")

func TestStringPointer(t *testing.T) {
	ada := "Ada"
	empty := ""
	cases := []struct {
		name   string
		stored sql.NullString
		want   *string
	}{
		{name: "null answers nil", stored: sql.NullString{}},
		{name: "text is pointed to", stored: sql.NullString{String: ada, Valid: true}, want: &ada},
		{name: "stored empty string stays non-nil", stored: sql.NullString{Valid: true}, want: &empty},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := StringPointer(testCase.stored)

			if (got == nil) != (testCase.want == nil) || (got != nil && *got != *testCase.want) {
				t.Errorf("StringPointer(%+v) = %v, want %v", testCase.stored, got, testCase.want)
			}
		})
	}
}

func TestTimePointer(t *testing.T) {
	cases := []struct {
		name   string
		stored sql.NullTime
		want   *time.Time
	}{
		{name: "null answers nil", stored: sql.NullTime{}},
		{name: "time is pointed to", stored: sql.NullTime{Time: storedAt, Valid: true}, want: &storedAt},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := TimePointer(testCase.stored)

			if (got == nil) != (testCase.want == nil) || (got != nil && !got.Equal(*testCase.want)) {
				t.Errorf("TimePointer(%+v) = %v, want %v", testCase.stored, got, testCase.want)
			}
		})
	}
}

func TestUUIDPointer(t *testing.T) {
	cases := []struct {
		name   string
		stored uuid.NullUUID
		want   *uuid.UUID
	}{
		{name: "null answers nil", stored: uuid.NullUUID{}},
		{name: "id is pointed to", stored: uuid.NullUUID{UUID: storedID, Valid: true}, want: &storedID},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			got := UUIDPointer(testCase.stored)

			if (got == nil) != (testCase.want == nil) || (got != nil && *got != *testCase.want) {
				t.Errorf("UUIDPointer(%+v) = %v, want %v", testCase.stored, got, testCase.want)
			}
		})
	}
}

func TestStringUnlessEmpty(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  sql.NullString
	}{
		{name: "empty stores null", value: ""},
		{name: "text stores text", value: "Ada", want: sql.NullString{String: "Ada", Valid: true}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := StringUnlessEmpty(testCase.value); got != testCase.want {
				t.Errorf("StringUnlessEmpty(%q) = %+v, want %+v", testCase.value, got, testCase.want)
			}
		})
	}
}

func TestInt32UnlessZero(t *testing.T) {
	cases := []struct {
		name  string
		value int
		want  sql.NullInt32
	}{
		{name: "zero stores null", value: 0},
		{name: "number stores number", value: 800, want: sql.NullInt32{Int32: 800, Valid: true}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := Int32UnlessZero(testCase.value); got != testCase.want {
				t.Errorf("Int32UnlessZero(%d) = %+v, want %+v", testCase.value, got, testCase.want)
			}
		})
	}
}

func TestUUIDUnlessNil(t *testing.T) {
	cases := []struct {
		name  string
		value uuid.UUID
		want  uuid.NullUUID
	}{
		{name: "nil id stores null", value: uuid.Nil},
		{name: "id stores id", value: storedID, want: uuid.NullUUID{UUID: storedID, Valid: true}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := UUIDUnlessNil(testCase.value); got != testCase.want {
				t.Errorf("UUIDUnlessNil(%s) = %+v, want %+v", testCase.value, got, testCase.want)
			}
		})
	}
}
