package main

import (
	"net/http"
	"path"
	"path/filepath"
	"strings"
)

// spaFileHandler serves static files from distDir and falls back to index.html
// for client-side routes such as /login and /register. Paths under /api are
// never rewritten to the SPA, and missing files with an extension stay 404.
func spaFileHandler(distDir string) http.HandlerFunc {
	fileServer := http.FileServer(http.Dir(distDir))
	indexPath := filepath.Join(distDir, "index.html")

	return func(w http.ResponseWriter, r *http.Request) {
		cleanPath := path.Clean(r.URL.Path)
		if cleanPath == "/api" || strings.HasPrefix(cleanPath, "/api/") {
			http.NotFound(w, r)
			return
		}

		if cleanPath == "/" || path.Ext(cleanPath) == "" {
			http.ServeFile(w, r, indexPath)
			return
		}

		fileServer.ServeHTTP(w, r)
	}
}
