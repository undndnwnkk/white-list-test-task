package model

import (
	"time"

	"github.com/google/uuid"
)

type CompanyStatus string

const (
	CompanyStatusNew        CompanyStatus = "new"
	CompanyStatusNoAnswer   CompanyStatus = "no_answer"
	CompanyStatusRefused    CompanyStatus = "refused"
	CompanyStatusMeetingSet CompanyStatus = "meeting_set"
)

func (s CompanyStatus) Valid() bool {
	switch s {
	case CompanyStatusNew, CompanyStatusNoAnswer, CompanyStatusRefused, CompanyStatusMeetingSet:
		return true
	}
	return false
}

type Company struct {
	ID        uuid.UUID     `json:"id"`
	Name      string        `json:"name"`
	BIN       string        `json:"bin"`
	Phone     string        `json:"phone"`
	Status    CompanyStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
}

type CreateCompanyRequest struct {
	Name  string `json:"name"`
	BIN   string `json:"bin"`
	Phone string `json:"phone"`
}

type ListCompaniesFilter struct {
	Status *CompanyStatus
	Limit  int
	Offset int
}

type CompanyList struct {
	Items  []Company `json:"items"`
	Total  int       `json:"total"`
	Limit  int       `json:"limit"`
	Offset int       `json:"offset"`
}
