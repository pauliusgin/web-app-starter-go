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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsUniqueViolation(tc.err); got != tc.want {
				t.Errorf("IsUniqueViolation(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
