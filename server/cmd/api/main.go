package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
	"tutorpilot/internal/modules/admin/livekit"

	"tutorpilot/internal/config"
	"tutorpilot/internal/pkg/database"
	"tutorpilot/internal/pkg/outbox"
	"tutorpilot/internal/pkg/redisclient"
	"tutorpilot/internal/server"
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

	lkt := livekit.New(livekit.Options{
		URL:             cfg.LiveKitURL,
		APIKey:          cfg.LiveKitKey,
		APISecret:       cfg.LiveKitSecret,
		EmptyTimeout:    cfg.LiveKitRoomEmptyTimeout,
		MaxParticipants: cfg.LiveKitMaxParticipants,
	})

	r := server.New(cfg, db, rdb, lkt)

	relayCtx, stopRelay := context.WithCancel(context.Background())
	var relayWG sync.WaitGroup
	if cfg.RelayEnabled {
		relay := outbox.NewRelay(db, rdb, outbox.RelayConfig{
			PollInterval: cfg.OutboxPollInterval,
			BatchSize:    cfg.OutboxBatchSize,
			Retention: map[string]time.Duration{
				cfg.EventStreamNotifications: cfg.EventRetentionNotifications,
				cfg.EventStreamAuth:          cfg.EventRetentionAuth,
			},
		})
		relayWG.Add(1)
		go func() {
			defer relayWG.Done()
			relay.Run(relayCtx)
		}()
	} else {
		log.Println("outbox relay: disabled (RELAY_ENABLED=false)")
	}

	srv := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("tutorpilot api listening on :%s (env=%s)", cfg.AppPort, cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("forced shutdown: %v", err)
	}

	stopRelay()
	relayWG.Wait()

	log.Println("server stopped")
}
