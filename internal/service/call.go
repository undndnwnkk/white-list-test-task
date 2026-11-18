package service

import (
	"context"

	"github.com/google/uuid"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
	"github.com/undndnwnkk/white-list-test-task/internal/repository"
)

func (s *Service) CreateCall(ctx context.Context, req model.CreateCallRequest) (model.Call, error) {
	if err := validateCallRequest(req); err != nil {
		return model.Call{}, err
	}

	now := s.now()

	var result model.Call
	err := s.repo.WithTx(ctx, func(repo *repository.Repository) error {
		if _, err := repo.LockCompany(ctx, req.CompanyID); err != nil {
			return err
		}
		if _, err := repo.GetManager(ctx, req.ManagerID); err != nil {
			return err
		}

		if req.Result == model.CallResultMeetingSet {
			meeting, err := repo.GetMeeting(ctx, *req.MeetingID)
			if err != nil {
				return err
			}
			if meeting.CompanyID != req.CompanyID {
				return model.ErrMeetingCompanyMismatch
			}
			if err := ensureMeetingChangeable(meeting, now); err != nil {
				return err
			}
		}

		call, err := repo.CreateCall(ctx, req)
		if err != nil {
			return err
		}
		if err := repo.UpdateCompanyStatus(ctx, req.CompanyID, req.Result.CompanyStatus()); err != nil {
			return err
		}

		result = call
		return nil
	})
	return result, err
}

func validateCallRequest(req model.CreateCallRequest) error {
	if req.CompanyID == uuid.Nil {
		return model.NewValidationError("company_id", "is required")
	}
	if req.ManagerID == uuid.Nil {
		return model.NewValidationError("manager_id", "is required")
	}
	if !req.Result.Valid() {
		return model.NewValidationError("result", "must be one of: no_answer, refused, meeting_set")
	}
	if req.Result == model.CallResultMeetingSet && (req.MeetingID == nil || *req.MeetingID == uuid.Nil) {
		return model.NewValidationError("meeting_id", "is required when result is meeting_set: create the meeting first")
	}
	if req.Result != model.CallResultMeetingSet && req.MeetingID != nil {
		return model.NewValidationError("meeting_id", "is allowed only when result is meeting_set")
	}
	return nil
}
