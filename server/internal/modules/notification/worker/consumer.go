package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"tutorpilot/internal/modules/notification/mailer"
	"tutorpilot/internal/pkg/events"
)

type Sender interface {
	SendTemplate(ctx context.Context, to, templateName string, vars map[string]string) error
}

type Config struct {
	Streams       []string
	ConsumerGroup string
	ConsumerName  string
	Concurrency   int
	MaxAttempts   int
	ClaimMinIdle  time.Duration

	BlockTimeout time.Duration
}

type Consumer struct {
	db     *pgxpool.Pool
	rdb    *redis.Client
	sender Sender
	cfg    Config
}

func New(db *pgxpool.Pool, rdb *redis.Client, sender Sender, cfg Config) *Consumer {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 4
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 5
	}
	if cfg.ClaimMinIdle <= 0 {
		cfg.ClaimMinIdle = 5 * time.Minute
	}
	if cfg.BlockTimeout <= 0 {
		cfg.BlockTimeout = 5 * time.Second
	}
	return &Consumer{db: db, rdb: rdb, sender: sender, cfg: cfg}
}

func (c *Consumer) EnsureGroups(ctx context.Context) error {
	for _, s := range c.cfg.Streams {
		err := c.rdb.XGroupCreateMkStream(ctx, s, c.cfg.ConsumerGroup, "0").Err()
		if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
			return fmt.Errorf("create group on %s: %w", s, err)
		}
	}
	return nil
}

func (c *Consumer) RunStream(ctx context.Context, stream string) {
	log.Printf("worker: consuming %s as %s/%s", stream, c.cfg.ConsumerGroup, c.cfg.ConsumerName)

	for ctx.Err() == nil {
		if err := c.reclaim(ctx, stream); err != nil && ctx.Err() == nil {
			log.Printf("worker: reclaim on %s failed: %v", stream, err)
		}

		res, err := c.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    c.cfg.ConsumerGroup,
			Consumer: c.cfg.ConsumerName,
			Streams:  []string{stream, ">"},
			Count:    int64(c.cfg.Concurrency),
			Block:    c.cfg.BlockTimeout,
		}).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) || ctx.Err() != nil {
				continue
			}
			log.Printf("worker: read from %s failed: %v", stream, err)
			if !sleepCtx(ctx, time.Second) {
				return
			}
			continue
		}

		for _, s := range res {
			c.handleMessages(ctx, s.Stream, s.Messages)
		}
	}
	log.Printf("worker: stopped consuming %s", stream)
}

func (c *Consumer) reclaim(ctx context.Context, stream string) error {
	msgs, _, err := c.rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   stream,
		Group:    c.cfg.ConsumerGroup,
		Consumer: c.cfg.ConsumerName,
		MinIdle:  c.cfg.ClaimMinIdle,
		Start:    "0",
		Count:    int64(c.cfg.Concurrency),
	}).Result()
	if err != nil {
		return err
	}
	if len(msgs) > 0 {
		c.handleMessages(ctx, stream, msgs)
	}
	return nil
}

func (c *Consumer) handleMessages(ctx context.Context, stream string, msgs []redis.XMessage) {
	sem := make(chan struct{}, c.cfg.Concurrency)
	done := make(chan struct{}, len(msgs))

	for _, m := range msgs {
		select {
		case <-ctx.Done():
			return
		case sem <- struct{}{}:
		}
		go func(m redis.XMessage) {
			defer func() { <-sem; done <- struct{}{} }()
			c.handle(ctx, stream, m)
		}(m)
	}

	for range msgs {
		select {
		case <-done:
		case <-ctx.Done():
			return
		}
	}
}

func (c *Consumer) handle(ctx context.Context, stream string, m redis.XMessage) {
	evt, err := decode(m)
	if err != nil {
		log.Printf("worker: undecodable message %s on %s: %v", m.ID, stream, err)
		if dlqErr := c.deadLetterRaw(ctx, stream, m.ID, err); dlqErr != nil {
			log.Printf("worker: could not dead-letter %s: %v", m.ID, dlqErr)
			return
		}
		c.ack(ctx, stream, m.ID)
		return
	}

	attempts, claimed, err := c.claim(ctx, evt.ID.String())
	if err != nil {
		log.Printf("worker: claim failed for %s: %v", evt.ID, err)
		return
	}
	if !claimed {
		c.ack(ctx, stream, m.ID)
		return
	}

	sendErr := c.send(ctx, evt)
	if sendErr == nil {
		if err := c.markProcessed(ctx, evt.ID.String()); err != nil {
			log.Printf("worker: sent %s but could not record it: %v", evt.ID, err)
		}
		c.ack(ctx, stream, m.ID)
		return
	}

	retryable := mailer.IsRetryable(sendErr)
	if retryable && attempts < c.cfg.MaxAttempts {
		log.Printf("worker: attempt %d/%d for %s failed (retryable): %v",
			attempts, c.cfg.MaxAttempts, evt.ID, sendErr)

		if err := c.releaseClaim(ctx, evt.ID.String()); err != nil {
			log.Printf("worker: could not release claim for %s: %v", evt.ID, err)
		}
		return
	}

	reason := "permanent"
	if retryable {
		reason = "retries exhausted"
	}
	log.Printf("worker: dead-lettering %s (%s): %v", evt.ID, reason, sendErr)
	if err := c.deadLetter(ctx, stream, evt, attempts, sendErr); err != nil {
		log.Printf("worker: could not dead-letter %s: %v", evt.ID, err)
		return
	}
	c.ack(ctx, stream, m.ID)
}

func (c *Consumer) send(ctx context.Context, evt events.Event) error {
	var p events.EmailRequested
	if err := json.Unmarshal(evt.Payload, &p); err != nil {
		return fmt.Errorf("%w: %v", mailer.ErrInvalidRecipient, err)
	}
	return c.sender.SendTemplate(ctx, p.To, p.TemplateName, p.Vars)
}

func (c *Consumer) ack(ctx context.Context, stream, id string) {
	if err := c.rdb.XAck(context.WithoutCancel(ctx), stream, c.cfg.ConsumerGroup, id).Err(); err != nil {
		log.Printf("worker: ack failed for %s: %v", id, err)
	}
}

func (c *Consumer) claim(ctx context.Context, eventID string) (attempts int, ok bool, err error) {
	err = c.db.QueryRow(ctx, `
		INSERT INTO processed_events (event_id, claimed_at, attempts)
		VALUES ($1, now(), 1)
		ON CONFLICT (event_id) DO UPDATE
		   SET claimed_at = now(),
		       attempts   = processed_events.attempts + 1
		 WHERE processed_events.processed_at IS NULL
		   AND (processed_events.claimed_at IS NULL
		        OR processed_events.claimed_at < now() - ($2 * interval '1 second'))
		RETURNING attempts`,

		eventID, c.cfg.ClaimMinIdle.Seconds(),
	).Scan(&attempts)

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return attempts, true, nil
}

func (c *Consumer) markProcessed(ctx context.Context, eventID string) error {
	_, err := c.db.Exec(context.WithoutCancel(ctx),
		`UPDATE processed_events SET processed_at = now() WHERE event_id = $1`, eventID)
	return err
}

func (c *Consumer) releaseClaim(ctx context.Context, eventID string) error {
	_, err := c.db.Exec(context.WithoutCancel(ctx),
		`UPDATE processed_events SET claimed_at = NULL WHERE event_id = $1 AND processed_at IS NULL`,
		eventID)
	return err
}

func (c *Consumer) deadLetter(ctx context.Context, stream string, evt events.Event, attempts int, cause error) error {
	stored := events.ForDLQ(evt)
	_, err := c.db.Exec(context.WithoutCancel(ctx), `
		INSERT INTO dead_events
		    (event_id, stream, event_type, version, customer_id, payload, occurred_at, last_error, attempts)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (event_id) DO UPDATE
		   SET last_error = EXCLUDED.last_error, attempts = EXCLUDED.attempts, failed_at = now()`,
		stored.ID.String(), stream, stored.Type, stored.Version, stored.CustomerID,
		string(stored.Payload), stored.OccurredAt, cause.Error(), attempts)
	return err
}

func (c *Consumer) deadLetterRaw(ctx context.Context, stream, messageID string, cause error) error {
	_, err := c.db.Exec(context.WithoutCancel(ctx), `
		INSERT INTO dead_events (stream, stream_message_id, last_error)
		VALUES ($1, $2, $3)`,
		stream, messageID, cause.Error())
	return err
}

var errTrimmed = errors.New("stream entry trimmed before delivery: payload unrecoverable")

func decode(m redis.XMessage) (events.Event, error) {
	raw, ok := m.Values["event"]
	if !ok {
		return events.Event{}, errTrimmed
	}
	s, ok := raw.(string)
	if !ok {
		return events.Event{}, fmt.Errorf("event field is %T, want string", raw)
	}
	var evt events.Event
	if err := json.Unmarshal([]byte(s), &evt); err != nil {
		return events.Event{}, err
	}
	return evt, nil
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
