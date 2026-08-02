-- Transactional outbox + notification-worker bookkeeping.
--
-- Ownership note for the eventual service split: outbox_events belongs to each
-- PRODUCING service (admin, auth) and would be duplicated per service database.
-- processed_events and dead_events belong to the NOTIFICATION WORKER. They live
-- in one database today only because this is still a modular monolith.

CREATE TABLE outbox_events (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id    UUID        NOT NULL UNIQUE,
    stream      TEXT        NOT NULL,
    event_type  TEXT        NOT NULL,
    version     INT         NOT NULL,
    customer_id INT         NOT NULL,
    payload     JSONB       NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE outbox_events IS
    'Producer-owned. Rows are written in the same transaction as the business change they describe, then hard-deleted by the relay once published to Redis. Only unpublished rows are ever present, so the poll query rides the primary key and needs no extra index.';
COMMENT ON COLUMN outbox_events.event_id IS
    'Idempotency key. Travels to the stream and on to processed_events; survives a replay unchanged.';
COMMENT ON COLUMN outbox_events.created_at IS
    'Used by the relay to log queue latency immediately before deleting the row -- the only point at which that number is observable.';

CREATE TABLE processed_events (
    event_id     UUID PRIMARY KEY,
    attempts     INT NOT NULL DEFAULT 0,
    claimed_at   TIMESTAMPTZ,
    processed_at TIMESTAMPTZ
);

COMMENT ON TABLE processed_events IS
    'Worker-owned. Doubles as the dedupe record and the in-flight claim, which is what stops two consumers sending the same email when one stalls past the reclaim window. State is encoded by the two timestamps: claimed_at set + processed_at null = in flight; both null = failed, free to retry; processed_at set = delivered (terminal).';

-- The hourly purge is `WHERE processed_at < now() - interval '30 days'`. Rows
-- still in flight or awaiting retry have a NULL processed_at and can never
-- match, so the index excludes them.
CREATE INDEX processed_events_purge_idx
    ON processed_events (processed_at)
    WHERE processed_at IS NOT NULL;

CREATE TABLE dead_events (
    id                BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    event_id          UUID UNIQUE,
    stream            TEXT        NOT NULL,
    stream_message_id TEXT,
    event_type        TEXT,
    version           INT,
    customer_id       INT,
    payload           JSONB,
    occurred_at       TIMESTAMPTZ,
    last_error        TEXT        NOT NULL,
    attempts          INT         NOT NULL DEFAULT 0,
    failed_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    replayed_at       TIMESTAMPTZ
);

COMMENT ON TABLE dead_events IS
    'Worker-owned audit trail. Deliberately has no automatic retention -- the depth log is what keeps it from growing unnoticed. Triage with: SELECT event_type, last_error, attempts, failed_at FROM dead_events WHERE replayed_at IS NULL ORDER BY failed_at DESC;';
COMMENT ON COLUMN dead_events.event_id IS
    'Null when the stream entry was trimmed before it could be delivered: only the Redis message id survives in that case. Nullable UNIQUE permits many such rows because Postgres treats NULLs as distinct.';
COMMENT ON COLUMN dead_events.payload IS
    'Stored intact for replayable events and redacted for one-time codes -- see events.Redactable. An invite payload MUST be kept whole: its temporary password exists nowhere else, so a redacted row could never be replayed.';
