package api

import (
	"net/http"
	"strings"

	"github.com/manmathbh/distributed-job-scheduler/internal/store"
)

type createProjectRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func (s *Server) handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	owner, _ := currentOwner(r.Context())
	p, err := s.svc.CreateProject(r.Context(), owner, req.Name, req.Description)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, p)
}

func (s *Server) handleListProjects(w http.ResponseWriter, r *http.Request) {
	owner := ""
	if !isAdmin(r.Context()) {
		owner, _ = currentOwner(r.Context())
	}
	limit := queryInt(r, "limit", store.DefaultMaxPageSize)
	if limit > store.DefaultMaxPageSize {
		limit = store.DefaultMaxPageSize
	}
	items, cursor, err := s.svc.ListProjects(r.Context(), owner, limit, queryParam(r, "cursor"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": items, "next_cursor": cursor})
}

func (s *Server) handleGetProject(w http.ResponseWriter, r *http.Request) {
	p, err := s.authorizeProject(r.Context(), r.PathValue("projectID"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleUpdateProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("projectID")
	if _, err := s.authorizeProject(r.Context(), id); err != nil {
		s.writeStoreError(w, err)
		return
	}
	var req createProjectRequest
	if !s.decodeJSON(w, r, &req) {
		return
	}
	p, err := s.svc.UpdateProject(r.Context(), id, req.Name, req.Description)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, p)
}

func (s *Server) handleDeleteProject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("projectID")
	if _, err := s.authorizeProject(r.Context(), id); err != nil {
		s.writeStoreError(w, err)
		return
	}
	if err := s.svc.DeleteProject(r.Context(), id); err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func trimSpace(s string) string { return strings.TrimSpace(s) }
