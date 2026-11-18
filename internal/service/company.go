package service

import (
	"context"
	"regexp"
	"strings"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

const (
	DefaultListLimit = 20
	MaxListLimit     = 100
)

var (
	binPattern   = regexp.MustCompile(`^[0-9]{12}$`)
	phonePattern = regexp.MustCompile(`^\+?[0-9][0-9\s\-()]{8,20}$`)
)

func (s *Service) CreateCompany(ctx context.Context, req model.CreateCompanyRequest) (model.Company, error) {
	req.Name = strings.TrimSpace(req.Name)
	req.BIN = strings.TrimSpace(req.BIN)
	req.Phone = strings.TrimSpace(req.Phone)

	if req.Name == "" {
		return model.Company{}, model.NewValidationError("name", "is required")
	}
	if !binPattern.MatchString(req.BIN) {
		return model.Company{}, model.NewValidationError("bin", "must consist of exactly 12 digits")
	}
	if !phonePattern.MatchString(req.Phone) {
		return model.Company{}, model.NewValidationError("phone", "must be a phone number, e.g. +77011234567")
	}

	return s.repo.CreateCompany(ctx, req)
}

func (s *Service) ListCompanies(ctx context.Context, filter model.ListCompaniesFilter) (model.CompanyList, error) {
	if filter.Status != nil && !filter.Status.Valid() {
		return model.CompanyList{}, model.NewValidationError("status", "must be one of: new, no_answer, refused, meeting_set")
	}
	if filter.Limit == 0 {
		filter.Limit = DefaultListLimit
	}
	if filter.Limit < 0 || filter.Limit > MaxListLimit {
		return model.CompanyList{}, model.NewValidationError("limit", "must be between 1 and 100")
	}
	if filter.Offset < 0 {
		return model.CompanyList{}, model.NewValidationError("offset", "must not be negative")
	}

	return s.repo.ListCompanies(ctx, filter)
}
