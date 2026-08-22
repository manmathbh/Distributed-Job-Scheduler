package api

import (
	"net/http"
	"time"

	"github.com/manmathbh/distributed-job-scheduler/internal/models"
	"github.com/manmathbh/distributed-job-scheduler/internal/service"
)

type queueRequest struct {
	Name           string               `json:"name"`
	Description    string               `json:"description"`
	Priority       int                  `json:"priority"`
	Concurrency    int                  `json:"concurrency"`
	RetryStrategy  models.RetryStrategy `json:"retry_strategy"`
	MaxAttempts    int                  `json:"max_attempts"`
	InitialDelayMS int64                `json:"initial_delay_ms"`
	MaxDelayMS     int64                `json:"max_delay_ms"`
	Multiplier     float64              `json:"multiplier"`
}

func (q queueRequest) toConfig() service.QueueConfig {
	return service.QueueConfig{
		Name:          q.Name,
		Description:   q.Description,
		Priority:      q.Priority,
		Concurrency:   q.Concurrency,
		RetryStrategy: q.RetryStrategy,
		MaxAttempts:   q.MaxAttempts,
		InitialDelay:  time.Duration(q.InitialDelayMS) * time.Millisecond,
		MaxDelay:      time.Duration(q.MaxDelayMS) * time.Millisecond,
		Multiplier:    q.Multiplier,
	}
}

func (s *Server) handleCreateQueue(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, err := s.authorizeProject(r.Context(), projectID); err != nil {
		s.writeStoreError(w, err)
		return
	}
	var req queueRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	q, err := s.svc.CreateQueue(r.Context(), projectID, req.toConfig())
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, q)
}

func (s *Server) handleListQueues(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	if _, err := s.authorizeProject(r.Context(), projectID); err != nil {
		s.writeStoreError(w, err)
		return
	}
	queues, err := s.svc.ListQueues(r.Context(), projectID)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": queues})
}

func (s *Server) handleGetQueue(w http.ResponseWriter, r *http.Request) {
	q, err := s.svc.GetQueue(r.Context(), r.PathValue("queueID"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	if _, err := s.authorizeProject(r.Context(), q.ProjectID); err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, q)
}

func (s *Server) handleUpdateQueue(w http.ResponseWriter, r *http.Request) {
	q, err := s.svc.GetQueue(r.Context(), r.PathValue("queueID"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	if _, err := s.authorizeProject(r.Context(), q.ProjectID); err != nil {
		s.writeStoreError(w, err)
		return
	}
	var req queueRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	updated, err := s.svc.UpdateQueue(r.Context(), q.ID, req.toConfig())
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handlePauseQueue(w http.ResponseWriter, r *http.Request) {
	s.setQueueStatus(w, r, models.QueueStatusPaused)
}

func (s *Server) handleResumeQueue(w http.ResponseWriter, r *http.Request) {
	s.setQueueStatus(w, r, models.QueueStatusActive)
}

func (s *Server) setQueueStatus(w http.ResponseWriter, r *http.Request, status models.QueueStatus) {
	q, err := s.svc.GetQueue(r.Context(), r.PathValue("queueID"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	if _, err := s.authorizeProject(r.Context(), q.ProjectID); err != nil {
		s.writeStoreError(w, err)
		return
	}
	updated, err := s.svc.SetQueueStatus(r.Context(), q.ID, status)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, updated)
}

func (s *Server) handleDeleteQueue(w http.ResponseWriter, r *http.Request) {
	q, err := s.svc.GetQueue(r.Context(), r.PathValue("queueID"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	if _, err := s.authorizeProject(r.Context(), q.ProjectID); err != nil {
		s.writeStoreError(w, err)
		return
	}
	if err := s.svc.DeleteQueue(r.Context(), q.ID); err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleQueueStats(w http.ResponseWriter, r *http.Request) {
	q, err := s.svc.GetQueue(r.Context(), r.PathValue("queueID"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	if _, err := s.authorizeProject(r.Context(), q.ProjectID); err != nil {
		s.writeStoreError(w, err)
		return
	}
	stats, err := s.svc.QueueStats(r.Context(), q.ID)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, stats)
}
