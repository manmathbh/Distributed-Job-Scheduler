package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/api"
	"github.com/manmathbh/distributed-job-scheduler/internal/auth"
	"github.com/manmathbh/distributed-job-scheduler/internal/config"
	"github.com/manmathbh/distributed-job-scheduler/internal/db"
	"github.com/manmathbh/distributed-job-scheduler/internal/metrics"
	"github.com/manmathbh/distributed-job-scheduler/internal/queue"
	redisclient "github.com/manmathbh/distributed-job-scheduler/internal/redis"
	"github.com/manmathbh/distributed-job-scheduler/internal/scheduler"
	"github.com/manmathbh/distributed-job-scheduler/internal/seed"
	"github.com/manmathbh/distributed-job-scheduler/internal/service"
	"github.com/manmathbh/distributed-job-scheduler/internal/store"
	"github.com/manmathbh/distributed-job-scheduler/internal/wal"
	"github.com/manmathbh/distributed-job-scheduler/internal/worker"
)

func main() {
	cfg := config.Load()

	log.Printf("Starting distributed job scheduler...")
	log.Printf("  Port: %s", cfg.Port)
	log.Printf("  Database URL: %s", maskURL(cfg.DatabaseURL))
	log.Printf("  Redis URL: %s", cfg.RedisURL)
	log.Printf("  Store mode: %s", storeMode())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Redis (API keys) ---
	redisCfg := &redisclient.Config{
		URL:            cfg.RedisURL,
		MaxRetries:     3,
		PoolSize:       10,
		MinIdleConns:   2,
		ConnectTimeout: 5 * time.Second,
		ReadTimeout:    3 * time.Second,
		WriteTimeout:   3 * time.Second,
	}
	redisClient, err := redisclient.NewClient(redisCfg)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()
	log.Printf("Connected to Redis")

	authStore := auth.NewRedisStore(redisClient.GetClient())
	if err := auth.SeedDevKeys(ctx, authStore); err != nil {
		log.Fatalf("Failed to seed dev keys: %v", err)
	}

	// --- Legacy WAL-backed queue (backward compatible /jobs API) ---
	w, err := wal.Open(cfg.WALDir)
	if err != nil {
		log.Fatalf("Failed to open WAL: %v", err)
	}
	defer w.Close()
	core, err := queue.NewCore(w)
	if err != nil {
		log.Fatalf("Failed to create queue core: %v", err)
	}
	log.Printf("Legacy WAL queue recovered")

	// --- PostgreSQL authoritative store ---
	st, err := buildStore(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}

	// --- Metrics + service ---
	reg := metrics.NewRegistry()
	svc := service.New(st, reg)

	// --- Demo seed ---
	if os.Getenv("SEED_DEMO") != "false" {
		if err := seed.SeedDemo(ctx, svc, "dev-client"); err != nil {
			log.Printf("Warning: demo seed failed: %v", err)
		}
	}

	// --- HTTP API ---
	server := api.NewServer(core, cfg.LegacyLeaseDuration, authStore,
		api.WithService(svc), api.WithMetrics(reg))

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)
	server.RegisterStatic(mux)

	httpServer := &http.Server{Addr: ":" + cfg.Port, Handler: mux}

	// --- Legacy lease checker ---
	go func() {
		ticker := time.NewTicker(cfg.LegacyLeaseCheckInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				core.CheckExpiredLeases()
			case <-ctx.Done():
				return
			}
		}
	}()

	// --- Scheduler ---
	if cfg.SchedulerEnabled {
		sched := scheduler.New(st, reg, cfg.SchedulerInterval, log.Default(),
			scheduler.WithMaterializer(svc.MaterializeScheduledJob))
		go sched.Run(ctx)
		log.Printf("Scheduler started (interval=%v)", cfg.SchedulerInterval)
	}

	// --- In-process worker ---
	if cfg.WorkerEnabled {
		wkr := worker.New(st, svc, reg, worker.Config{
			ID:                cfg.WorkerID,
			Concurrency:       cfg.WorkerConcurrency,
			PollInterval:      500 * time.Millisecond,
			LeaseDuration:     cfg.LeaseDuration,
			HeartbeatInterval: cfg.HeartbeatInterval,
			Logger:            log.Default(),
		})
		go func() {
			if err := wkr.Run(ctx); err != nil && err != context.Canceled {
				log.Printf("Worker error: %v", err)
			}
		}()
		log.Printf("Worker %s started (concurrency=%d)", wkr.ID(), cfg.WorkerConcurrency)
	}

	// --- HTTP server ---
	go func() {
		log.Printf("Server listening on :%s", cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// --- Graceful shutdown ---
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}
	log.Println("Server stopped")
}

func buildStore(ctx context.Context, cfg config.Config) (store.Store, error) {
	if storeMode() == "memory" {
		log.Printf("Using in-memory store (STORE_MODE=memory)")
		return store.NewMemoryStore(), nil
	}
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	if err := db.Migrate(ctx, pool); err != nil {
		return nil, err
	}
	log.Printf("PostgreSQL connected and migrated")
	return store.NewPostgresStore(pool), nil
}

func storeMode() string {
	m := os.Getenv("STORE_MODE")
	if m == "" {
		return "postgres"
	}
	return m
}

func maskURL(u string) string {
	if len(u) <= 8 {
		return u
	}
	return u[:len(u)-8] + "******"
}
