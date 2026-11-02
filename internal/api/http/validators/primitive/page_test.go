package primitive

import (
	"errors"
	"testing"
)

func TestParsePageNumber(t *testing.T) {
	cases := []struct {
		name      string
		raw       string
		want      int
		wantError bool
	}{
		{name: "absent defaults", raw: "", want: DefaultPageNumber},
		{name: "first page", raw: "1", want: 1},
		{name: "last allowed page", raw: "1000", want: MaxPageNumber},
		{name: "zero", raw: "0", wantError: true},
		{name: "negative", raw: "-2", wantError: true},
		{name: "past the maximum", raw: "1001", wantError: true},
		{name: "not a number", raw: "two", wantError: true},
		{name: "fraction", raw: "1.5", wantError: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			page, err := ParsePageNumber(testCase.raw)

			if testCase.wantError {
				if _, ok := errors.AsType[ErrPageNumberInvalid](err); !ok {
					t.Errorf("err = %v, want ErrPageNumberInvalid", err)
				}
				return
			}
			if err != nil || page != testCase.want {
				t.Errorf("ParsePageNumber(%q) = %d, %v, want %d", testCase.raw, page, err, testCase.want)
			}
		})
	}
}

func TestParsePageLimit(t *testing.T) {
	cases := []struct {
		name      string
		raw       string
		want      int
		wantError bool
	}{
		{name: "absent defaults", raw: "", want: DefaultPageLimit},
		{name: "one", raw: "1", want: 1},
		{name: "maximum", raw: "100", want: MaxPageLimit},
		{name: "zero", raw: "0", wantError: true},
		{name: "past the maximum", raw: "101", wantError: true},
		{name: "not a number", raw: "all", wantError: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			limit, err := ParsePageLimit(testCase.raw)

			if testCase.wantError {
				if _, ok := errors.AsType[ErrPageLimitInvalid](err); !ok {
					t.Errorf("err = %v, want ErrPageLimitInvalid", err)
				}
				return
			}
			if err != nil || limit != testCase.want {
				t.Errorf("ParsePageLimit(%q) = %d, %v, want %d", testCase.raw, limit, err, testCase.want)
			}
		})
	}
}
