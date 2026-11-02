package postgrestest

import (
	"errors"
	"testing"
)

func TestNewTestDatabaseSkipsWhenPostgresNeverStarted(t *testing.T) {
	t.Setenv(IntegrationRequiredEnvVar, "")
	unavailable := &TestPostgres{startFailure: ErrTestPostgresUnavailable{Cause: errors.New("docker socket not found")}}

	var skipped bool
	t.Run("integration test", func(t *testing.T) {
		defer func() { skipped = t.Skipped() }()

		unavailable.NewTestDatabase(t)
	})

	if !skipped {
		t.Error("the integration test ran on, want it skipped without Docker")
	}
}
