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
		// URL path, not a filesystem path: path.Clean is used only for routing
		// checks; file reads go through http.FileServer/http.Dir, which reject
		// traversal. The Semgrep filepath rule does not apply to URL paths.
		cleanPath := path.Clean(r.URL.Path) // nosemgrep: go.lang.security.filepath-clean-misuse
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
