package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/models"
	"github.com/manmathbh/distributed-job-scheduler/internal/service"
	"github.com/manmathbh/distributed-job-scheduler/internal/store"
)

type submitJobRequest struct {
	Type          models.JobType       `json:"type"`
	Payload       json.RawMessage      `json:"payload"`
	Priority      int                  `json:"priority"`
	ScheduledAt   *time.Time           `json:"scheduled_at"`
	DelayMS       int64                `json:"delay_ms"`
	MaxAttempts   int                  `json:"max_attempts"`
	RetryStrategy models.RetryStrategy `json:"retry_strategy"`
	CronExpr      string               `json:"cron_expr"`
	Timezone      string               `json:"timezone"`
	Name          string               `json:"name"`
}

func (r submitJobRequest) toService(projectID, queueID string) service.SubmitJobRequest {
	return service.SubmitJobRequest{
		ProjectID:     projectID,
		QueueID:       queueID,
		Type:          r.Type,
		Payload:       r.Payload,
		Priority:      r.Priority,
		ScheduledAt:   r.ScheduledAt,
		Delay:         time.Duration(r.DelayMS) * time.Millisecond,
		MaxAttempts:   r.MaxAttempts,
		RetryStrategy: r.RetryStrategy,
		CronExpr:      r.CronExpr,
		Timezone:      r.Timezone,
		ScheduleName:  r.Name,
	}
}

func (s *Server) handleSubmitJob(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	queueID := r.PathValue("queueID")
	if _, err := s.authorizeProject(r.Context(), projectID); err != nil {
		s.writeStoreError(w, err)
		return
	}
	var req submitJobRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	svcReq := req.toService(projectID, queueID)

	if svcReq.Type == models.JobTypeRecurring {
		sj, err := s.svc.CreateSchedule(r.Context(), svcReq)
		if err != nil {
			s.writeStoreError(w, err)
			return
		}
		s.writeJSON(w, http.StatusCreated, map[string]any{
			"type":             "recurring",
			"scheduled_job_id": sj.ID,
			"scheduled_job":    sj,
		})
		return
	}

	job, err := s.svc.SubmitJob(r.Context(), svcReq)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, job)
}

type batchSubmitRequest struct {
	Jobs []batchJobItem `json:"jobs"`
}

type batchJobItem struct {
	Type        models.JobType  `json:"type"`
	Payload     json.RawMessage `json:"payload"`
	Priority    int             `json:"priority"`
	DelayMS     int64           `json:"delay_ms"`
	ScheduledAt *time.Time      `json:"scheduled_at"`
}

func (s *Server) handleBatchSubmit(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	queueID := r.PathValue("queueID")
	if _, err := s.authorizeProject(r.Context(), projectID); err != nil {
		s.writeStoreError(w, err)
		return
	}
	var req batchSubmitRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	items := make([]service.BatchJob, 0, len(req.Jobs))
	for _, it := range req.Jobs {
		items = append(items, service.BatchJob{
			Type:        it.Type,
			Payload:     it.Payload,
			Priority:    it.Priority,
			Delay:       time.Duration(it.DelayMS) * time.Millisecond,
			ScheduledAt: it.ScheduledAt,
		})
	}
	jobs, err := s.svc.SubmitBatch(r.Context(), projectID, queueID, items)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]any{"count": len(jobs), "items": jobs})
}

func (s *Server) handleListJobs(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, err := s.authorizeProject(r.Context(), projectID); err != nil {
		s.writeStoreError(w, err)
		return
	}
	limit := queryInt(r, "limit", store.DefaultMaxPageSize)
	if limit > store.DefaultMaxPageSize {
		limit = store.DefaultMaxPageSize
	}
	page, err := s.svc.ListJobs(r.Context(), store.ListJobsFilter{
		ProjectID: projectID,
		QueueID:   queryParam(r, "queue_id"),
		Status:    queryParam(r, "status"),
		Type:      queryParam(r, "type"),
		Limit:     limit,
		Cursor:    queryParam(r, "cursor"),
	})
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": page.Items, "next_cursor": page.NextCursor})
}

func (s *Server) handleGetJob(w http.ResponseWriter, r *http.Request) {
	j, err := s.authorizeJob(r.Context(), r.PathValue("jobID"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, j)
}

func (s *Server) handleRetryJob(w http.ResponseWriter, r *http.Request) {
	j, err := s.authorizeJob(r.Context(), r.PathValue("jobID"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	retried, err := s.svc.RetryJob(r.Context(), j.ID)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, retried)
}

func (s *Server) handleCancelJob(w http.ResponseWriter, r *http.Request) {
	j, err := s.authorizeJob(r.Context(), r.PathValue("jobID"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	cancelled, err := s.svc.CancelJob(r.Context(), j.ID)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, cancelled)
}

func (s *Server) handleListExecutions(w http.ResponseWriter, r *http.Request) {
	j, err := s.authorizeJob(r.Context(), r.PathValue("jobID"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	execs, err := s.svc.Store().ListExecutions(r.Context(), j.ID)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": execs})
}

func (s *Server) handleListLogs(w http.ResponseWriter, r *http.Request) {
	j, err := s.authorizeJob(r.Context(), r.PathValue("jobID"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	logs, err := s.svc.Store().ListLogs(r.Context(), j.ID, queryInt(r, "limit", 200))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": logs})
}

// ---- schedules ----

func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, err := s.authorizeProject(r.Context(), projectID); err != nil {
		s.writeStoreError(w, err)
		return
	}
	var req submitJobRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	req.Type = models.JobTypeRecurring
	sj, err := s.svc.CreateSchedule(r.Context(), req.toService(projectID, queryParam(r, "queue_id")))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, sj)
}

func (s *Server) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, err := s.authorizeProject(r.Context(), projectID); err != nil {
		s.writeStoreError(w, err)
		return
	}
	items, err := s.svc.ListSchedules(r.Context(), projectID)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// ---- dead-letter queue ----

func (s *Server) handleListDLQ(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, err := s.authorizeProject(r.Context(), projectID); err != nil {
		s.writeStoreError(w, err)
		return
	}
	limit := queryInt(r, "limit", store.DefaultMaxPageSize)
	if limit > store.DefaultMaxPageSize {
		limit = store.DefaultMaxPageSize
	}
	items, cursor, err := s.svc.ListDLQ(r.Context(), store.ListDLQFilter{
		ProjectID: projectID,
		QueueID:   queryParam(r, "queue_id"),
		Limit:     limit,
		Cursor:    queryParam(r, "cursor"),
	})
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": items, "next_cursor": cursor})
}

func (s *Server) handleRequeueDLQ(w http.ResponseWriter, r *http.Request) {
	dlqID := r.PathValue("dlqID")
	entry, err := s.svc.Store().GetDLQEntry(r.Context(), dlqID)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	if _, err := s.authorizeProject(r.Context(), entry.ProjectID); err != nil {
		s.writeStoreError(w, err)
		return
	}
	job, err := s.svc.RequeueDLQ(r.Context(), dlqID)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"job": job})
}
