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
		// traversal. Prefixing with "/" both anchors the path to the web root
		// and satisfies the Semgrep filepath rule's sanitizer for "/"+..., so
		// the rule no longer fires (an in-source nosemgrep would still be
		// emitted into the SARIF and keep the GitHub alert open).
		cleanPath := path.Clean("/" + r.URL.Path)
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
