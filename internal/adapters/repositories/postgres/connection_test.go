package postgres

import (
	"context"
	"errors"
	"testing"
)

const unroutableDatabaseURL = "postgres://user:password@[100::1]:5432/web_app_starter?sslmode=disable"

const malformedDatabaseURL = "postgres://user:password@host:not-a-port/web_app_starter"

func TestNewDatabaseConnectionRejectsAMalformedURL(t *testing.T) {
	db, err := NewDatabaseConnection(context.Background(), malformedDatabaseURL)
	if db != nil {
		t.Fatalf("db = %v, want nil", db)
	}

	var unreachable ErrDatabaseUnreachable
	if !errors.As(err, &unreachable) {
		t.Fatalf("err = %v, want ErrDatabaseUnreachable", err)
	}

	if unreachable.Unwrap() == nil {
		t.Fatal("unwrapped cause is nil, want the underlying driver error")
	}
}

func TestNewDatabaseConnectionReportsAnUnreachableDatabase(t *testing.T) {
	db, err := NewDatabaseConnection(context.Background(), unroutableDatabaseURL)
	if db != nil {
		t.Fatalf("db = %v, want nil", db)
	}

	var unreachable ErrDatabaseUnreachable
	if !errors.As(err, &unreachable) {
		t.Fatalf("err = %v, want ErrDatabaseUnreachable", err)
	}
}

func TestNewDatabaseConnectionHonoursACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	db, err := NewDatabaseConnection(ctx, unroutableDatabaseURL)
	if db != nil {
		t.Fatalf("db = %v, want nil", db)
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want it to wrap context.Canceled", err)
	}
}
