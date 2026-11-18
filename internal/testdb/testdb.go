// Package testdb prepares an isolated PostgreSQL database for integration tests.
//
// Every test package gets its own database (<TEST_DB_DSN database>_<name>), which
// is created on demand and migrated to the latest schema. This keeps tests away
// from the application database and lets `go test ./...` run packages in parallel.
package testdb

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/undndnwnkk/white-list-test-task/migrations"
)

const (
	pgInvalidCatalogName = "3D000"
	pgDuplicateDatabase  = "42P04"
)

// Open returns a pool connected to a clean, fully migrated database dedicated
// to the calling test package. The test is skipped when TEST_DB_DSN is not set.
func Open(t testing.TB, name string) *pgxpool.Pool {
	t.Helper()

	baseDSN := os.Getenv("TEST_DB_DSN")
	if baseDSN == "" {
		t.Skip("TEST_DB_DSN is not set: integration tests need PostgreSQL (see README)")
	}

	ctx := context.Background()
	cfg, err := pgxpool.ParseConfig(baseDSN)
	if err != nil {
		t.Fatalf("parse TEST_DB_DSN: %v", err)
	}
	cfg.ConnConfig.Database = cfg.ConnConfig.Database + "_" + name

	if err := ensureDatabase(ctx, cfg.ConnConfig); err != nil {
		t.Fatalf("prepare test database %q: %v", cfg.ConnConfig.Database, err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	t.Cleanup(pool.Close)

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()
	if _, _, err := migrations.Up(ctx, db); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}

	if _, err := pool.Exec(ctx, `TRUNCATE calls, meeting_events, meetings, companies`); err != nil {
		t.Fatalf("truncate test database: %v", err)
	}
	return pool
}

// ensureDatabase creates the database from cfg if it does not exist yet,
// using the maintenance "postgres" database of the same server.
func ensureDatabase(ctx context.Context, cfg *pgx.ConnConfig) error {
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err == nil {
		return conn.Close(ctx)
	}
	if !hasCode(err, pgInvalidCatalogName) {
		return err
	}

	admin := cfg.Copy()
	admin.Database = "postgres"
	adminConn, err := pgx.ConnectConfig(ctx, admin)
	if err != nil {
		return fmt.Errorf("connect to maintenance database: %w", err)
	}
	defer adminConn.Close(ctx)

	_, err = adminConn.Exec(ctx, "CREATE DATABASE "+pgx.Identifier{cfg.Database}.Sanitize())
	if err != nil && !hasCode(err, pgDuplicateDatabase) {
		return fmt.Errorf("create database: %w", err)
	}
	return nil
}

func hasCode(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
