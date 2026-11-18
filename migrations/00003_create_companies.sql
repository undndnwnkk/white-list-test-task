-- +goose Up
CREATE TABLE companies (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT        NOT NULL CHECK (btrim(name) <> ''),
    bin        CHAR(12)    NOT NULL UNIQUE CHECK (bin ~ '^[0-9]{12}$'),
    phone      TEXT        NOT NULL CHECK (btrim(phone) <> ''),
    status     TEXT        NOT NULL DEFAULT 'new'
        CHECK (status IN ('new', 'no_answer', 'refused', 'meeting_set')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_companies_status ON companies (status, created_at);

-- +goose Down
DROP TABLE IF EXISTS companies;
