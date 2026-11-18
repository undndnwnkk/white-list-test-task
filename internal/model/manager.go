package model

import (
	"time"

	"github.com/google/uuid"
)

type Manager struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

type ManagerSchedule struct {
	ManagerID uuid.UUID `json:"manager_id"`
	Date      string    `json:"date"`
	Meetings  []Meeting `json:"meetings"`
}
