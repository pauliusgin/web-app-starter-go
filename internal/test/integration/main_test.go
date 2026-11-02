package integration

import (
	"context"
	"os"
	"testing"

	"github.com/pauliusgin/web_app_starter-be/internal/test/support/postgrestest"
)

// testPostgres is shared by every test in the package so the container starts once per run.
var testPostgres *postgrestest.TestPostgres

func TestMain(m *testing.M) {
	testPostgres = postgrestest.StartTestPostgres(context.Background())

	exitCode := m.Run()

	testPostgres.Terminate()
	os.Exit(exitCode)
}
