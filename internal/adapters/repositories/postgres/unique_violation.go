package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// uniqueViolationCode is Postgres' SQLSTATE for a unique constraint violation.
const uniqueViolationCode = "23505"

// IsUniqueViolation reports whether err, or any error it wraps, is a Postgres unique constraint violation.
func IsUniqueViolation(err error) bool {
	postgresErr := &pgconn.PgError{}

	return errors.As(err, &postgresErr) && postgresErr.Code == uniqueViolationCode
}
