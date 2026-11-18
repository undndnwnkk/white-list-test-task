package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
	"github.com/undndnwnkk/white-list-test-task/internal/repository"
)

var (
	managerA = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	managerB = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	testNow  = time.Date(2030, 1, 6, 12, 0, 0, 0, time.UTC)
)

func setupService(t *testing.T) *Service {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN is not set")
	}

	ctx := context.Background()
	pool, err := repository.NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, `TRUNCATE calls, meeting_events, meetings, companies`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return New(repository.New(pool), func() time.Time { return testNow })
}

var binCounter int

func newCompany(t *testing.T, s *Service) model.Company {
	t.Helper()
	binCounter++
	c, err := s.CreateCompany(context.Background(), model.CreateCompanyRequest{
		Name:  fmt.Sprintf("Company %d", binCounter),
		BIN:   fmt.Sprintf("%012d", binCounter),
		Phone: "+77011234567",
	})
	if err != nil {
		t.Fatalf("create company: %v", err)
	}
	return c
}

func newMeeting(t *testing.T, s *Service, companyID, managerID uuid.UUID, start time.Time) model.Meeting {
	t.Helper()
	m, err := s.CreateMeeting(context.Background(), model.CreateMeetingRequest{
		CompanyID: companyID, ManagerID: managerID, StartAt: start,
	})
	if err != nil {
		t.Fatalf("create meeting at %s: %v", start, err)
	}
	return m
}

func TestManagerMeetingsCannotOverlap(t *testing.T) {
	s := setupService(t)
	ctx := context.Background()

	newMeeting(t, s, newCompany(t, s).ID, managerA, almatyTime(7, 10, 0))

	_, err := s.CreateMeeting(ctx, model.CreateMeetingRequest{
		CompanyID: newCompany(t, s).ID, ManagerID: managerA, StartAt: almatyTime(7, 10, 30),
	})
	if !errors.Is(err, model.ErrManagerBusy) {
		t.Fatalf("overlapping meeting: got %v, want ErrManagerBusy", err)
	}

	_, err = s.CreateMeeting(ctx, model.CreateMeetingRequest{
		CompanyID: newCompany(t, s).ID, ManagerID: managerA, StartAt: almatyTime(7, 9, 30),
	})
	if !errors.Is(err, model.ErrManagerBusy) {
		t.Fatalf("meeting overlapping from the left: got %v, want ErrManagerBusy", err)
	}

	newMeeting(t, s, newCompany(t, s).ID, managerA, almatyTime(7, 11, 0))
	newMeeting(t, s, newCompany(t, s).ID, managerA, almatyTime(7, 9, 0))
}

func TestMeetingOutsideWorkingHoursIsRejected(t *testing.T) {
	s := setupService(t)
	company := newCompany(t, s)

	_, err := s.CreateMeeting(context.Background(), model.CreateMeetingRequest{
		CompanyID: company.ID, ManagerID: managerA, StartAt: almatyTime(7, 17, 30),
	})
	if !errors.Is(err, model.ErrOutsideWorkingHours) {
		t.Fatalf("got %v, want ErrOutsideWorkingHours", err)
	}

	newMeeting(t, s, company.ID, managerA, almatyTime(7, 17, 0))
}

func TestMeetingInPastIsRejected(t *testing.T) {
	s := setupService(t)

	_, err := s.CreateMeeting(context.Background(), model.CreateMeetingRequest{
		CompanyID: newCompany(t, s).ID, ManagerID: managerA, StartAt: almatyTime(3, 10, 0),
	})
	if !errors.Is(err, model.ErrMeetingInPast) {
		t.Fatalf("got %v, want ErrMeetingInPast", err)
	}
}

func TestCompanyCannotHaveTwoActiveMeetings(t *testing.T) {
	s := setupService(t)
	ctx := context.Background()
	company := newCompany(t, s)

	first := newMeeting(t, s, company.ID, managerA, almatyTime(7, 10, 0))

	_, err := s.CreateMeeting(ctx, model.CreateMeetingRequest{
		CompanyID: company.ID, ManagerID: managerB, StartAt: almatyTime(8, 15, 0),
	})
	if !errors.Is(err, model.ErrCompanyHasActiveMeeting) {
		t.Fatalf("got %v, want ErrCompanyHasActiveMeeting", err)
	}

	cancelled := model.MeetingStatusCancelled
	if _, err := s.UpdateMeeting(ctx, first.ID, model.UpdateMeetingRequest{Status: &cancelled}); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	newMeeting(t, s, company.ID, managerB, almatyTime(8, 15, 0))
}

func TestUnknownCompanyAndManager(t *testing.T) {
	s := setupService(t)
	ctx := context.Background()

	_, err := s.CreateMeeting(ctx, model.CreateMeetingRequest{
		CompanyID: uuid.New(), ManagerID: managerA, StartAt: almatyTime(7, 10, 0),
	})
	if !errors.Is(err, model.ErrCompanyNotFound) {
		t.Fatalf("got %v, want ErrCompanyNotFound", err)
	}

	_, err = s.CreateMeeting(ctx, model.CreateMeetingRequest{
		CompanyID: newCompany(t, s).ID, ManagerID: uuid.New(), StartAt: almatyTime(7, 10, 0),
	})
	if !errors.Is(err, model.ErrManagerNotFound) {
		t.Fatalf("got %v, want ErrManagerNotFound", err)
	}
}

func TestRescheduleKeepsHistory(t *testing.T) {
	s := setupService(t)
	ctx := context.Background()
	original := almatyTime(7, 10, 0)
	moved := almatyTime(8, 14, 0)

	m := newMeeting(t, s, newCompany(t, s).ID, managerA, original)

	updated, err := s.UpdateMeeting(ctx, m.ID, model.UpdateMeetingRequest{StartAt: &moved})
	if err != nil {
		t.Fatalf("reschedule: %v", err)
	}

	if !updated.StartAt.Equal(moved) || !updated.EndAt.Equal(moved.Add(model.MeetingDuration)) {
		t.Fatalf("unexpected time: %s - %s", updated.StartAt, updated.EndAt)
	}
	if !updated.OriginalStartAt.Equal(original) {
		t.Fatalf("original_start_at changed to %s", updated.OriginalStartAt)
	}
	if len(updated.History) != 2 {
		t.Fatalf("expected 2 history events, got %d", len(updated.History))
	}
	ev := updated.History[1]
	if ev.Type != model.MeetingEventRescheduled || !ev.OldStartAt.Equal(original) || !ev.NewStartAt.Equal(moved) {
		t.Fatalf("unexpected reschedule event: %+v", ev)
	}
}

func TestRescheduleValidatesNewTime(t *testing.T) {
	s := setupService(t)
	ctx := context.Background()
	original := almatyTime(7, 10, 0)
	m := newMeeting(t, s, newCompany(t, s).ID, managerA, original)
	newMeeting(t, s, newCompany(t, s).ID, managerA, almatyTime(7, 12, 0))

	cases := []struct {
		name  string
		start time.Time
		want  error
	}{
		{"outside working hours", almatyTime(7, 17, 30), model.ErrOutsideWorkingHours},
		{"overlaps another meeting of the manager", almatyTime(7, 12, 30), model.ErrManagerBusy},
		{"same time", original, model.ErrValidation},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := s.UpdateMeeting(ctx, m.ID, model.UpdateMeetingRequest{StartAt: &tc.start})
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}

	current, err := s.GetMeeting(ctx, m.ID)
	if err != nil {
		t.Fatalf("get meeting: %v", err)
	}
	if !current.StartAt.Equal(original) || len(current.History) != 1 {
		t.Fatalf("failed reschedule must not change meeting: %+v", current)
	}
}

func TestCancelledMeetingCannotBeChanged(t *testing.T) {
	s := setupService(t)
	ctx := context.Background()
	m := newMeeting(t, s, newCompany(t, s).ID, managerA, almatyTime(7, 10, 0))
	cancelled := model.MeetingStatusCancelled

	if _, err := s.UpdateMeeting(ctx, m.ID, model.UpdateMeetingRequest{Status: &cancelled}); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if _, err := s.UpdateMeeting(ctx, m.ID, model.UpdateMeetingRequest{Status: &cancelled}); !errors.Is(err, model.ErrMeetingCancelled) {
		t.Fatalf("second cancel: got %v, want ErrMeetingCancelled", err)
	}
	newStart := almatyTime(8, 10, 0)
	if _, err := s.UpdateMeeting(ctx, m.ID, model.UpdateMeetingRequest{StartAt: &newStart}); !errors.Is(err, model.ErrMeetingCancelled) {
		t.Fatalf("reschedule cancelled: got %v, want ErrMeetingCancelled", err)
	}
}

func TestCallMeetingSetRules(t *testing.T) {
	s := setupService(t)
	ctx := context.Background()
	company := newCompany(t, s)
	other := newCompany(t, s)
	otherMeeting := newMeeting(t, s, other.ID, managerB, almatyTime(7, 10, 0))

	_, err := s.CreateCall(ctx, model.CreateCallRequest{
		CompanyID: company.ID, ManagerID: managerA, Result: model.CallResultMeetingSet,
	})
	if !errors.Is(err, model.ErrValidation) {
		t.Fatalf("without meeting: got %v, want validation error", err)
	}

	_, err = s.CreateCall(ctx, model.CreateCallRequest{
		CompanyID: company.ID, ManagerID: managerA, Result: model.CallResultMeetingSet, MeetingID: &otherMeeting.ID,
	})
	if !errors.Is(err, model.ErrMeetingCompanyMismatch) {
		t.Fatalf("foreign meeting: got %v, want ErrMeetingCompanyMismatch", err)
	}

	unknown := uuid.New()
	_, err = s.CreateCall(ctx, model.CreateCallRequest{
		CompanyID: company.ID, ManagerID: managerA, Result: model.CallResultMeetingSet, MeetingID: &unknown,
	})
	if !errors.Is(err, model.ErrMeetingNotFound) {
		t.Fatalf("unknown meeting: got %v, want ErrMeetingNotFound", err)
	}

	m := newMeeting(t, s, company.ID, managerA, almatyTime(7, 10, 0))
	call, err := s.CreateCall(ctx, model.CreateCallRequest{
		CompanyID: company.ID, ManagerID: managerA, Result: model.CallResultMeetingSet, MeetingID: &m.ID,
	})
	if err != nil {
		t.Fatalf("valid call: %v", err)
	}
	if call.MeetingID == nil || *call.MeetingID != m.ID {
		t.Fatalf("meeting not linked: %+v", call)
	}

	status := model.CompanyStatusMeetingSet
	list, err := s.ListCompanies(ctx, model.ListCompaniesFilter{Status: &status})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if list.Total != 1 || list.Items[0].ID != company.ID {
		t.Fatalf("company status not updated: %+v", list)
	}
}

func TestManagerScheduleForAlmatyDay(t *testing.T) {
	s := setupService(t)
	ctx := context.Background()
	monday := newMeeting(t, s, newCompany(t, s).ID, managerA, almatyTime(7, 9, 0))
	newMeeting(t, s, newCompany(t, s).ID, managerA, almatyTime(8, 9, 0))
	newMeeting(t, s, newCompany(t, s).ID, managerB, almatyTime(7, 11, 0))

	schedule, err := s.GetManagerSchedule(ctx, managerA, "2030-01-07")
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	if len(schedule.Meetings) != 1 || schedule.Meetings[0].ID != monday.ID {
		t.Fatalf("unexpected schedule: %+v", schedule.Meetings)
	}

	if _, err := s.GetManagerSchedule(ctx, uuid.New(), "2030-01-07"); !errors.Is(err, model.ErrManagerNotFound) {
		t.Fatalf("unknown manager: got %v, want ErrManagerNotFound", err)
	}
}

func TestConcurrentBookingOfSameSlot(t *testing.T) {
	s := setupService(t)
	ctx := context.Background()
	const workers = 10

	companies := make([]model.Company, workers)
	for i := range companies {
		companies[i] = newCompany(t, s)
	}
	sharedCompany := newCompany(t, s)
	managers := []uuid.UUID{managerA, managerB, uuid.MustParse("33333333-3333-3333-3333-333333333333")}

	run := func(build func(i int) model.CreateMeetingRequest) (int, []error) {
		var (
			wg      sync.WaitGroup
			mu      sync.Mutex
			success int
			errs    []error
		)
		for i := 0; i < workers; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_, err := s.CreateMeeting(ctx, build(i))
				mu.Lock()
				defer mu.Unlock()
				if err == nil {
					success++
				} else {
					errs = append(errs, err)
				}
			}(i)
		}
		wg.Wait()
		return success, errs
	}

	success, errs := run(func(i int) model.CreateMeetingRequest {
		return model.CreateMeetingRequest{CompanyID: companies[i].ID, ManagerID: managerA, StartAt: almatyTime(7, 10, 0)}
	})
	if success != 1 {
		t.Fatalf("same manager slot: %d successful bookings, want 1", success)
	}
	for _, err := range errs {
		if !errors.Is(err, model.ErrManagerBusy) {
			t.Fatalf("same manager slot: unexpected error %v", err)
		}
	}

	success, errs = run(func(i int) model.CreateMeetingRequest {
		return model.CreateMeetingRequest{CompanyID: sharedCompany.ID, ManagerID: managers[i%len(managers)], StartAt: almatyTime(8, 9+i%8, 0)}
	})
	if success != 1 {
		t.Fatalf("same company: %d successful bookings, want 1", success)
	}
	for _, err := range errs {
		if !errors.Is(err, model.ErrCompanyHasActiveMeeting) && !errors.Is(err, model.ErrManagerBusy) {
			t.Fatalf("same company: unexpected error %v", err)
		}
	}
}
