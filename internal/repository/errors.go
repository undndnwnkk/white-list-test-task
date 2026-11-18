package repository

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgCheckViolation      = "23514"
	pgExclusionViolation  = "23P01"
)

var constraintErrors = map[string]error{
	"companies_bin_key":              model.ErrBINAlreadyExists,
	"meetings_manager_no_overlap":    model.ErrManagerBusy,
	"meetings_company_no_overlap":    model.ErrCompanyBusy,
	"meetings_company_id_fkey":       model.ErrCompanyNotFound,
	"meetings_manager_id_fkey":       model.ErrManagerNotFound,
	"calls_company_id_fkey":          model.ErrCompanyNotFound,
	"calls_manager_id_fkey":          model.ErrManagerNotFound,
	"calls_meeting_id_fkey":          model.ErrMeetingNotFound,
	"calls_meeting_id_key":           model.ErrMeetingAlreadyLinked,
	"meeting_events_meeting_id_fkey": model.ErrMeetingNotFound,
}

func mapError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return err
	}

	if mapped, ok := constraintErrors[pgErr.ConstraintName]; ok {
		return mapped
	}

	switch pgErr.Code {
	case pgCheckViolation, pgUniqueViolation, pgForeignKeyViolation, pgExclusionViolation:
		return errors.Join(model.ErrInvalidData, err)
	}
	return err
}

func notFound(err error, target error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return target
	}
	return mapError(err)
}
