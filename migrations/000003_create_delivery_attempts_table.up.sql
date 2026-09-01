CREATE TABLE IF NOT EXISTS delivery_attempts (
    id             BIGSERIAL PRIMARY KEY,
    event_id       TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    attempt_number INTEGER NOT NULL,
    status_code    INTEGER,
    response_body  TEXT,
    error          TEXT,
    duration_ms    INTEGER,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_delivery_attempts_event_id ON delivery_attempts(event_id);
