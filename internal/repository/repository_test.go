package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

var (
	managerA = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	managerB = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	slot     = time.Date(2030, 1, 7, 5, 0, 0, 0, time.UTC)
)

func setup(t *testing.T) *Repository {
	t.Helper()
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		t.Skip("TEST_DB_DSN is not set")
	}

	ctx := context.Background()
	pool, err := NewPool(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(ctx, `TRUNCATE calls, meeting_events, meetings, companies`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return New(pool)
}

func createCompany(t *testing.T, r *Repository, bin string) model.Company {
	t.Helper()
	c, err := r.CreateCompany(context.Background(), model.CreateCompanyRequest{
		Name: "Company " + bin, BIN: bin, Phone: "+77010000000",
	})
	if err != nil {
		t.Fatalf("create company: %v", err)
	}
	return c
}

func TestCreateCompanyDuplicateBIN(t *testing.T) {
	r := setup(t)
	createCompany(t, r, "170340021554")

	_, err := r.CreateCompany(context.Background(), model.CreateCompanyRequest{
		Name: "Other", BIN: "170340021554", Phone: "+77010000001",
	})
	if !errors.Is(err, model.ErrBINAlreadyExists) {
		t.Fatalf("expected ErrBINAlreadyExists, got %v", err)
	}
}

func TestManagerMeetingsOverlap(t *testing.T) {
	r := setup(t)
	ctx := context.Background()
	c1 := createCompany(t, r, "100000000001")
	c2 := createCompany(t, r, "100000000002")
	c3 := createCompany(t, r, "100000000003")

	if _, err := r.CreateMeeting(ctx, c1.ID, managerA, slot); err != nil {
		t.Fatalf("first meeting: %v", err)
	}

	_, err := r.CreateMeeting(ctx, c2.ID, managerA, slot.Add(30*time.Minute))
	if !errors.Is(err, model.ErrManagerBusy) {
		t.Fatalf("expected ErrManagerBusy, got %v", err)
	}

	if _, err := r.CreateMeeting(ctx, c3.ID, managerA, slot.Add(time.Hour)); err != nil {
		t.Fatalf("adjacent meeting must be allowed: %v", err)
	}
}

func TestCompanyMeetingsOverlapWithDifferentManagers(t *testing.T) {
	r := setup(t)
	ctx := context.Background()
	c := createCompany(t, r, "100000000001")

	if _, err := r.CreateMeeting(ctx, c.ID, managerA, slot); err != nil {
		t.Fatalf("first meeting: %v", err)
	}

	_, err := r.CreateMeeting(ctx, c.ID, managerB, slot.Add(15*time.Minute))
	if !errors.Is(err, model.ErrCompanyBusy) {
		t.Fatalf("expected ErrCompanyBusy, got %v", err)
	}
}

func TestCancelledMeetingFreesSlot(t *testing.T) {
	r := setup(t)
	ctx := context.Background()
	c1 := createCompany(t, r, "100000000001")
	c2 := createCompany(t, r, "100000000002")

	m, err := r.CreateMeeting(ctx, c1.ID, managerA, slot)
	if err != nil {
		t.Fatalf("create meeting: %v", err)
	}
	if _, err := r.CancelMeeting(ctx, m.ID); err != nil {
		t.Fatalf("cancel meeting: %v", err)
	}
	if _, err := r.CreateMeeting(ctx, c2.ID, managerA, slot); err != nil {
		t.Fatalf("slot must be free after cancel: %v", err)
	}
}

func TestRescheduleKeepsOriginalStart(t *testing.T) {
	r := setup(t)
	ctx := context.Background()
	c := createCompany(t, r, "100000000001")

	m, err := r.CreateMeeting(ctx, c.ID, managerA, slot)
	if err != nil {
		t.Fatalf("create meeting: %v", err)
	}

	newStart := slot.Add(2 * time.Hour)
	moved, err := r.RescheduleMeeting(ctx, m.ID, newStart)
	if err != nil {
		t.Fatalf("reschedule: %v", err)
	}
	if !moved.StartAt.Equal(newStart) || !moved.EndAt.Equal(newStart.Add(model.MeetingDuration)) {
		t.Fatalf("unexpected new time: %v - %v", moved.StartAt, moved.EndAt)
	}
	if !moved.OriginalStartAt.Equal(slot) {
		t.Fatalf("original start changed: %v", moved.OriginalStartAt)
	}
}

func TestUnknownReferencesAreMapped(t *testing.T) {
	r := setup(t)
	ctx := context.Background()
	c := createCompany(t, r, "100000000001")

	_, err := r.CreateMeeting(ctx, c.ID, uuid.New(), slot)
	if !errors.Is(err, model.ErrManagerNotFound) {
		t.Fatalf("expected ErrManagerNotFound, got %v", err)
	}

	_, err = r.CreateMeeting(ctx, uuid.New(), managerA, slot)
	if !errors.Is(err, model.ErrCompanyNotFound) {
		t.Fatalf("expected ErrCompanyNotFound, got %v", err)
	}

	_, err = r.GetMeeting(ctx, uuid.New())
	if !errors.Is(err, model.ErrMeetingNotFound) {
		t.Fatalf("expected ErrMeetingNotFound, got %v", err)
	}
}

func TestCallMeetingSetRequiresMeeting(t *testing.T) {
	r := setup(t)
	ctx := context.Background()
	c := createCompany(t, r, "100000000001")

	_, err := r.CreateCall(ctx, model.CreateCallRequest{
		CompanyID: c.ID, ManagerID: managerA, Result: model.CallResultMeetingSet,
	})
	if !errors.Is(err, model.ErrInvalidData) {
		t.Fatalf("expected ErrInvalidData, got %v", err)
	}

	m, err := r.CreateMeeting(ctx, c.ID, managerA, slot)
	if err != nil {
		t.Fatalf("create meeting: %v", err)
	}
	call, err := r.CreateCall(ctx, model.CreateCallRequest{
		CompanyID: c.ID, ManagerID: managerA, Result: model.CallResultMeetingSet, MeetingID: &m.ID,
	})
	if err != nil {
		t.Fatalf("create call: %v", err)
	}
	if call.MeetingID == nil || *call.MeetingID != m.ID {
		t.Fatalf("meeting id not stored: %v", call.MeetingID)
	}
}

func TestListCompaniesFilterAndPagination(t *testing.T) {
	r := setup(t)
	ctx := context.Background()
	c1 := createCompany(t, r, "100000000001")
	createCompany(t, r, "100000000002")
	createCompany(t, r, "100000000003")

	if err := r.UpdateCompanyStatus(ctx, c1.ID, model.CompanyStatusRefused); err != nil {
		t.Fatalf("update status: %v", err)
	}

	status := model.CompanyStatusNew
	list, err := r.ListCompanies(ctx, model.ListCompaniesFilter{Status: &status, Limit: 1, Offset: 0})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if list.Total != 2 || len(list.Items) != 1 || list.Items[0].Status != model.CompanyStatusNew {
		t.Fatalf("unexpected list: total=%d items=%+v", list.Total, list.Items)
	}
}

func TestWithTxRollsBackOnError(t *testing.T) {
	r := setup(t)
	ctx := context.Background()
	c := createCompany(t, r, "100000000001")
	boom := errors.New("boom")

	err := r.WithTx(ctx, func(tx *Repository) error {
		if _, err := tx.CreateMeeting(ctx, c.ID, managerA, slot); err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("expected boom, got %v", err)
	}

	meetings, err := r.ListManagerMeetings(ctx, managerA, slot.Add(-time.Hour), slot.Add(time.Hour))
	if err != nil {
		t.Fatalf("list meetings: %v", err)
	}
	if len(meetings) != 0 {
		t.Fatalf("meeting must be rolled back, got %d", len(meetings))
	}
}
