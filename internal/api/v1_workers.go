package api

import (
	"net/http"
)

func (s *Server) handleListWorkers(w http.ResponseWriter, r *http.Request) {
	projectID := queryParam(r, "project_id")
	if !isAdmin(r.Context()) {
		// Non-admins see workers for their own projects only.
		owner, _ := currentOwner(r.Context())
		if projectID == "" {
			// list workers across all projects owned by the caller
			items, err := s.svc.ListWorkers(r.Context(), "")
			if err != nil {
				s.writeStoreError(w, err)
				return
			}
			filtered := make([]any, 0)
			for _, w := range items {
				if p, err := s.svc.GetProject(r.Context(), w.ProjectID); err == nil && p.OwnerID == owner {
					filtered = append(filtered, w)
				}
			}
			s.writeJSON(w, http.StatusOK, map[string]any{"items": filtered})
			return
		}
		if _, err := s.authorizeProject(r.Context(), projectID); err != nil {
			s.writeStoreError(w, err)
			return
		}
	}
	items, err := s.svc.ListWorkers(r.Context(), projectID)
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Server) handleGetWorker(w http.ResponseWriter, r *http.Request) {
	wkr, err := s.svc.Store().GetWorker(r.Context(), r.PathValue("workerID"))
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	if !isAdmin(r.Context()) && wkr.ProjectID != "" {
		if _, err := s.authorizeProject(r.Context(), wkr.ProjectID); err != nil {
			s.writeStoreError(w, err)
			return
		}
	}
	s.writeJSON(w, http.StatusOK, wkr)
}

func (s *Server) handleWorkerHeartbeat(w http.ResponseWriter, r *http.Request) {
	workerID := r.PathValue("workerID")
	var req struct {
		Running int `json:"running"`
	}
	_ = s.decodeJSON(w, r, &req)
	if err := s.svc.Heartbeat(r.Context(), workerID, req.Running); err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	if s.metrics == nil {
		s.writeJSON(w, http.StatusOK, map[string]any{})
		return
	}
	s.writeJSON(w, http.StatusOK, s.metrics.Snapshot())
}

func (s *Server) handleOverview(w http.ResponseWriter, r *http.Request) {
	ov, err := s.svc.Overview(r.Context())
	if err != nil {
		s.writeStoreError(w, err)
		return
	}
	s.writeJSON(w, http.StatusOK, ov)
}
