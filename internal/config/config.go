package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all runtime configuration for the scheduler.
type Config struct {
	Port string

	// PostgreSQL
	DatabaseURL string

	// Redis
	RedisURL string

	// WAL (legacy queue)
	WALDir string

	// Lease / heartbeat
	LeaseDuration      time.Duration
	LeaseCheckInterval time.Duration
	HeartbeatInterval  time.Duration
	WorkerStaleAfter   time.Duration

	// Scheduler
	SchedulerInterval time.Duration
	SchedulerEnabled  bool

	// Worker (in-process)
	WorkerEnabled     bool
	WorkerID          string
	WorkerConcurrency int

	// Legacy queue lease (backward compat)
	LegacyLeaseDuration      time.Duration
	LegacyLeaseCheckInterval time.Duration
}

// Load reads configuration from the environment, applying defaults.
func Load() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/jobscheduler?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		WALDir:      getEnv("WAL_DIR", "./data"),

		LeaseDuration:      getEnvDuration("LEASE_DURATION", 30*time.Second),
		LeaseCheckInterval: getEnvDuration("LEASE_CHECK_INTERVAL", 5*time.Second),
		HeartbeatInterval:  getEnvDuration("HEARTBEAT_INTERVAL", 10*time.Second),
		WorkerStaleAfter:   getEnvDuration("WORKER_STALE_AFTER", 60*time.Second),

		SchedulerInterval: getEnvDuration("SCHEDULER_INTERVAL", 2*time.Second),
		SchedulerEnabled:  getEnvBool("SCHEDULER_ENABLED", true),

		WorkerEnabled:     getEnvBool("WORKER_ENABLED", true),
		WorkerID:          getEnv("WORKER_ID", ""),
		WorkerConcurrency: getEnvInt("WORKER_CONCURRENCY", 4),

		LegacyLeaseDuration:      getEnvDuration("LEASE_DURATION", 30*time.Second),
		LegacyLeaseCheckInterval: getEnvDuration("LEASE_CHECK_INTERVAL", 5*time.Second),
	}
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}
