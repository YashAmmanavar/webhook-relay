CREATE TABLE IF NOT EXISTS events (
    id           TEXT PRIMARY KEY,
    endpoint_id  TEXT NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    event_type   TEXT NOT NULL,
    payload      JSONB NOT NULL,
    status       TEXT NOT NULL DEFAULT 'PENDING'
                 CHECK (status IN ('PENDING', 'PROCESSING', 'RETRYING', 'DELIVERED', 'FAILED')),
    attempt_count INTEGER NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_events_endpoint_id ON events(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_events_status ON events(status);
