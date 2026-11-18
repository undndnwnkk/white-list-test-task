package model

import (
	"time"

	"github.com/google/uuid"
)

type CallResult string

const (
	CallResultNoAnswer   CallResult = "no_answer"
	CallResultRefused    CallResult = "refused"
	CallResultMeetingSet CallResult = "meeting_set"
)

type Call struct {
	ID        uuid.UUID  `json:"id"`
	CompanyID uuid.UUID  `json:"company_id"`
	ManagerID uuid.UUID  `json:"manager_id"`
	Result    CallResult `json:"result"`
	MeetingID *uuid.UUID `json:"meeting_id,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

type CreateCallRequest struct {
	CompanyID uuid.UUID  `json:"company_id"`
	ManagerID uuid.UUID  `json:"manager_id"`
	Result    CallResult `json:"result"`
	MeetingID *uuid.UUID `json:"meeting_id,omitempty"`
}
