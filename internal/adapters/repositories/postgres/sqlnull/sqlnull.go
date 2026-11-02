// Package sqlnull converts between the sql.Null* column values sqlc generates and plain Go values.
package sqlnull

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// StringPointer answers nil for a NULL column, else a pointer to its text.
func StringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

// TimePointer answers nil for a NULL column, else a pointer to its time.
func TimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}

	return &value.Time
}

// UUIDPointer answers nil for a NULL column, else a pointer to its UUID.
func UUIDPointer(value uuid.NullUUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}

	return &value.UUID
}

// StringUnlessEmpty stores an empty domain string as NULL.
func StringUnlessEmpty(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

// Int32UnlessZero stores a zero domain number as NULL.
func Int32UnlessZero(value int) sql.NullInt32 {
	return sql.NullInt32{Int32: int32(value), Valid: value != 0}
}

// UUIDUnlessNil stores a nil domain UUID as NULL.
func UUIDUnlessNil(value uuid.UUID) uuid.NullUUID {
	return uuid.NullUUID{UUID: value, Valid: value != uuid.Nil}
}
