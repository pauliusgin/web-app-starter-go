// Package postgrestest runs a throwaway Postgres container and hands each test its own migrated database.
package postgrestest

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/pauliusgin/web_app_starter-be/internal/adapters/repositories/postgres"
	"github.com/pauliusgin/web_app_starter-be/sql/schema"
)

// postgresImage matches the image `make db-build` runs, so tests see the production engine version.
const postgresImage = "postgres:18"

const templateDatabaseName = "web_app_starter_template"

const adminDatabaseName = "postgres"

const containerStartTimeout = 3 * time.Minute

const databaseStatementTimeout = 30 * time.Second

// IntegrationRequiredEnvVar, when set to "1", turns a missing Docker into a test failure instead of a skip.
const IntegrationRequiredEnvVar = "INTEGRATION_REQUIRED"

// TestPostgres is a running Postgres server holding a migrated template every test database is cloned from.
type TestPostgres struct {
	container        *tcpostgres.PostgresContainer
	adminDatabase    *sql.DB
	connectionString *url.URL
	startFailure     error

	// cloneLock serializes CREATE DATABASE ... TEMPLATE, which Postgres refuses while another
	// session (including a concurrent clone) is reading the template.
	cloneLock      sync.Mutex
	clonedDatabase int
}

// StartTestPostgres starts the container and migrates the template; a failure is kept for NewTestDatabase.
func StartTestPostgres(ctx context.Context) *TestPostgres {
	startCtx, cancelStart := context.WithTimeout(ctx, containerStartTimeout)
	defer cancelStart()

	testPostgres := &TestPostgres{}

	container, err := tcpostgres.Run(startCtx, postgresImage,
		tcpostgres.WithDatabase(templateDatabaseName),
		tcpostgres.BasicWaitStrategies(),
	)
	// Run can hand back a container alongside its error; keeping it lets Terminate remove it.
	testPostgres.container = container
	if err != nil {
		testPostgres.startFailure = ErrTestPostgresUnavailable{Cause: err}
		return testPostgres
	}

	if err := testPostgres.prepare(startCtx); err != nil {
		testPostgres.startFailure = err
	}

	return testPostgres
}

func (p *TestPostgres) prepare(ctx context.Context) error {
	rawConnectionString, err := p.container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return ErrTestPostgresUnavailable{Cause: err}
	}
	connectionString, err := url.Parse(rawConnectionString)
	if err != nil {
		return ErrTestPostgresUnavailable{Cause: err}
	}
	p.connectionString = connectionString

	if err := p.migrateTemplate(ctx); err != nil {
		return err
	}

	adminDatabase, err := postgres.NewDatabaseConnection(ctx, p.connectionStringFor(adminDatabaseName))
	if err != nil {
		return ErrTestPostgresUnavailable{Cause: err}
	}
	p.adminDatabase = adminDatabase

	return nil
}

// migrateTemplate closes its connection before returning: a template with an open session cannot be cloned.
func (p *TestPostgres) migrateTemplate(ctx context.Context) error {
	templateDatabase, err := postgres.NewDatabaseConnection(ctx, p.connectionStringFor(templateDatabaseName))
	if err != nil {
		return ErrTemplateMigrationFailed{Cause: err}
	}
	defer templateDatabase.Close()

	migrations, err := goose.NewProvider(goose.DialectPostgres, templateDatabase, schema.Migrations)
	if err != nil {
		return ErrTemplateMigrationFailed{Cause: err}
	}
	if _, err := migrations.Up(ctx); err != nil {
		return ErrTemplateMigrationFailed{Cause: err}
	}

	return nil
}

func (p *TestPostgres) connectionStringFor(databaseName string) string {
	withDatabase := *p.connectionString
	withDatabase.Path = "/" + databaseName

	return withDatabase.String()
}

// NewTestDatabase clones the migrated template into a database only this test sees, dropped on cleanup.
func (p *TestPostgres) NewTestDatabase(t testing.TB) *sql.DB {
	t.Helper()

	if p.startFailure != nil {
		skipOrFailWithoutPostgres(t, p.startFailure)
	}

	databaseName := p.cloneTemplate(t)

	ctx, cancel := context.WithTimeout(context.Background(), databaseStatementTimeout)
	defer cancel()

	testDatabase, err := postgres.NewDatabaseConnection(ctx, p.connectionStringFor(databaseName))
	if err != nil {
		t.Fatal(ErrTestDatabaseCreationFailed{DatabaseName: databaseName, Cause: err})
	}

	t.Cleanup(func() {
		testDatabase.Close()
		p.dropDatabase(t, databaseName)
	})

	return testDatabase
}

func (p *TestPostgres) cloneTemplate(t testing.TB) string {
	t.Helper()

	p.cloneLock.Lock()
	defer p.cloneLock.Unlock()

	p.clonedDatabase++
	databaseName := fmt.Sprintf("test_%d", p.clonedDatabase)

	ctx, cancel := context.WithTimeout(context.Background(), databaseStatementTimeout)
	defer cancel()

	statement := fmt.Sprintf("CREATE DATABASE %s TEMPLATE %s", databaseName, templateDatabaseName)
	if _, err := p.adminDatabase.ExecContext(ctx, statement); err != nil {
		t.Fatal(ErrTestDatabaseCreationFailed{DatabaseName: databaseName, Cause: err})
	}

	return databaseName
}

func (p *TestPostgres) dropDatabase(t testing.TB, databaseName string) {
	ctx, cancel := context.WithTimeout(context.Background(), databaseStatementTimeout)
	defer cancel()

	statement := fmt.Sprintf("DROP DATABASE IF EXISTS %s WITH (FORCE)", databaseName)
	if _, err := p.adminDatabase.ExecContext(ctx, statement); err != nil {
		t.Errorf("dropping test database %q: %v", databaseName, err)
	}
}

// Terminate closes the admin connection and removes the container; call it once after m.Run.
func (p *TestPostgres) Terminate() {
	if p.adminDatabase != nil {
		p.adminDatabase.Close()
	}
	if p.container == nil {
		return
	}
	if err := testcontainers.TerminateContainer(p.container); err != nil {
		slog.Error("failed terminating the test postgres container", "error", err)
	}
}

func skipOrFailWithoutPostgres(t testing.TB, startFailure error) {
	t.Helper()

	if os.Getenv(IntegrationRequiredEnvVar) == "1" {
		t.Fatalf("%v (%s=1 requires Docker)", startFailure, IntegrationRequiredEnvVar)
	}

	t.Skipf("postgres integration tests need Docker: %v", startFailure)
}
