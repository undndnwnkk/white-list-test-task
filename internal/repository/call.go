package repository

import (
	"context"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

func (r *Repository) CreateCall(ctx context.Context, req model.CreateCallRequest) (model.Call, error) {
	var c model.Call
	err := r.db.QueryRow(ctx,
		`INSERT INTO calls (company_id, manager_id, result, meeting_id)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, company_id, manager_id, result, meeting_id, created_at`,
		req.CompanyID, req.ManagerID, string(req.Result), req.MeetingID,
	).Scan(&c.ID, &c.CompanyID, &c.ManagerID, &c.Result, &c.MeetingID, &c.CreatedAt)
	if err != nil {
		return model.Call{}, mapError(err)
	}
	c.CreatedAt = c.CreatedAt.UTC()
	return c, nil
}
