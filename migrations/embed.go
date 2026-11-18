package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var FS embed.FS

// Up applies all pending migrations and returns the applied ones together
// with the resulting schema version.
func Up(ctx context.Context, db *sql.DB) ([]*goose.MigrationResult, int64, error) {
	provider, err := goose.NewProvider(goose.DialectPostgres, db, FS)
	if err != nil {
		return nil, 0, fmt.Errorf("create migration provider: %w", err)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("apply migrations: %w", err)
	}

	version, err := provider.GetDBVersion(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("get schema version: %w", err)
	}
	return results, version, nil
}
