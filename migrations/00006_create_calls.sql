-- +goose Up
CREATE TABLE calls (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID        NOT NULL REFERENCES companies (id),
    manager_id UUID        NOT NULL REFERENCES managers (id),
    result     TEXT        NOT NULL CHECK (result IN ('no_answer', 'refused', 'meeting_set')),
    meeting_id UUID REFERENCES meetings (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT calls_meeting_set_requires_meeting
        CHECK ((result = 'meeting_set') = (meeting_id IS NOT NULL))
);

CREATE INDEX idx_calls_company ON calls (company_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS calls;
