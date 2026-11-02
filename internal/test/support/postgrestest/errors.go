package postgrestest

import "fmt"

const testPostgresUnavailableText = "failed starting the test postgres container"

const templateMigrationFailedText = "failed migrating the template database"

const testDatabaseCreationFailedText = "failed creating the test database"

// ErrTestPostgresUnavailable reports a Postgres container that could not be started or reached.
type ErrTestPostgresUnavailable struct {
	Cause error
}

func (e ErrTestPostgresUnavailable) Error() string {
	return fmt.Sprintf("%s: %s", testPostgresUnavailableText, e.Cause)
}

func (e ErrTestPostgresUnavailable) Unwrap() error {
	return e.Cause
}

// ErrTemplateMigrationFailed reports migrations that could not be applied to the template database.
type ErrTemplateMigrationFailed struct {
	Cause error
}

func (e ErrTemplateMigrationFailed) Error() string {
	return fmt.Sprintf("%s: %s", templateMigrationFailedText, e.Cause)
}

func (e ErrTemplateMigrationFailed) Unwrap() error {
	return e.Cause
}

// ErrTestDatabaseCreationFailed reports a per-test database that could not be cloned or opened.
type ErrTestDatabaseCreationFailed struct {
	DatabaseName string
	Cause        error
}

func (e ErrTestDatabaseCreationFailed) Error() string {
	return fmt.Sprintf("%s %q: %s", testDatabaseCreationFailedText, e.DatabaseName, e.Cause)
}

func (e ErrTestDatabaseCreationFailed) Unwrap() error {
	return e.Cause
}
