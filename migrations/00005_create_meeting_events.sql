-- +goose Up
CREATE TABLE meeting_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id   UUID        NOT NULL REFERENCES meetings (id) ON DELETE CASCADE,
    type         TEXT        NOT NULL CHECK (type IN ('created', 'rescheduled', 'cancelled')),
    old_start_at TIMESTAMPTZ,
    new_start_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_meeting_events_meeting ON meeting_events (meeting_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS meeting_events;
