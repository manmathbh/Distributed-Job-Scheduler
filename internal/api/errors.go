package api

import (
	"errors"
	"net/http"

	"github.com/manmathbh/distributed-job-scheduler/internal/store"
)

var (
	errForbidden = errors.New("access denied")
	errNotFound  = errors.New("resource not found")
)

// writeStoreError maps a store/service error to an HTTP response.
func (s *Server) writeStoreError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errForbidden):
		s.writeErr(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, store.ErrNotFound), errors.Is(err, errNotFound):
		s.writeErr(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, store.ErrConflict):
		s.writeErr(w, http.StatusConflict, "conflict", err.Error())
	case errors.Is(err, store.ErrNoJobs):
		s.writeErr(w, http.StatusNotFound, "no_jobs", "no jobs available")
	default:
		s.writeErr(w, http.StatusBadRequest, "bad_request", err.Error())
	}
}
