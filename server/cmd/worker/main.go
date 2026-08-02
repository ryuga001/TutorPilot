package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"tutorpilot/internal/config"
	"tutorpilot/internal/modules/notification/mailer"
	notificationrepo "tutorpilot/internal/modules/notification/repository"
	notification "tutorpilot/internal/modules/notification/service"
	"tutorpilot/internal/modules/notification/worker"
	"tutorpilot/internal/pkg/database"
	"tutorpilot/internal/pkg/redisclient"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	rdb, err := redisclient.New(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer rdb.Close()

	mail := mailer.New(mailer.Config{
		Host:    cfg.SMTPHost,
		Port:    cfg.SMTPPort,
		From:    cfg.SMTPFrom,
		Timeout: cfg.SMTPTimeout,
	})
	notifier := notification.New(mail, notificationrepo.NewTemplateStore(db), notification.Config{
		VerifyURL:      cfg.AppVerifyURL,
		SignInURL:      cfg.AppSignInURL,
		OTPTTL:         cfg.OTPTTL,
		SystemTenantID: notificationrepo.SystemTenantID,
	})

	streams := []string{cfg.EventStreamNotifications, cfg.EventStreamAuth}
	consumer := worker.New(db, rdb, notifier, worker.Config{
		Streams:       streams,
		ConsumerGroup: cfg.EventConsumerGroup,

		ConsumerName: consumerName(),
		Concurrency:  cfg.WorkerConcurrency,
		MaxAttempts:  cfg.WorkerMaxAttempts,
		ClaimMinIdle: cfg.WorkerClaimMinIdle,
		BlockTimeout: 5 * time.Second,
	})

	if err := consumer.EnsureGroups(ctx); err != nil {
		log.Fatalf("worker: could not create consumer groups (has `make migrate-up` run?): %v", err)
	}

	runCtx, stop := context.WithCancel(ctx)
	var wg sync.WaitGroup

	for _, s := range streams {
		wg.Add(1)
		go func(stream string) {
			defer wg.Done()
			consumer.RunStream(runCtx, stream)
		}(s)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.NewMaintenance(db).Run(runCtx, 30*time.Second, time.Hour)
	}()

	srv := readinessServer(cfg, db, rdb)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("worker: readiness server: %v", err)
		}
	}()

	log.Printf("tutorpilot worker started (streams=%v concurrency=%d)", streams, cfg.WorkerConcurrency)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("worker: shutting down...")

	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)

	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
		log.Println("worker: stopped cleanly")
	case <-shutdownCtx.Done():
		log.Println("worker: shutdown timed out with work in flight")
	}
}

func consumerName() string {
	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}
	return fmt.Sprintf("%s-%d", host, os.Getpid())
}

func readinessServer(cfg *config.Config, db *pgxpool.Pool, rdb *redis.Client) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			http.Error(w, "database unreachable", http.StatusServiceUnavailable)
			return
		}
		if err := rdb.Ping(ctx).Err(); err != nil {
			http.Error(w, "redis unreachable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	return &http.Server{
		Addr:              ":" + cfg.WorkerPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
}
