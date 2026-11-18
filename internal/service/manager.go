package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

func (s *Service) GetManagerSchedule(ctx context.Context, managerID uuid.UUID, date string) (model.ManagerSchedule, error) {
	from, to, err := DayBounds(date)
	if err != nil {
		return model.ManagerSchedule{}, err
	}

	if _, err := s.repo.GetManager(ctx, managerID); err != nil {
		return model.ManagerSchedule{}, err
	}

	meetings, err := s.repo.ListManagerMeetings(ctx, managerID, from, to)
	if err != nil {
		return model.ManagerSchedule{}, err
	}

	return model.ManagerSchedule{ManagerID: managerID, Date: date, Meetings: meetings}, nil
}
