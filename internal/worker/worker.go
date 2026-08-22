package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/id"
	"github.com/manmathbh/distributed-job-scheduler/internal/metrics"
	"github.com/manmathbh/distributed-job-scheduler/internal/models"
	"github.com/manmathbh/distributed-job-scheduler/internal/service"
	"github.com/manmathbh/distributed-job-scheduler/internal/store"
)

// Handler executes a single claimed job. It should be idempotent; delivery is
// at-least-once and a job may be re-executed after a lease expiry.
type Handler interface {
	Handle(ctx context.Context, job *models.Job) (json.RawMessage, error)
}

// HandlerFunc adapts a function to the Handler interface.
type HandlerFunc func(ctx context.Context, job *models.Job) (json.RawMessage, error)

func (f HandlerFunc) Handle(ctx context.Context, job *models.Job) (json.RawMessage, error) {
	return f(ctx, job)
}

// EchoHandler returns the job payload as the result, recording a log line.
func EchoHandler(logger *log.Logger) HandlerFunc {
	return func(ctx context.Context, job *models.Job) (json.RawMessage, error) {
		if logger != nil {
			logger.Printf("worker: executing job %s (type=%s queue=%s attempt=%d)", job.ID, job.Type, job.QueueID, job.Attempts)
		}
		return job.Payload, nil
	}
}

// Config configures a Worker.
type Config struct {
	ID                string
	ProjectID         string
	Concurrency       int
	PollInterval      time.Duration
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
	Handler           Handler
	Logger            *log.Logger
}

// Worker polls queues, atomically claims jobs, executes them concurrently,
// sends heartbeats, and shuts down gracefully.
type Worker struct {
	cfg     Config
	store   store.Store
	service *service.Service
	metrics *metrics.Registry
	logger  *log.Logger

	running atomic.Int64
	mu      sync.Mutex
	claimed map[string]struct{}
}

// New constructs a Worker with sensible defaults.
func New(st store.Store, svc *service.Service, m *metrics.Registry, cfg Config) *Worker {
	if cfg.ID == "" {
		host, _ := os.Hostname()
		cfg.ID = fmt.Sprintf("worker-%s-%s", host, id.New()[:8])
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 4
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 500 * time.Millisecond
	}
	if cfg.LeaseDuration <= 0 {
		cfg.LeaseDuration = 30 * time.Second
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 10 * time.Second
	}
	if cfg.Handler == nil {
		cfg.Handler = EchoHandler(cfg.Logger)
	}
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	return &Worker{
		cfg:     cfg,
		store:   st,
		service: svc,
		metrics: m,
		logger:  cfg.Logger,
		claimed: make(map[string]struct{}),
	}
}

// ID returns the worker identifier.
func (w *Worker) ID() string { return w.cfg.ID }

// Run starts the worker and blocks until ctx is cancelled. It registers the
// worker, sends heartbeats, and processes jobs concurrently.
func (w *Worker) Run(ctx context.Context) error {
	if err := w.register(ctx); err != nil {
		return fmt.Errorf("register worker: %w", err)
	}
	w.logger.Printf("worker %s started (concurrency=%d)", w.cfg.ID, w.cfg.Concurrency)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	sem := make(chan struct{}, w.cfg.Concurrency)

	heartbeatDone := make(chan struct{})
	go func() {
		defer close(heartbeatDone)
		w.heartbeatLoop(ctx)
	}()

	poll := time.NewTicker(w.cfg.PollInterval)
	defer poll.Stop()

	for {
		select {
		case <-ctx.Done():
			// Stop claiming; wait for in-flight jobs (bounded).
			done := make(chan struct{})
			go func() { wg.Wait(); close(done) }()
			select {
			case <-done:
			case <-time.After(30 * time.Second):
				w.logger.Printf("worker %s: timed out waiting for in-flight jobs", w.cfg.ID)
			}
			<-heartbeatDone
			w.logger.Printf("worker %s stopped", w.cfg.ID)
			return ctx.Err()
		case <-poll.C:
			select {
			case sem <- struct{}{}:
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() { <-sem }()
					w.processOne(ctx)
				}()
			default:
				// Concurrency limit reached.
			}
		}
	}
}

func (w *Worker) register(ctx context.Context) error {
	now := time.Now()
	worker := &models.Worker{
		ID:            w.cfg.ID,
		ProjectID:     w.cfg.ProjectID,
		Hostname:      hostname(),
		Status:        models.WorkerStatusActive,
		Concurrency:   w.cfg.Concurrency,
		LastHeartbeat: now,
		StartedAt:     now,
		LastSeenAt:    now,
	}
	return w.store.RegisterWorker(ctx, worker)
}

func (w *Worker) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(w.cfg.HeartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.store.Heartbeat(ctx, w.cfg.ID, int(w.running.Load()), time.Now()); err != nil {
				w.logger.Printf("worker %s: heartbeat failed: %v", w.cfg.ID, err)
			}
		}
	}
}

func (w *Worker) processOne(ctx context.Context) {
	queues, err := w.store.ListQueues(ctx, w.cfg.ProjectID)
	if err != nil {
		return
	}
	for _, q := range queues {
		if q.Status != models.QueueStatusActive {
			continue
		}
		claimToken := id.New()
		job, err := w.store.ClaimJob(ctx, q.ID, w.cfg.ID, claimToken, w.cfg.LeaseDuration)
		if err != nil {
			if err == store.ErrNoJobs || err == store.ErrNotFound {
				continue
			}
			w.logger.Printf("worker %s: claim from queue %s: %v", w.cfg.ID, q.ID, err)
			return
		}
		w.metrics.IncClaims()
		w.execute(ctx, job)
		return
	}
}

func (w *Worker) execute(ctx context.Context, job *models.Job) {
	w.running.Add(1)
	defer w.running.Add(-1)

	w.appendLog(ctx, job, "info", fmt.Sprintf("claimed by worker %s (attempt %d)", w.cfg.ID, job.Attempts))

	exec, err := w.service.RecordExecutionStart(ctx, job)
	if err != nil {
		w.logger.Printf("worker %s: start job %s: %v", w.cfg.ID, job.ID, err)
		return
	}
	started := time.Now()
	result, handleErr := w.cfg.Handler.Handle(ctx, job)

	if handleErr != nil {
		w.appendLog(ctx, job, "error", fmt.Sprintf("attempt %d failed: %v", job.Attempts, handleErr))
		if err := w.service.FailJob(ctx, job, exec, handleErr.Error()); err != nil {
			w.logger.Printf("worker %s: fail job %s: %v", w.cfg.ID, job.ID, err)
		}
		return
	}

	w.appendLog(ctx, job, "info", fmt.Sprintf("attempt %d completed in %s", job.Attempts, time.Since(started)))
	if err := w.service.CompleteJob(ctx, job, exec, result); err != nil {
		w.logger.Printf("worker %s: complete job %s: %v", w.cfg.ID, job.ID, err)
	}
}

func (w *Worker) appendLog(ctx context.Context, job *models.Job, level, msg string) {
	_ = w.store.AppendLog(ctx, &models.JobLog{
		ID:        id.New(),
		JobID:     job.ID,
		Attempt:   job.Attempts,
		Level:     level,
		Message:   msg,
		CreatedAt: time.Now(),
	})
}

func hostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}
