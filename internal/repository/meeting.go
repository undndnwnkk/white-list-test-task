package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/undndnwnkk/white-list-test-task/internal/model"
)

const meetingColumns = `id, company_id, manager_id, start_at, end_at, original_start_at, status, created_at, updated_at`

func scanMeeting(row pgx.Row) (model.Meeting, error) {
	var m model.Meeting
	err := row.Scan(&m.ID, &m.CompanyID, &m.ManagerID, &m.StartAt, &m.EndAt,
		&m.OriginalStartAt, &m.Status, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return model.Meeting{}, err
	}
	m.StartAt = m.StartAt.UTC()
	m.EndAt = m.EndAt.UTC()
	m.OriginalStartAt = m.OriginalStartAt.UTC()
	m.CreatedAt = m.CreatedAt.UTC()
	m.UpdatedAt = m.UpdatedAt.UTC()
	return m, nil
}

func (r *Repository) CreateMeeting(ctx context.Context, companyID, managerID uuid.UUID, startAt time.Time) (model.Meeting, error) {
	m, err := scanMeeting(r.db.QueryRow(ctx,
		`INSERT INTO meetings (company_id, manager_id, start_at, end_at, original_start_at)
		 VALUES ($1, $2, $3, $4, $3)
		 RETURNING `+meetingColumns,
		companyID, managerID, startAt, startAt.Add(model.MeetingDuration),
	))
	if err != nil {
		return model.Meeting{}, mapError(err)
	}
	return m, nil
}

func (r *Repository) GetMeeting(ctx context.Context, id uuid.UUID) (model.Meeting, error) {
	m, err := scanMeeting(r.db.QueryRow(ctx,
		`SELECT `+meetingColumns+` FROM meetings WHERE id = $1`, id,
	))
	if err != nil {
		return model.Meeting{}, notFound(err, model.ErrMeetingNotFound)
	}
	return m, nil
}

func (r *Repository) LockMeeting(ctx context.Context, id uuid.UUID) (model.Meeting, error) {
	m, err := scanMeeting(r.db.QueryRow(ctx,
		`SELECT `+meetingColumns+` FROM meetings WHERE id = $1 FOR UPDATE`, id,
	))
	if err != nil {
		return model.Meeting{}, notFound(err, model.ErrMeetingNotFound)
	}
	return m, nil
}

func (r *Repository) RescheduleMeeting(ctx context.Context, id uuid.UUID, startAt time.Time) (model.Meeting, error) {
	m, err := scanMeeting(r.db.QueryRow(ctx,
		`UPDATE meetings SET start_at = $2, end_at = $3, updated_at = now()
		 WHERE id = $1
		 RETURNING `+meetingColumns,
		id, startAt, startAt.Add(model.MeetingDuration),
	))
	if err != nil {
		return model.Meeting{}, notFound(err, model.ErrMeetingNotFound)
	}
	return m, nil
}

func (r *Repository) CancelMeeting(ctx context.Context, id uuid.UUID) (model.Meeting, error) {
	m, err := scanMeeting(r.db.QueryRow(ctx,
		`UPDATE meetings SET status = 'cancelled', updated_at = now()
		 WHERE id = $1
		 RETURNING `+meetingColumns,
		id,
	))
	if err != nil {
		return model.Meeting{}, notFound(err, model.ErrMeetingNotFound)
	}
	return m, nil
}

func (r *Repository) HasActiveCompanyMeeting(ctx context.Context, companyID uuid.UUID, now time.Time, excludeID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM meetings
			WHERE company_id = $1 AND status = 'scheduled' AND end_at > $2 AND id <> $3
		)`,
		companyID, now, excludeID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check active company meeting: %w", err)
	}
	return exists, nil
}

func (r *Repository) ListManagerMeetings(ctx context.Context, managerID uuid.UUID, from, to time.Time) ([]model.Meeting, error) {
	rows, err := r.db.Query(ctx,
		`SELECT `+meetingColumns+` FROM meetings
		 WHERE manager_id = $1 AND status = 'scheduled' AND start_at < $3 AND end_at > $2
		 ORDER BY start_at`,
		managerID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("list manager meetings: %w", err)
	}
	defer rows.Close()

	meetings := []model.Meeting{}
	for rows.Next() {
		m, err := scanMeeting(rows)
		if err != nil {
			return nil, fmt.Errorf("scan meeting: %w", err)
		}
		meetings = append(meetings, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate meetings: %w", err)
	}
	return meetings, nil
}

func (r *Repository) AddMeetingEvent(ctx context.Context, meetingID uuid.UUID, eventType model.MeetingEventType, oldStartAt, newStartAt *time.Time) (model.MeetingEvent, error) {
	var e model.MeetingEvent
	err := r.db.QueryRow(ctx,
		`INSERT INTO meeting_events (meeting_id, type, old_start_at, new_start_at)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, meeting_id, type, old_start_at, new_start_at, created_at`,
		meetingID, string(eventType), oldStartAt, newStartAt,
	).Scan(&e.ID, &e.MeetingID, &e.Type, &e.OldStartAt, &e.NewStartAt, &e.CreatedAt)
	if err != nil {
		return model.MeetingEvent{}, mapError(err)
	}
	normalizeEvent(&e)
	return e, nil
}

func (r *Repository) ListMeetingEvents(ctx context.Context, meetingID uuid.UUID) ([]model.MeetingEvent, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, meeting_id, type, old_start_at, new_start_at, created_at
		 FROM meeting_events
		 WHERE meeting_id = $1
		 ORDER BY created_at, id`,
		meetingID,
	)
	if err != nil {
		return nil, fmt.Errorf("list meeting events: %w", err)
	}
	defer rows.Close()

	events := []model.MeetingEvent{}
	for rows.Next() {
		var e model.MeetingEvent
		if err := rows.Scan(&e.ID, &e.MeetingID, &e.Type, &e.OldStartAt, &e.NewStartAt, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan meeting event: %w", err)
		}
		normalizeEvent(&e)
		events = append(events, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate meeting events: %w", err)
	}
	return events, nil
}

func normalizeEvent(e *model.MeetingEvent) {
	e.CreatedAt = e.CreatedAt.UTC()
	if e.OldStartAt != nil {
		t := e.OldStartAt.UTC()
		e.OldStartAt = &t
	}
	if e.NewStartAt != nil {
		t := e.NewStartAt.UTC()
		e.NewStartAt = &t
	}
}
