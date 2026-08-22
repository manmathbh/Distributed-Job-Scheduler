package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/manmathbh/distributed-job-scheduler/internal/auth"
	"github.com/manmathbh/distributed-job-scheduler/internal/models"
)

// registerV1Routes wires the PostgreSQL-backed scheduler API under /api/v1.
func (s *Server) registerV1Routes(mux *http.ServeMux, authMW func(http.Handler) http.Handler) {
	authed := func(h http.HandlerFunc) http.Handler { return authMW(h) }

	// Projects
	mux.Handle("POST /api/v1/projects", authed(s.handleCreateProject))
	mux.Handle("GET /api/v1/projects", authed(s.handleListProjects))
	mux.Handle("GET /api/v1/projects/{projectID}", authed(s.handleGetProject))
	mux.Handle("PATCH /api/v1/projects/{projectID}", authed(s.handleUpdateProject))
	mux.Handle("DELETE /api/v1/projects/{projectID}", authed(s.handleDeleteProject))

	// Queues
	mux.Handle("POST /api/v1/projects/{projectID}/queues", authed(s.handleCreateQueue))
	mux.Handle("GET /api/v1/projects/{projectID}/queues", authed(s.handleListQueues))
	mux.Handle("GET /api/v1/queues/{queueID}", authed(s.handleGetQueue))
	mux.Handle("PATCH /api/v1/queues/{queueID}", authed(s.handleUpdateQueue))
	mux.Handle("POST /api/v1/queues/{queueID}/pause", authed(s.handlePauseQueue))
	mux.Handle("POST /api/v1/queues/{queueID}/resume", authed(s.handleResumeQueue))
	mux.Handle("DELETE /api/v1/queues/{queueID}", authed(s.handleDeleteQueue))
	mux.Handle("GET /api/v1/queues/{queueID}/stats", authed(s.handleQueueStats))

	// Jobs
	mux.Handle("POST /api/v1/projects/{projectID}/queues/{queueID}/jobs", authed(s.handleSubmitJob))
	mux.Handle("POST /api/v1/projects/{projectID}/queues/{queueID}/jobs/batch", authed(s.handleBatchSubmit))
	mux.Handle("GET /api/v1/projects/{projectID}/jobs", authed(s.handleListJobs))
	mux.Handle("GET /api/v1/jobs/{jobID}", authed(s.handleGetJob))
	mux.Handle("POST /api/v1/jobs/{jobID}/retry", authed(s.handleRetryJob))
	mux.Handle("POST /api/v1/jobs/{jobID}/cancel", authed(s.handleCancelJob))
	mux.Handle("GET /api/v1/jobs/{jobID}/executions", authed(s.handleListExecutions))
	mux.Handle("GET /api/v1/jobs/{jobID}/logs", authed(s.handleListLogs))

	// Schedules (recurring)
	mux.Handle("POST /api/v1/projects/{projectID}/schedules", authed(s.handleCreateSchedule))
	mux.Handle("GET /api/v1/projects/{projectID}/schedules", authed(s.handleListSchedules))

	// Dead-letter queue
	mux.Handle("GET /api/v1/projects/{projectID}/dead-letter", authed(s.handleListDLQ))
	mux.Handle("POST /api/v1/dead-letter/{dlqID}/requeue", authed(s.handleRequeueDLQ))

	// Workers
	mux.Handle("GET /api/v1/workers", authed(s.handleListWorkers))
	mux.Handle("GET /api/v1/workers/{workerID}", authed(s.handleGetWorker))
	mux.Handle("POST /api/v1/workers/{workerID}/heartbeat", authed(s.handleWorkerHeartbeat))

	// Metrics / overview
	mux.Handle("GET /api/v1/metrics", authed(s.handleMetrics))
	mux.Handle("GET /api/v1/overview", authed(s.handleOverview))
}

// ---- authorization helpers ----

// currentOwner returns the authenticated key's owner id.
func currentOwner(ctx context.Context) (string, bool) {
	return GetOwnerID(ctx)
}

// isAdmin reports whether the authenticated key is an admin key.
func isAdmin(ctx context.Context) bool {
	kt, ok := GetKeyType(ctx)
	return ok && kt == auth.KeyTypeAdmin
}

// authorizeProject loads a project and verifies the caller may access it.
func (s *Server) authorizeProject(ctx context.Context, projectID string) (*models.Project, error) {
	p, err := s.svc.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if isAdmin(ctx) {
		return p, nil
	}
	owner, _ := currentOwner(ctx)
	if p.OwnerID != owner {
		return nil, errForbidden
	}
	return p, nil
}

// authorizeJob verifies the caller may access the job's project.
func (s *Server) authorizeJob(ctx context.Context, jobID string) (*models.Job, error) {
	j, err := s.svc.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if isAdmin(ctx) {
		return j, nil
	}
	owner, _ := currentOwner(ctx)
	p, err := s.svc.GetProject(ctx, j.ProjectID)
	if err != nil {
		return nil, err
	}
	if p.OwnerID != owner {
		return nil, errForbidden
	}
	return j, nil
}

// ---- request helpers ----

type errorBody struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details any    `json:"details,omitempty"`
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) writeErr(w http.ResponseWriter, status int, code, msg string) {
	s.writeJSON(w, status, errorBody{Error: msg, Code: code})
}

func (s *Server) decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		s.writeErr(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return false
	}
	return true
}

func queryParam(r *http.Request, key string) string {
	return r.URL.Query().Get(key)
}

func queryInt(r *http.Request, key string, def int) int {
	v := queryParam(r, key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
