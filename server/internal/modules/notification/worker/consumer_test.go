package worker

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to run worker integration tests")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func testConsumer(t *testing.T, pool *pgxpool.Pool, minIdle time.Duration) *Consumer {
	t.Helper()

	return New(pool, nil, nil, Config{ClaimMinIdle: minIdle})
}

func newEventID(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()
	id := uuid.NewString()
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`DELETE FROM processed_events WHERE event_id = $1`, id)
	})
	return id
}

func TestConcurrentClaimersOnlyOneWins(t *testing.T) {
	pool := testPool(t)
	c := testConsumer(t, pool, 5*time.Minute)
	eventID := newEventID(t, pool)

	const racers = 8
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		wins  int
		fails []error
	)

	start := make(chan struct{})
	for i := 0; i < racers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, ok, err := c.claim(context.Background(), eventID)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				fails = append(fails, err)
				return
			}
			if ok {
				wins++
			}
		}()
	}
	close(start)
	wg.Wait()

	for _, err := range fails {
		t.Errorf("claim returned an error: %v", err)
	}
	if wins != 1 {
		t.Errorf("%d of %d racers claimed the message, want exactly 1", wins, racers)
	}
}

func TestClaimRefusedOnceDelivered(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	c := testConsumer(t, pool, 5*time.Minute)
	eventID := newEventID(t, pool)

	if _, ok, err := c.claim(ctx, eventID); err != nil || !ok {
		t.Fatalf("first claim: ok=%v err=%v", ok, err)
	}
	if err := c.markProcessed(ctx, eventID); err != nil {
		t.Fatalf("markProcessed: %v", err)
	}

	_, ok, err := c.claim(ctx, eventID)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if ok {
		t.Error("claimed an already-delivered event; it would be sent twice")
	}
}

func TestReleasedClaimIsImmediatelyReclaimable(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	c := testConsumer(t, pool, time.Hour)
	eventID := newEventID(t, pool)

	if _, ok, err := c.claim(ctx, eventID); err != nil || !ok {
		t.Fatalf("first claim: ok=%v err=%v", ok, err)
	}
	if _, ok, _ := c.claim(ctx, eventID); ok {
		t.Fatal("a live claim was stolen while still held")
	}

	if err := c.releaseClaim(ctx, eventID); err != nil {
		t.Fatalf("releaseClaim: %v", err)
	}

	attempts, ok, err := c.claim(ctx, eventID)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if !ok {
		t.Fatal("could not reclaim after release")
	}

	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestStaleClaimIsReclaimable(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	eventID := newEventID(t, pool)

	holder := testConsumer(t, pool, time.Hour)
	if _, ok, err := holder.claim(ctx, eventID); err != nil || !ok {
		t.Fatalf("first claim: ok=%v err=%v", ok, err)
	}

	reclaimer := testConsumer(t, pool, 0)

	if _, err := pool.Exec(ctx,
		`UPDATE processed_events SET claimed_at = now() - interval '10 minutes' WHERE event_id = $1`,
		eventID); err != nil {
		t.Fatalf("age claim: %v", err)
	}

	attempts, ok, err := reclaimer.claim(ctx, eventID)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if !ok {
		t.Fatal("a stale claim was not reclaimable; the message would be stuck")
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}
