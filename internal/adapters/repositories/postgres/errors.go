package postgres

import "fmt"

const databaseUnreachableText = "failed connecting to the database"

// ErrDatabaseUnreachable reports a database the server could not open or reach in time.
type ErrDatabaseUnreachable struct {
	Cause error
}

func (e ErrDatabaseUnreachable) Error() string {
	return fmt.Sprintf("%s: %s", databaseUnreachableText, e.Cause)
}

func (e ErrDatabaseUnreachable) Unwrap() error {
	return e.Cause
}
