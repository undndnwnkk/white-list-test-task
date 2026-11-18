-- +goose Up
-- A meeting is the outcome of exactly one call: it cannot be reported twice.
CREATE UNIQUE INDEX calls_meeting_id_key ON calls (meeting_id) WHERE meeting_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS calls_meeting_id_key;
