package service

import (
	"time"
	_ "time/tzdata"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

const (
	workdayStartHour = 9
	workdayEndHour   = 18
)

var almaty = mustLoadLocation("Asia/Almaty")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

func ValidateWorkingHours(start time.Time) error {
	local := start.In(almaty)
	if local.Weekday() == time.Saturday || local.Weekday() == time.Sunday {
		return model.ErrOutsideWorkingHours
	}

	year, month, day := local.Date()
	dayStart := time.Date(year, month, day, workdayStartHour, 0, 0, 0, almaty)
	dayEnd := time.Date(year, month, day, workdayEndHour, 0, 0, 0, almaty)

	if local.Before(dayStart) || local.Add(model.MeetingDuration).After(dayEnd) {
		return model.ErrOutsideWorkingHours
	}
	return nil
}

func validateNewMeetingStart(start, now time.Time) error {
	if start.IsZero() {
		return model.NewValidationError("start_at", "is required")
	}
	if !start.After(now) {
		return model.ErrMeetingInPast
	}
	return ValidateWorkingHours(start)
}

func DayBounds(date string) (time.Time, time.Time, error) {
	day, err := time.ParseInLocation(time.DateOnly, date, almaty)
	if err != nil {
		return time.Time{}, time.Time{}, model.NewValidationError("date", "must be in YYYY-MM-DD format")
	}
	return day.UTC(), day.AddDate(0, 0, 1).UTC(), nil
}
