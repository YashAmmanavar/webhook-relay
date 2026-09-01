CREATE TABLE IF NOT EXISTS endpoints (
    id          TEXT PRIMARY KEY,
    url         TEXT NOT NULL,
    secret      TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
