// Package postgres opens the database handle every Postgres repository in this tree reads through.
package postgres

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const databaseDriverName = "pgx"

const databasePingTimeout = 5 * time.Second

const (
	maxOpenConnections    = 25
	maxIdleConnections    = 25
	connectionMaxLifetime = 30 * time.Minute
	connectionMaxIdleTime = 5 * time.Minute
)

// NewDatabaseConnection opens a pooled handle and proves it reachable before returning it.
func NewDatabaseConnection(ctx context.Context, databaseURL string) (*sql.DB, error) {
	db, err := sql.Open(databaseDriverName, databaseURL)
	if err != nil {
		return nil, ErrDatabaseUnreachable{Cause: err}
	}

	db.SetMaxOpenConns(maxOpenConnections)
	db.SetMaxIdleConns(maxIdleConnections)
	db.SetConnMaxLifetime(connectionMaxLifetime)
	db.SetConnMaxIdleTime(connectionMaxIdleTime)

	pingCtx, cancelPing := context.WithTimeout(ctx, databasePingTimeout)
	defer cancelPing()

	if err := db.PingContext(pingCtx); err != nil {
		db.Close()
		return nil, ErrDatabaseUnreachable{Cause: err}
	}

	return db, nil
}
