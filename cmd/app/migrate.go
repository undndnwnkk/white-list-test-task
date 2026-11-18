package main

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/undndnwnkk/white-list-test-task/migrations"
)

func applyMigrations(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) error {
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	results, version, err := migrations.Up(ctx, db)
	if err != nil {
		return err
	}

	for _, r := range results {
		log.Info("migration applied", "source", r.Source.Path, "duration", r.Duration)
	}
	log.Info("database schema is up to date", "version", version)
	return nil
}
