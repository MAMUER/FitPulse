package middleware

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestErrorPages_NotFoundHTML(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	handler := ErrorPages(next)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/not-found", nil)
	req.Header.Set("Accept", "text/html")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "not found", rr.Body.String())
}

func TestErrorPages_NotFoundJSON(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})

	handler := ErrorPages(next)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/not-found", nil)
	req.Header.Set("Accept", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Equal(t, "not found", rr.Body.String())
}

func TestErrorPages_InternalServerErrorHTML(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	})

	handler := ErrorPages(next)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/error", nil)
	req.Header.Set("Accept", "text/html")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Equal(t, "server error", rr.Body.String())
}

func TestErrorPages_ForbiddenHTML(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden"))
	})

	handler := ErrorPages(next)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/forbidden", nil)
	req.Header.Set("Accept", "text/html")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Equal(t, "forbidden", rr.Body.String())
}

func TestErrorPages_OKResponse(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler := ErrorPages(next)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "ok", rr.Body.String())
}

func TestErrorPages_UnknownStatusHTML(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("teapot"))
	})

	handler := ErrorPages(next)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("Accept", "text/html")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusTeapot, rr.Code)
	assert.Equal(t, "teapot", rr.Body.String())
}

func TestErrorPageRecorder_Write(t *testing.T) {
	rec := &errorPageRecorder{
		ResponseWriter: httptest.NewRecorder(),
		body:           &bytes.Buffer{},
	}

	n, err := rec.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", rec.body.String())
	assert.True(t, rec.wrote)
}

func TestErrorPageRecorder_WriteHeader(t *testing.T) {
	rec := &errorPageRecorder{
		ResponseWriter: httptest.NewRecorder(),
		body:           &bytes.Buffer{},
	}

	rec.WriteHeader(http.StatusNotFound)
	assert.Equal(t, http.StatusNotFound, rec.statusCode)
	assert.False(t, rec.wrote)
}

func TestErrorPageRecorder_Replay(t *testing.T) {
	rec := httptest.NewRecorder()
	body := []byte("error body")
	rec.Header().Set("X-Custom", "value")

	r := &errorPageRecorder{
		ResponseWriter: rec,
		headers:        rec.Header(),
		body:           bytes.NewBuffer(body),
		statusCode:     http.StatusNotFound,
	}

	r.replay(http.StatusInternalServerError, body)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "value", rec.Header().Get("X-Custom"))
	assert.Equal(t, "error body", rec.Body.String())
}

func TestServeErrorPage_MissingFile(t *testing.T) {
	rec := httptest.NewRecorder()
	body := []byte("original body")
	r := &errorPageRecorder{
		ResponseWriter: rec,
		headers:        rec.Header(),
		body:           bytes.NewBuffer(body),
		statusCode:     http.StatusNotFound,
	}

	serveErrorPage(rec, r, http.StatusNotFound)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Equal(t, "original body", rec.Body.String())
}

func TestServeErrorPage_DisallowedPath(t *testing.T) {
	rec := httptest.NewRecorder()
	body := []byte("original body")
	r := &errorPageRecorder{
		ResponseWriter: rec,
		headers:        rec.Header(),
		body:           bytes.NewBuffer(body),
		statusCode:     http.StatusNotFound,
	}

	serveErrorPage(rec, r, http.StatusOK)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "original body", rec.Body.String())
}

func TestErrorPageFileForStatus(t *testing.T) {
	base := filepath.Clean("./web/static/errors")

	t.Run("forbidden", func(t *testing.T) {
		assert.Equal(t, filepath.Join(base, "403.html"), errorPageFileForStatus(http.StatusForbidden))
	})

	t.Run("not found", func(t *testing.T) {
		assert.Equal(t, filepath.Join(base, "404.html"), errorPageFileForStatus(http.StatusNotFound))
	})

	t.Run("internal server error", func(t *testing.T) {
		assert.Equal(t, filepath.Join(base, "500.html"), errorPageFileForStatus(http.StatusInternalServerError))
	})
}

func TestErrorPages_WriteHeadersPreserved(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "custom-value")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	})

	handler := ErrorPages(next)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.Header.Set("Accept", "text/html")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Equal(t, "custom-value", rr.Header().Get("X-Custom-Header"))
}
