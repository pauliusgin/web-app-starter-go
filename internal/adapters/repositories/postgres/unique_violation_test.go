package postgres

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestIsUniqueViolation(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "unique violation", err: &pgconn.PgError{Code: "23505"}, want: true},
		{name: "wrapped unique violation", err: fmt.Errorf("saving: %w", &pgconn.PgError{Code: "23505"}), want: true},
		{name: "foreign key violation", err: &pgconn.PgError{Code: "23503"}, want: false},
		{name: "no rows", err: sql.ErrNoRows, want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := IsUniqueViolation(testCase.err); got != testCase.want {
				t.Errorf("IsUniqueViolation(%v) = %v, want %v", testCase.err, got, testCase.want)
			}
		})
	}
}
