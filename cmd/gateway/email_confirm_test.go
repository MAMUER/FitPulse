package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/MAMUER/project/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGateway_EmailConfirmPageHandler(t *testing.T) {
	// Create a temporary template directory
	tmpDir := t.TempDir()
	tmplDir := filepath.Join(tmpDir, "web", "templates")
	require.NoError(t, os.MkdirAll(tmplDir, 0755))

	tmplContent := `<!DOCTYPE html>
<html lang="ru"><head><meta charset="UTF-8">
<title>Test</title>
<link rel="stylesheet" href="/static/css/confirm.css">
</head>
<body>
  <div class="container">
    <h1>Подтверждение email</h1>
    <p>Токен: {{ .Token }}</p>
  </div>
</body></html>`
	require.NoError(t, os.WriteFile(filepath.Join(tmplDir, "confirm.html"), []byte(tmplContent), 0644))

	// Save and restore working directory
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	require.NoError(t, os.Chdir(tmpDir))

	log := logger.New("test-gateway")
	g := &gateway{log: log}

	t.Run("valid token - serves template", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/confirm?token=abc123", nil)
		rr := httptest.NewRecorder()

		g.emailConfirmPageHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "Подтверждение email")
		assert.Contains(t, rr.Body.String(), "abc123")
		assert.Contains(t, rr.Body.String(), `<link rel="stylesheet" href="/static/css/confirm.css">`)
		assert.NotContains(t, rr.Body.String(), "<style>")
		assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
	})

	t.Run("empty token - serves template with empty token", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/confirm", nil)
		rr := httptest.NewRecorder()

		g.emailConfirmPageHandler(rr, req)

		// When template exists, it renders with empty token (frontend JS handles the error)
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "Подтверждение email")
		assert.Contains(t, rr.Header().Get("Content-Type"), "text/html")
	})
}

func TestGateway_EmailConfirmPageHandler_NoTemplateFile(t *testing.T) {
	// Test with non-existent template directory
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()
	require.NoError(t, os.Chdir(tmpDir))

	log := logger.New("test-gateway")
	g := &gateway{log: log}

	t.Run("fallback with token", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/confirm?token=xyz789", nil)
		rr := httptest.NewRecorder()

		g.emailConfirmPageHandler(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "xyz789")
		assert.Contains(t, rr.Body.String(), "Подтверждение email")
	})

	t.Run("fallback without token", func(t *testing.T) {
		req := httptest.NewRequestWithContext(context.Background(), "GET", "/confirm", nil)
		rr := httptest.NewRecorder()

		g.emailConfirmPageHandler(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "Ошибка")
		assert.Contains(t, rr.Body.String(), "Токен не найден")
	})
}
