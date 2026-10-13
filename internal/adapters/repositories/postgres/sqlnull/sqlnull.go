// Package sqlnull converts between sqlc's sql.Null* column values and the shapes mappers hand on.
package sqlnull

import (
	"database/sql"
	"time"
)

// StringPointer answers nil for a NULL column, otherwise a pointer to its text.
func StringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	return &value.String
}

// TimePointer answers nil for a NULL column, otherwise a pointer to its time.
func TimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}

	return &value.Time
}

// StringUnlessEmpty stores an empty domain string as NULL, any other as itself.
func StringUnlessEmpty(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}

// Int32UnlessZero stores a zero domain int as NULL, any other as itself.
func Int32UnlessZero(value int) sql.NullInt32 {
	return sql.NullInt32{Int32: int32(value), Valid: value != 0}
}
