package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/*
var staticFS embed.FS

// FS returns the embedded static filesystem for the dashboard.
func FS() fs.FS {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	return sub
}

// Handler serves the dashboard SPA.
func Handler() http.Handler {
	return http.FileServer(http.FS(FS()))
}
