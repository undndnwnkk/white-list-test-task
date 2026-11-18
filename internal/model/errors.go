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

	ErrValidation              = errors.New("validation failed")
	ErrOutsideWorkingHours     = errors.New("meeting must fit within working hours: Mon-Fri 09:00-18:00 Asia/Almaty")
	ErrMeetingInPast           = errors.New("meeting cannot start in the past")
	ErrMeetingCancelled        = errors.New("meeting is cancelled")
	ErrMeetingAlreadyStarted   = errors.New("meeting has already started")
	ErrCompanyHasActiveMeeting = errors.New("company already has an active meeting")
	ErrMeetingCompanyMismatch  = errors.New("meeting belongs to another company")
)

type ValidationError struct {
	Field   string
	Message string
}

func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{Field: field, Message: message}
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

func (e *ValidationError) Is(target error) bool {
	return target == ErrValidation
}
