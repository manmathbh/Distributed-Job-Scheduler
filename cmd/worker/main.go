package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/config"
	"github.com/manmathbh/distributed-job-scheduler/internal/db"
	"github.com/manmathbh/distributed-job-scheduler/internal/metrics"
	"github.com/manmathbh/distributed-job-scheduler/internal/service"
	"github.com/manmathbh/distributed-job-scheduler/internal/store"
	"github.com/manmathbh/distributed-job-scheduler/internal/worker"
)

// Standalone worker process: connects to PostgreSQL and processes jobs from
// all active queues, sending heartbeats and shutting down gracefully.
func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}
	defer pool.Close()

	st := store.NewPostgresStore(pool)
	reg := metrics.NewRegistry()
	svc := service.New(st, reg)

	wkr := worker.New(st, svc, reg, worker.Config{
		ID:                cfg.WorkerID,
		Concurrency:       cfg.WorkerConcurrency,
		PollInterval:      500 * time.Millisecond,
		LeaseDuration:     cfg.LeaseDuration,
		HeartbeatInterval: cfg.HeartbeatInterval,
		Logger:            log.Default(),
	})
	log.Printf("Worker %s starting (concurrency=%d)", wkr.ID(), cfg.WorkerConcurrency)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		log.Println("Shutdown signal received...")
		cancel()
	}()

	if err := wkr.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Worker error: %v", err)
	}
	log.Println("Worker stopped")
}
