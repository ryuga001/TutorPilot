-- Drops the eventing tables. Any events still queued in outbox_events, and the
-- whole dead_events audit trail, are lost -- there is nowhere else to put them.

DROP TABLE IF EXISTS dead_events;
DROP TABLE IF EXISTS processed_events;
DROP TABLE IF EXISTS outbox_events;
