package model

import "errors"

var (
	ErrCompanyNotFound  = errors.New("company not found")
	ErrManagerNotFound  = errors.New("manager not found")
	ErrMeetingNotFound  = errors.New("meeting not found")
	ErrBINAlreadyExists = errors.New("company with this BIN already exists")
	ErrManagerBusy      = errors.New("manager already has a meeting at this time")
	ErrCompanyBusy      = errors.New("company already has a meeting at this time")
	ErrInvalidData      = errors.New("data violates database constraints")
)
