package outbox

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tutorpilot/internal/pkg/events"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run outbox integration tests")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func testEvent(t *testing.T) events.Event {
	t.Helper()
	evt, err := events.NewEmail(events.TypeEmailRequested, 1, time.Now(), events.EmailRequested{
		To:           "test@example.com",
		TemplateName: "member_invite",
		Vars:         map[string]string{"name": "Ada"},
	})
	if err != nil {
		t.Fatalf("NewEmail: %v", err)
	}
	return evt
}

func countByID(t *testing.T, pool *pgxpool.Pool, eventID string) int {
	t.Helper()
	var n int
	err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM outbox_events WHERE event_id = $1`, eventID).Scan(&n)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	return n
}

func TestInsertRollsBackWithItsTransaction(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	evt := testEvent(t)

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := Insert(ctx, tx, "test:stream", evt); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	if n := countByID(t, pool, evt.ID.String()); n != 0 {
		t.Errorf("found %d outbox rows after rollback, want 0", n)
	}
}

func TestInsertCommitsWithItsTransaction(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	evt := testEvent(t)
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM outbox_events WHERE event_id = $1`, evt.ID.String())
	})

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if err := Insert(ctx, tx, "test:stream", evt); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}

	if n := countByID(t, pool, evt.ID.String()); n != 1 {
		t.Errorf("found %d outbox rows after commit, want 1", n)
	}
}

func TestInsertBatchWritesEveryEvent(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()

	batch := []events.Event{testEvent(t), testEvent(t), testEvent(t)}
	t.Cleanup(func() {
		for _, e := range batch {
			_, _ = pool.Exec(ctx, `DELETE FROM outbox_events WHERE event_id = $1`, e.ID.String())
		}
	})

	if err := InsertBatch(ctx, pool, "test:stream", batch); err != nil {
		t.Fatalf("InsertBatch: %v", err)
	}

	for _, e := range batch {
		if n := countByID(t, pool, e.ID.String()); n != 1 {
			t.Errorf("event %s: found %d rows, want 1", e.ID, n)
		}
	}
}

func TestInsertBatchEmptyIsNoop(t *testing.T) {
	pool := testPool(t)

	if err := InsertBatch(context.Background(), pool, "test:stream", nil); err != nil {
		t.Errorf("InsertBatch(nil) = %v, want nil", err)
	}
}
