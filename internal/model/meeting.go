package model

import (
	"time"

	"github.com/google/uuid"
)

const MeetingDuration = 60 * time.Minute

type MeetingStatus string

const (
	MeetingStatusScheduled MeetingStatus = "scheduled"
	MeetingStatusCancelled MeetingStatus = "cancelled"
)

type Meeting struct {
	ID              uuid.UUID      `json:"id"`
	CompanyID       uuid.UUID      `json:"company_id"`
	ManagerID       uuid.UUID      `json:"manager_id"`
	StartAt         time.Time      `json:"start_at"`
	EndAt           time.Time      `json:"end_at"`
	OriginalStartAt time.Time      `json:"original_start_at"`
	Status          MeetingStatus  `json:"status"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	History         []MeetingEvent `json:"history,omitempty"`
}

type CreateMeetingRequest struct {
	CompanyID uuid.UUID `json:"company_id"`
	ManagerID uuid.UUID `json:"manager_id"`
	StartAt   time.Time `json:"start_at"`
}

type UpdateMeetingRequest struct {
	StartAt *time.Time     `json:"start_at,omitempty"`
	Status  *MeetingStatus `json:"status,omitempty"`
}

type MeetingEventType string

const (
	MeetingEventCreated     MeetingEventType = "created"
	MeetingEventRescheduled MeetingEventType = "rescheduled"
	MeetingEventCancelled   MeetingEventType = "cancelled"
)

type MeetingEvent struct {
	ID         uuid.UUID        `json:"id"`
	MeetingID  uuid.UUID        `json:"meeting_id"`
	Type       MeetingEventType `json:"type"`
	OldStartAt *time.Time       `json:"old_start_at,omitempty"`
	NewStartAt *time.Time       `json:"new_start_at,omitempty"`
	CreatedAt  time.Time        `json:"created_at"`
}
