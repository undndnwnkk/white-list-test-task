package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

const companyColumns = `id, name, bin, phone, status, created_at`

func scanCompany(row pgx.Row) (model.Company, error) {
	var c model.Company
	if err := row.Scan(&c.ID, &c.Name, &c.BIN, &c.Phone, &c.Status, &c.CreatedAt); err != nil {
		return model.Company{}, err
	}
	c.CreatedAt = c.CreatedAt.UTC()
	return c, nil
}

func (r *Repository) CreateCompany(ctx context.Context, req model.CreateCompanyRequest) (model.Company, error) {
	c, err := scanCompany(r.db.QueryRow(ctx,
		`INSERT INTO companies (name, bin, phone) VALUES ($1, $2, $3) RETURNING `+companyColumns,
		req.Name, req.BIN, req.Phone,
	))
	if err != nil {
		return model.Company{}, mapError(err)
	}
	return c, nil
}

func (r *Repository) GetCompany(ctx context.Context, id uuid.UUID) (model.Company, error) {
	c, err := scanCompany(r.db.QueryRow(ctx,
		`SELECT `+companyColumns+` FROM companies WHERE id = $1`, id,
	))
	if err != nil {
		return model.Company{}, notFound(err, model.ErrCompanyNotFound)
	}
	return c, nil
}

func (r *Repository) LockCompany(ctx context.Context, id uuid.UUID) (model.Company, error) {
	c, err := scanCompany(r.db.QueryRow(ctx,
		`SELECT `+companyColumns+` FROM companies WHERE id = $1 FOR UPDATE`, id,
	))
	if err != nil {
		return model.Company{}, notFound(err, model.ErrCompanyNotFound)
	}
	return c, nil
}

func (r *Repository) UpdateCompanyStatus(ctx context.Context, id uuid.UUID, status model.CompanyStatus) error {
	tag, err := r.db.Exec(ctx,
		`UPDATE companies SET status = $2 WHERE id = $1`, id, string(status),
	)
	if err != nil {
		return mapError(err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrCompanyNotFound
	}
	return nil
}

// RecalculateCompanyStatus sets the company status to the result of its latest
// call that is still relevant: calls whose meeting was cancelled are ignored.
// Without such calls the company returns to "new".
func (r *Repository) RecalculateCompanyStatus(ctx context.Context, id uuid.UUID) (model.CompanyStatus, error) {
	var status model.CompanyStatus
	err := r.db.QueryRow(ctx,
		`UPDATE companies SET status = COALESCE((
			SELECT c.result FROM calls c
			LEFT JOIN meetings m ON m.id = c.meeting_id
			WHERE c.company_id = $1 AND (c.meeting_id IS NULL OR m.status = 'scheduled')
			ORDER BY c.created_at DESC, c.id DESC
			LIMIT 1
		), 'new')
		WHERE id = $1
		RETURNING status`, id,
	).Scan(&status)
	if err != nil {
		return "", notFound(err, model.ErrCompanyNotFound)
	}
	return status, nil
}

func (r *Repository) ListCompanies(ctx context.Context, filter model.ListCompaniesFilter) (model.CompanyList, error) {
	var status *string
	if filter.Status != nil {
		s := string(*filter.Status)
		status = &s
	}

	result := model.CompanyList{Items: []model.Company{}, Limit: filter.Limit, Offset: filter.Offset}

	err := r.db.QueryRow(ctx,
		`SELECT count(*) FROM companies WHERE ($1::text IS NULL OR status = $1)`, status,
	).Scan(&result.Total)
	if err != nil {
		return model.CompanyList{}, fmt.Errorf("count companies: %w", err)
	}

	rows, err := r.db.Query(ctx,
		`SELECT `+companyColumns+` FROM companies
		 WHERE ($1::text IS NULL OR status = $1)
		 ORDER BY created_at DESC, id
		 LIMIT $2 OFFSET $3`,
		status, filter.Limit, filter.Offset,
	)
	if err != nil {
		return model.CompanyList{}, fmt.Errorf("list companies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		c, err := scanCompany(rows)
		if err != nil {
			return model.CompanyList{}, fmt.Errorf("scan company: %w", err)
		}
		result.Items = append(result.Items, c)
	}
	if err := rows.Err(); err != nil {
		return model.CompanyList{}, fmt.Errorf("iterate companies: %w", err)
	}
	return result, nil
}
