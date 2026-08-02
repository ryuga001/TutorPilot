package worker

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const processedRetention = 30 * 24 * time.Hour

const purgeBatch = 5000

type Maintenance struct {
	db *pgxpool.Pool
}

func NewMaintenance(db *pgxpool.Pool) *Maintenance { return &Maintenance{db: db} }

func (m *Maintenance) Run(ctx context.Context, reportEvery, purgeEvery time.Duration) {
	report := time.NewTicker(reportEvery)
	defer report.Stop()
	purge := time.NewTicker(purgeEvery)
	defer purge.Stop()

	m.reportDLQ(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-report.C:
			m.reportDLQ(ctx)
		case <-purge.C:
			m.purgeProcessed(ctx)
		}
	}
}

func (m *Maintenance) reportDLQ(ctx context.Context) {
	var count int
	var oldest *time.Time
	err := m.db.QueryRow(ctx, `
		SELECT count(*), min(failed_at) FROM dead_events WHERE replayed_at IS NULL`,
	).Scan(&count, &oldest)
	if err != nil {
		log.Printf("worker: could not read dead-letter depth: %v", err)
		return
	}
	if count == 0 {
		return
	}
	since := "unknown"
	if oldest != nil {
		since = time.Since(*oldest).Round(time.Second).String()
	}
	log.Printf("worker: DLQ has %d unreplayed event(s), oldest %s ago -- inspect with: "+
		"SELECT event_type, last_error, attempts, failed_at FROM dead_events WHERE replayed_at IS NULL ORDER BY failed_at DESC",
		count, since)
}

func (m *Maintenance) purgeProcessed(ctx context.Context) {
	cutoff := time.Now().Add(-processedRetention)
	total := int64(0)
	for {
		tag, err := m.db.Exec(ctx, `
			DELETE FROM processed_events
			WHERE event_id IN (
			    SELECT event_id FROM processed_events
			    WHERE processed_at IS NOT NULL AND processed_at < $1
			    LIMIT $2
			)`, cutoff, purgeBatch)
		if err != nil {
			log.Printf("worker: purge of processed_events failed after %d rows: %v", total, err)
			return
		}
		n := tag.RowsAffected()
		total += n
		if n < purgeBatch {
			break
		}
		if ctx.Err() != nil {
			return
		}
	}
	if total > 0 {
		log.Printf("worker: purged %d processed_events older than %s", total, processedRetention)
	}
}
