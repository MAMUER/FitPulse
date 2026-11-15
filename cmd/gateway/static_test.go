package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSPAFileHandler(t *testing.T) {
	distDir := t.TempDir()
	indexBody := []byte("<html><body><div id=\"app\"></div></body></html>")
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "index.html"), indexBody, 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "app.js"), []byte("console.log('ok')"), 0o600))

	handler := spaFileHandler(distDir)

	tests := []struct {
		name        string
		path        string
		wantStatus  int
		wantBody    string
		wantIndex   bool
		notFoundAPI bool
	}{
		{name: "root", path: "/", wantStatus: http.StatusOK, wantIndex: true},
		{name: "login route", path: "/login", wantStatus: http.StatusOK, wantIndex: true},
		{name: "nested route", path: "/legal/privacy", wantStatus: http.StatusOK, wantIndex: true},
		{name: "existing asset", path: "/app.js", wantStatus: http.StatusOK, wantBody: "console.log('ok')"},
		{name: "missing asset", path: "/missing.js", wantStatus: http.StatusNotFound},
		{name: "unknown api", path: "/api/v1/unknown", wantStatus: http.StatusNotFound, notFoundAPI: true},
		{name: "api root", path: "/api", wantStatus: http.StatusNotFound, notFoundAPI: true},
		{name: "double slash route", path: "//login", wantStatus: http.StatusOK, wantIndex: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			assert.Equal(t, tt.wantStatus, rec.Code)
			if tt.wantIndex {
				assert.Equal(t, string(indexBody), rec.Body.String())
			}
			if tt.wantBody != "" {
				assert.Equal(t, tt.wantBody, rec.Body.String())
			}
			if tt.notFoundAPI {
				assert.NotContains(t, rec.Body.String(), `<div id="app">`)
			}
		})
	}
}

// TestSPAFileHandlerRejectsTraversal guards the routing path normalisation in
// spaFileHandler: the "/api" prefix check must still catch dot-segment
// variants, and file reads must never escape distDir.
func TestSPAFileHandlerRejectsTraversal(t *testing.T) {
	distDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "index.html"), []byte("index"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(distDir, "app.js"), []byte("console.log('ok')"), 0o600))

	outsideDir := t.TempDir()
	secret := filepath.Join(outsideDir, "secret.js")
	require.NoError(t, os.WriteFile(secret, []byte("SECRET"), 0o600))

	handler := spaFileHandler(distDir)

	paths := []string{
		"/../secret.js",
		"/..%2fsecret.js",
		"/%2e%2e/secret.js",
		"/api/../secret.js",
		"/api/%2e%2e/secret.js",
		"/static/../secret.js",
		"/app.js/../../secret.js",
		"/..",
		"/../..",
		"//../secret.js",
	}

	for _, p := range paths {
		t.Run(p, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, p, nil)
			rec := httptest.NewRecorder()

			handler(rec, req)

			assert.NotContains(t, rec.Body.String(), "SECRET",
				"response leaked content outside distDir for path %q", p)
			assert.NotEqual(t, http.StatusOK, rec.Code,
				"expected non-200 for traversal path %q", p)
			// Dot-segments must never reach the SPA fallback either.
			assert.NotContains(t, rec.Body.String(), `<div id="app">`,
				"traversal path %q was rewritten to the SPA index", p)
		})
	}
}
