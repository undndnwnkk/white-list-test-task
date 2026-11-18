package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
	"github.com/undndnwnkk/white-list-test-task/internal/repository"
)

func (s *Service) CreateMeeting(ctx context.Context, req model.CreateMeetingRequest) (model.Meeting, error) {
	if req.CompanyID == uuid.Nil {
		return model.Meeting{}, model.NewValidationError("company_id", "is required")
	}
	if req.ManagerID == uuid.Nil {
		return model.Meeting{}, model.NewValidationError("manager_id", "is required")
	}

	now := s.now()
	start := req.StartAt.UTC()
	if err := validateNewMeetingStart(start, now); err != nil {
		return model.Meeting{}, err
	}

	var result model.Meeting
	err := s.repo.WithTx(ctx, func(repo *repository.Repository) error {
		if _, err := repo.LockCompany(ctx, req.CompanyID); err != nil {
			return err
		}
		if _, err := repo.GetManager(ctx, req.ManagerID); err != nil {
			return err
		}

		hasActive, err := repo.HasActiveCompanyMeeting(ctx, req.CompanyID, now, uuid.Nil)
		if err != nil {
			return err
		}
		if hasActive {
			return model.ErrCompanyHasActiveMeeting
		}

		meeting, err := repo.CreateMeeting(ctx, req.CompanyID, req.ManagerID, start)
		if err != nil {
			return err
		}

		event, err := repo.AddMeetingEvent(ctx, meeting.ID, model.MeetingEventCreated, nil, &meeting.StartAt)
		if err != nil {
			return err
		}

		meeting.History = []model.MeetingEvent{event}
		result = meeting
		return nil
	})
	return result, err
}

func (s *Service) GetMeeting(ctx context.Context, id uuid.UUID) (model.Meeting, error) {
	meeting, err := s.repo.GetMeeting(ctx, id)
	if err != nil {
		return model.Meeting{}, err
	}

	meeting.History, err = s.repo.ListMeetingEvents(ctx, id)
	if err != nil {
		return model.Meeting{}, err
	}
	return meeting, nil
}

func (s *Service) UpdateMeeting(ctx context.Context, id uuid.UUID, req model.UpdateMeetingRequest) (model.Meeting, error) {
	switch {
	case req.StartAt != nil && req.Status != nil:
		return model.Meeting{}, model.NewValidationError("body", "provide either start_at to reschedule or status to cancel, not both")
	case req.StartAt != nil:
		return s.rescheduleMeeting(ctx, id, req.StartAt.UTC())
	case req.Status != nil:
		if *req.Status != model.MeetingStatusCancelled {
			return model.Meeting{}, model.NewValidationError("status", "only \"cancelled\" is allowed")
		}
		return s.cancelMeeting(ctx, id)
	default:
		return model.Meeting{}, model.NewValidationError("body", "start_at or status is required")
	}
}

func (s *Service) rescheduleMeeting(ctx context.Context, id uuid.UUID, newStart time.Time) (model.Meeting, error) {
	now := s.now()
	if err := validateNewMeetingStart(newStart, now); err != nil {
		return model.Meeting{}, err
	}

	var result model.Meeting
	err := s.repo.WithTx(ctx, func(repo *repository.Repository) error {
		meeting, err := repo.LockMeeting(ctx, id)
		if err != nil {
			return err
		}
		if err := ensureMeetingChangeable(meeting, now); err != nil {
			return err
		}
		if meeting.StartAt.Equal(newStart) {
			return model.NewValidationError("start_at", "meeting is already scheduled at this time")
		}

		oldStart := meeting.StartAt
		meeting, err = repo.RescheduleMeeting(ctx, id, newStart)
		if err != nil {
			return err
		}
		if _, err := repo.AddMeetingEvent(ctx, id, model.MeetingEventRescheduled, &oldStart, &meeting.StartAt); err != nil {
			return err
		}

		meeting.History, err = repo.ListMeetingEvents(ctx, id)
		result = meeting
		return err
	})
	return result, err
}

func (s *Service) cancelMeeting(ctx context.Context, id uuid.UUID) (model.Meeting, error) {
	now := s.now()

	var result model.Meeting
	err := s.repo.WithTx(ctx, func(repo *repository.Repository) error {
		meeting, err := repo.LockMeeting(ctx, id)
		if err != nil {
			return err
		}
		if err := ensureMeetingChangeable(meeting, now); err != nil {
			return err
		}

		// Lock order meeting -> company is safe: flows that lock the company
		// first (CreateCall, CreateMeeting) never wait for a meeting row lock.
		if _, err := repo.LockCompany(ctx, meeting.CompanyID); err != nil {
			return err
		}

		meeting, err = repo.CancelMeeting(ctx, id)
		if err != nil {
			return err
		}
		if _, err := repo.RecalculateCompanyStatus(ctx, meeting.CompanyID); err != nil {
			return err
		}
		if _, err := repo.AddMeetingEvent(ctx, id, model.MeetingEventCancelled, &meeting.StartAt, nil); err != nil {
			return err
		}

		meeting.History, err = repo.ListMeetingEvents(ctx, id)
		result = meeting
		return err
	})
	return result, err
}

func ensureMeetingChangeable(meeting model.Meeting, now time.Time) error {
	if meeting.Status == model.MeetingStatusCancelled {
		return model.ErrMeetingCancelled
	}
	if !meeting.StartAt.After(now) {
		return model.ErrMeetingAlreadyStarted
	}
	return nil
}
