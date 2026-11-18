-- +goose Up
CREATE TABLE meetings (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id        UUID        NOT NULL REFERENCES companies (id),
    manager_id        UUID        NOT NULL REFERENCES managers (id),
    start_at          TIMESTAMPTZ NOT NULL,
    end_at            TIMESTAMPTZ NOT NULL,
    original_start_at TIMESTAMPTZ NOT NULL,
    status            TEXT        NOT NULL DEFAULT 'scheduled'
        CHECK (status IN ('scheduled', 'cancelled')),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT meetings_duration_check CHECK (end_at = start_at + INTERVAL '60 minutes'),
    CONSTRAINT meetings_manager_no_overlap EXCLUDE USING gist (
        manager_id WITH =,
        tstzrange(start_at, end_at, '[)') WITH &&
    ) WHERE (status = 'scheduled'),
    CONSTRAINT meetings_company_no_overlap EXCLUDE USING gist (
        company_id WITH =,
        tstzrange(start_at, end_at, '[)') WITH &&
    ) WHERE (status = 'scheduled')
);

CREATE INDEX idx_meetings_manager_start ON meetings (manager_id, start_at);
CREATE INDEX idx_meetings_company_status ON meetings (company_id, status);

-- +goose Down
DROP TABLE IF EXISTS meetings;
