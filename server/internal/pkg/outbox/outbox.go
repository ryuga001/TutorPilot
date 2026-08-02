package outbox

import (
	"context"
	"time"

	"tutorpilot/internal/pkg/events"
	"tutorpilot/internal/pkg/pg"
)

func Insert(ctx context.Context, q pg.Querier, stream string, evt events.Event) error {
	_, err := q.Exec(ctx, `
		INSERT INTO outbox_events
		    (event_id, stream, event_type, version, customer_id, payload, occurred_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		evt.ID.String(), stream, evt.Type, evt.Version, evt.CustomerID,
		string(evt.Payload), evt.OccurredAt,
	)
	return err
}

func InsertBatch(ctx context.Context, q pg.Querier, stream string, evts []events.Event) error {
	if len(evts) == 0 {
		return nil
	}

	ids := make([]string, len(evts))
	types := make([]string, len(evts))
	versions := make([]int32, len(evts))
	customers := make([]int32, len(evts))
	payloads := make([]string, len(evts))
	occurred := make([]time.Time, len(evts))

	for i, e := range evts {
		ids[i] = e.ID.String()
		types[i] = e.Type
		versions[i] = int32(e.Version)
		customers[i] = int32(e.CustomerID)
		payloads[i] = string(e.Payload)
		occurred[i] = e.OccurredAt
	}

	_, err := q.Exec(ctx, `
		INSERT INTO outbox_events
		    (event_id, stream, event_type, version, customer_id, payload, occurred_at)
		SELECT u.event_id::uuid, $1, u.event_type, u.version, u.customer_id,
		       u.payload::jsonb, u.occurred_at
		FROM unnest($2::text[], $3::text[], $4::int[], $5::int[], $6::text[], $7::timestamptz[])
		     AS u(event_id, event_type, version, customer_id, payload, occurred_at)`,
		stream, ids, types, versions, customers, payloads, occurred,
	)
	return err
}
