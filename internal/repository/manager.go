package repository

import (
	"context"

	"github.com/google/uuid"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

func (r *Repository) GetManager(ctx context.Context, id uuid.UUID) (model.Manager, error) {
	var m model.Manager
	err := r.db.QueryRow(ctx,
		`SELECT id, name, created_at FROM managers WHERE id = $1`, id,
	).Scan(&m.ID, &m.Name, &m.CreatedAt)
	if err != nil {
		return model.Manager{}, notFound(err, model.ErrManagerNotFound)
	}
	m.CreatedAt = m.CreatedAt.UTC()
	return m, nil
}
