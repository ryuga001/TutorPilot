package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"tutorpilot/internal/pkg/events"
)

type RelayConfig struct {
	PollInterval time.Duration
	BatchSize    int

	Retention map[string]time.Duration
}

type Relay struct {
	db  *pgxpool.Pool
	rdb *redis.Client
	cfg RelayConfig

	warnedNoRetention map[string]bool
}

func NewRelay(db *pgxpool.Pool, rdb *redis.Client, cfg RelayConfig) *Relay {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	return &Relay{db: db, rdb: rdb, cfg: cfg, warnedNoRetention: map[string]bool{}}
}

func (r *Relay) Run(ctx context.Context) {
	log.Printf("outbox relay: started (poll=%s batch=%d)", r.cfg.PollInterval, r.cfg.BatchSize)

	backoff := r.cfg.PollInterval
	const maxBackoff = 30 * time.Second

	for {
		n, err := r.publishBatch(ctx)
		switch {
		case ctx.Err() != nil:
			log.Printf("outbox relay: stopped")
			return
		case err != nil:
			log.Printf("outbox relay: batch failed, retrying in %s: %v", backoff, err)
			if !sleepCtx(ctx, backoff) {
				log.Printf("outbox relay: stopped")
				return
			}
			if backoff *= 2; backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		backoff = r.cfg.PollInterval

		if n == r.cfg.BatchSize {
			continue
		}
		if !sleepCtx(ctx, r.cfg.PollInterval) {
			log.Printf("outbox relay: stopped")
			return
		}
	}
}

func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}

type pendingRow struct {
	id        int64
	event     events.Event
	stream    string
	createdAt time.Time
}

func (r *Relay) publishBatch(ctx context.Context) (int, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()

	rows, err := tx.Query(ctx, `
		SELECT id, event_id, stream, event_type, version, customer_id, payload, occurred_at, created_at
		FROM outbox_events
		ORDER BY id
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, r.cfg.BatchSize)
	if err != nil {
		return 0, err
	}

	var batch []pendingRow
	for rows.Next() {
		var p pendingRow
		var eventID string
		var payload []byte
		if err := rows.Scan(&p.id, &eventID, &p.stream, &p.event.Type, &p.event.Version,
			&p.event.CustomerID, &payload, &p.event.OccurredAt, &p.createdAt); err != nil {
			rows.Close()
			return 0, err
		}
		p.event.ID, err = uuid.Parse(eventID)
		if err != nil {
			rows.Close()
			return 0, err
		}
		p.event.Payload = payload
		batch = append(batch, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(batch) == 0 {
		return 0, tx.Commit(ctx)
	}

	published, err := r.xaddAll(ctx, batch)
	if len(published) > 0 {
		if _, delErr := tx.Exec(ctx, `DELETE FROM outbox_events WHERE id = ANY($1)`, published); delErr != nil {
			return 0, delErr
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return 0, commitErr
		}
		r.logLatency(batch, published)
	}

	return len(published), err
}

func (r *Relay) xaddAll(ctx context.Context, batch []pendingRow) ([]int64, error) {
	pipe := r.rdb.Pipeline()
	cmds := make([]*redis.StringCmd, len(batch))

	for i, p := range batch {
		body, err := json.Marshal(p.event)
		if err != nil {
			return nil, err
		}
		args := &redis.XAddArgs{
			Stream: p.stream,
			Values: map[string]any{"event": body},
		}

		if d, ok := r.cfg.Retention[p.stream]; ok && d > 0 {
			args.MinID = strconv.FormatInt(time.Now().Add(-d).UnixMilli(), 10) + "-0"
			args.Approx = true
		} else if !r.warnedNoRetention[p.stream] {
			r.warnedNoRetention[p.stream] = true
			log.Printf("outbox relay: stream %q has no configured retention and will not be trimmed", p.stream)
		}
		cmds[i] = pipe.XAdd(ctx, args)
	}

	_, execErr := pipe.Exec(ctx)
	if execErr != nil && !errors.Is(execErr, redis.Nil) {
		_ = execErr
	}

	var published []int64
	var firstErr error
	for i, cmd := range cmds {
		if err := cmd.Err(); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		published = append(published, batch[i].id)
	}
	return published, firstErr
}

func (r *Relay) logLatency(batch []pendingRow, published []int64) {
	inBatch := make(map[int64]bool, len(published))
	for _, id := range published {
		inBatch[id] = true
	}

	now := time.Now()
	var total, minAge, maxAge time.Duration
	first := true
	for _, p := range batch {
		if !inBatch[p.id] {
			continue
		}
		age := now.Sub(p.createdAt)
		total += age
		if first || age < minAge {
			minAge = age
		}
		if first || age > maxAge {
			maxAge = age
		}
		first = false
	}
	if first {
		return
	}
	log.Printf("outbox relay: published %d events (queue latency min=%s avg=%s max=%s)",
		len(published), minAge.Round(time.Millisecond),
		(total / time.Duration(len(published))).Round(time.Millisecond),
		maxAge.Round(time.Millisecond))
}

func (r *Relay) Lag(ctx context.Context) (count int, oldest time.Time, err error) {
	var oldestPtr *time.Time
	err = r.db.QueryRow(ctx,
		`SELECT count(*), min(created_at) FROM outbox_events`).Scan(&count, &oldestPtr)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, time.Time{}, err
	}
	if oldestPtr != nil {
		oldest = *oldestPtr
	}
	return count, oldest, nil
}
