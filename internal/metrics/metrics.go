package metrics

import (
	"sync/atomic"
	"time"
)

// Counters tracks scheduler throughput metrics using atomic counters.
type Counters struct {
	JobsSubmitted    atomic.Int64
	JobsCompleted    atomic.Int64
	JobsFailed       atomic.Int64
	JobsRetried      atomic.Int64
	JobsDeadLettered atomic.Int64
	ClaimsMade       atomic.Int64
	LeasesRecovered  atomic.Int64
	ExecutionsDone   atomic.Int64
	ExecutionMS      atomic.Int64 // cumulative execution milliseconds
}

// Registry is a lightweight metrics collection point.
type Registry struct {
	Counters Counters
	start    time.Time
}

// NewRegistry returns a zeroed registry.
func NewRegistry() *Registry {
	return &Registry{start: time.Now()}
}

func (r *Registry) IncSubmitted()       { r.Counters.JobsSubmitted.Add(1) }
func (r *Registry) IncCompleted()       { r.Counters.JobsCompleted.Add(1) }
func (r *Registry) IncFailed()          { r.Counters.JobsFailed.Add(1) }
func (r *Registry) IncRetried()         { r.Counters.JobsRetried.Add(1) }
func (r *Registry) IncDeadLettered()    { r.Counters.JobsDeadLettered.Add(1) }
func (r *Registry) IncClaims()          { r.Counters.ClaimsMade.Add(1) }
func (r *Registry) IncLeasesRecovered() { r.Counters.LeasesRecovered.Add(1) }
func (r *Registry) AddExecution(d time.Duration) {
	r.Counters.ExecutionsDone.Add(1)
	r.Counters.ExecutionMS.Add(d.Milliseconds())
}

// Snapshot captures a point-in-time metrics view.
type Snapshot struct {
	UptimeSeconds    int64   `json:"uptime_seconds"`
	JobsSubmitted    int64   `json:"jobs_submitted"`
	JobsCompleted    int64   `json:"jobs_completed"`
	JobsFailed       int64   `json:"jobs_failed"`
	JobsRetried      int64   `json:"jobs_retried"`
	JobsDeadLettered int64   `json:"jobs_dead_lettered"`
	ClaimsMade       int64   `json:"claims_made"`
	LeasesRecovered  int64   `json:"leases_recovered"`
	ExecutionsDone   int64   `json:"executions_done"`
	AvgExecutionMS   float64 `json:"avg_execution_ms"`
}

// Snapshot returns the current metrics.
func (r *Registry) Snapshot() Snapshot {
	done := r.Counters.ExecutionsDone.Load()
	avg := float64(0)
	if done > 0 {
		avg = float64(r.Counters.ExecutionMS.Load()) / float64(done)
	}
	return Snapshot{
		UptimeSeconds:    int64(time.Since(r.start).Seconds()),
		JobsSubmitted:    r.Counters.JobsSubmitted.Load(),
		JobsCompleted:    r.Counters.JobsCompleted.Load(),
		JobsFailed:       r.Counters.JobsFailed.Load(),
		JobsRetried:      r.Counters.JobsRetried.Load(),
		JobsDeadLettered: r.Counters.JobsDeadLettered.Load(),
		ClaimsMade:       r.Counters.ClaimsMade.Load(),
		LeasesRecovered:  r.Counters.LeasesRecovered.Load(),
		ExecutionsDone:   done,
		AvgExecutionMS:   avg,
	}
}
