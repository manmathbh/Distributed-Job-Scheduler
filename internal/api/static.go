package api

import (
	"net/http"

	"github.com/manmathbh/distributed-job-scheduler/internal/web"
)

// RegisterStatic serves the embedded dashboard under /dashboard.
func (s *Server) RegisterStatic(mux *http.ServeMux) {
	// Render/production landing page: send the root URL to the dashboard.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/dashboard/", http.StatusTemporaryRedirect)
	})

	mux.Handle("/dashboard/", http.StripPrefix("/dashboard", web.Handler()))
	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/dashboard/", http.StatusTemporaryRedirect)
	})
}
