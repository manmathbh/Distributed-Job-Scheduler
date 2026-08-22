package scheduler

import (
	"context"
	"log"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/id"
	"github.com/manmathbh/distributed-job-scheduler/internal/metrics"
	"github.com/manmathbh/distributed-job-scheduler/internal/models"
	"github.com/manmathbh/distributed-job-scheduler/internal/store"
	"github.com/robfig/cron/v3"
)

// Scheduler periodically promotes due jobs, recovers expired leases, marks
// stale workers, and fires due recurring schedules. Firing is guarded by the
// store's atomic claim (FOR UPDATE SKIP LOCKED) so multiple scheduler
// instances never duplicate a scheduled job.
type Scheduler struct {
	store    store.Store
	metrics  *metrics.Registry
	interval time.Duration
	logger   *log.Logger
	cron     cron.Parser

	// materialize creates a queued job from a due recurring schedule. Wired by
	// the caller (e.g. service.MaterializeScheduledJob) so the job inherits the
	// queue's retry policy.
	materialize func(ctx context.Context, sj *models.ScheduledJob) (*models.Job, error)
}

// Option configures a Scheduler.
type Option func(*Scheduler)

// WithMaterializer sets the function used to materialize recurring schedules
// into jobs.
func WithMaterializer(fn func(ctx context.Context, sj *models.ScheduledJob) (*models.Job, error)) Option {
	return func(s *Scheduler) { s.materialize = fn }
}

// New constructs a Scheduler.
func New(st store.Store, m *metrics.Registry, interval time.Duration, logger *log.Logger, opts ...Option) *Scheduler {
	if logger == nil {
		logger = log.Default()
	}
	s := &Scheduler{
		store:    st,
		metrics:  m,
		interval: interval,
		logger:   logger,
		cron:     cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
	}
	s.materialize = s.defaultMaterialize
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Run loops until ctx is cancelled.
func (s *Scheduler) Run(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	s.promoteDue(ctx)
	s.recoverLeases(ctx)
	s.markStaleWorkers(ctx)
	s.fireSchedules(ctx)
}

func (s *Scheduler) promoteDue(ctx context.Context) {
	n, err := s.store.PromoteDueJobs(ctx, time.Now())
	if err != nil {
		s.logger.Printf("scheduler: promote due jobs: %v", err)
		return
	}
	if n > 0 {
		s.logger.Printf("scheduler: promoted %d scheduled job(s) to queued", n)
	}
}

func (s *Scheduler) recoverLeases(ctx context.Context) {
	n, err := s.store.RecoverExpiredLeases(ctx, time.Now())
	if err != nil {
		s.logger.Printf("scheduler: recover leases: %v", err)
		return
	}
	if n > 0 {
		s.metrics.Counters.LeasesRecovered.Add(int64(n))
		s.logger.Printf("scheduler: recovered %d expired lease(s)", n)
	}
}

func (s *Scheduler) markStaleWorkers(ctx context.Context) {
	n, err := s.store.MarkStaleWorkers(ctx, 60*time.Second, time.Now())
	if err != nil {
		s.logger.Printf("scheduler: mark stale workers: %v", err)
		return
	}
	if n > 0 {
		s.logger.Printf("scheduler: marked %d stale worker(s) dead", n)
	}
}

func (s *Scheduler) fireSchedules(ctx context.Context) {
	for {
		now := time.Now()
		sj, err := s.store.FireDueScheduledJob(ctx, now, s.nextRun)
		if err != nil {
			s.logger.Printf("scheduler: fire schedule: %v", err)
			return
		}
		if sj == nil {
			return
		}

		job, err := s.materialize(ctx, sj)
		if err != nil {
			s.logger.Printf("scheduler: materialize scheduled job: %v", err)
			continue
		}
		s.logger.Printf("scheduler: fired schedule %q -> job %s", sj.Name, job.ID)
	}
}

// defaultMaterialize is the fallback used when no materializer is wired. It
// builds a bare recurring job without a retry-policy snapshot; production
// wiring should use WithMaterializer(service.MaterializeScheduledJob).
func (s *Scheduler) defaultMaterialize(ctx context.Context, sj *models.ScheduledJob) (*models.Job, error) {
	now := time.Now()
	job := &models.Job{
		ID:          id.New(),
		ProjectID:   sj.ProjectID,
		QueueID:     sj.QueueID,
		Type:        models.JobTypeRecurring,
		Payload:     sj.Payload,
		Priority:    sj.Priority,
		Status:      models.JobStatusQueued,
		AvailableAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.CreateJob(ctx, job); err != nil {
		return nil, err
	}
	s.metrics.IncSubmitted()
	return job, nil
}

func (s *Scheduler) nextRun(sj *models.ScheduledJob) (time.Time, error) {
	loc, err := time.LoadLocation(sj.Timezone)
	if err != nil {
		loc = time.UTC
	}
	sched, err := s.cron.Parse(sj.CronExpr)
	if err != nil {
		return time.Time{}, err
	}
	return sched.Next(time.Now().In(loc)), nil
}
