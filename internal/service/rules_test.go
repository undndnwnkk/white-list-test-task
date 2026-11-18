package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

func almatyTime(day, hour, minute int) time.Time {
	return time.Date(2030, time.January, day, hour, minute, 0, 0, almaty)
}

func TestValidateWorkingHoursBoundaries(t *testing.T) {
	cases := []struct {
		name  string
		start time.Time
		want  error
	}{
		{"monday 09:00 is the first allowed slot", almatyTime(7, 9, 0), nil},
		{"monday 08:59 starts before working day", almatyTime(7, 8, 59), model.ErrOutsideWorkingHours},
		{"monday 12:00 midday", almatyTime(7, 12, 0), nil},
		{"monday 17:00 ends exactly at 18:00", almatyTime(7, 17, 0), nil},
		{"monday 17:01 ends after 18:00", almatyTime(7, 17, 1), model.ErrOutsideWorkingHours},
		{"monday 17:30 ends after 18:00", almatyTime(7, 17, 30), model.ErrOutsideWorkingHours},
		{"monday 18:00 starts after working day", almatyTime(7, 18, 0), model.ErrOutsideWorkingHours},
		{"friday 17:00 is allowed", almatyTime(11, 17, 0), nil},
		{"saturday 10:00 is a day off", almatyTime(12, 10, 0), model.ErrOutsideWorkingHours},
		{"sunday 10:00 is a day off", almatyTime(13, 10, 0), model.ErrOutsideWorkingHours},
		{"03:59 UTC is 08:59 in Almaty", time.Date(2030, 1, 7, 3, 59, 0, 0, time.UTC), model.ErrOutsideWorkingHours},
		{"04:00 UTC is 09:00 in Almaty", time.Date(2030, 1, 7, 4, 0, 0, 0, time.UTC), nil},
		{"12:00 UTC is 17:00 in Almaty", time.Date(2030, 1, 7, 12, 0, 0, 0, time.UTC), nil},
		{"12:30 UTC is 17:30 in Almaty", time.Date(2030, 1, 7, 12, 30, 0, 0, time.UTC), model.ErrOutsideWorkingHours},
		{"sunday 23:00 UTC is monday 04:00 in Almaty", time.Date(2030, 1, 6, 23, 0, 0, 0, time.UTC), model.ErrOutsideWorkingHours},
		{"friday 22:00 UTC is saturday 03:00 in Almaty", time.Date(2030, 1, 11, 22, 0, 0, 0, time.UTC), model.ErrOutsideWorkingHours},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateWorkingHours(tc.start); !errors.Is(err, tc.want) {
				t.Fatalf("ValidateWorkingHours(%s) = %v, want %v", tc.start.Format(time.RFC3339), err, tc.want)
			}
		})
	}
}

func TestValidateNewMeetingStart(t *testing.T) {
	now := almatyTime(7, 12, 0)

	cases := []struct {
		name  string
		start time.Time
		want  error
	}{
		{"zero time", time.Time{}, model.ErrValidation},
		{"in the past", almatyTime(7, 10, 0), model.ErrMeetingInPast},
		{"exactly now", now, model.ErrMeetingInPast},
		{"future within working hours", almatyTime(7, 13, 0), nil},
		{"future outside working hours", almatyTime(7, 19, 0), model.ErrOutsideWorkingHours},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateNewMeetingStart(tc.start, now); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestDayBoundsUsesAlmatyDay(t *testing.T) {
	from, to, err := DayBounds("2030-01-07")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantFrom := time.Date(2030, 1, 6, 19, 0, 0, 0, time.UTC)
	wantTo := time.Date(2030, 1, 7, 19, 0, 0, 0, time.UTC)
	if !from.Equal(wantFrom) || !to.Equal(wantTo) {
		t.Fatalf("got [%s, %s), want [%s, %s)", from, to, wantFrom, wantTo)
	}

	for _, bad := range []string{"", "07.01.2030", "2030-13-01", "2030-01-07T00:00:00Z"} {
		if _, _, err := DayBounds(bad); !errors.Is(err, model.ErrValidation) {
			t.Fatalf("DayBounds(%q) = %v, want validation error", bad, err)
		}
	}
}

func TestValidateCallRequest(t *testing.T) {
	company, manager, meeting := uuid.New(), uuid.New(), uuid.New()

	cases := []struct {
		name string
		req  model.CreateCallRequest
		want error
	}{
		{"no answer", model.CreateCallRequest{CompanyID: company, ManagerID: manager, Result: model.CallResultNoAnswer}, nil},
		{"meeting set with meeting", model.CreateCallRequest{CompanyID: company, ManagerID: manager, Result: model.CallResultMeetingSet, MeetingID: &meeting}, nil},
		{"meeting set without meeting", model.CreateCallRequest{CompanyID: company, ManagerID: manager, Result: model.CallResultMeetingSet}, model.ErrValidation},
		{"refused with meeting", model.CreateCallRequest{CompanyID: company, ManagerID: manager, Result: model.CallResultRefused, MeetingID: &meeting}, model.ErrValidation},
		{"unknown result", model.CreateCallRequest{CompanyID: company, ManagerID: manager, Result: "busy"}, model.ErrValidation},
		{"missing company", model.CreateCallRequest{ManagerID: manager, Result: model.CallResultNoAnswer}, model.ErrValidation},
		{"missing manager", model.CreateCallRequest{CompanyID: company, Result: model.CallResultNoAnswer}, model.ErrValidation},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := validateCallRequest(tc.req); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestUpdateMeetingRequestValidation(t *testing.T) {
	s := New(nil, nil)
	start := almatyTime(7, 10, 0)
	cancelled := model.MeetingStatusCancelled
	scheduled := model.MeetingStatusScheduled

	cases := []struct {
		name string
		req  model.UpdateMeetingRequest
	}{
		{"empty body", model.UpdateMeetingRequest{}},
		{"both fields", model.UpdateMeetingRequest{StartAt: &start, Status: &cancelled}},
		{"status other than cancelled", model.UpdateMeetingRequest{Status: &scheduled}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := s.UpdateMeeting(context.Background(), uuid.New(), tc.req); !errors.Is(err, model.ErrValidation) {
				t.Fatalf("got %v, want validation error", err)
			}
		})
	}
}
