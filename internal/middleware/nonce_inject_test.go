package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTMLNonceInject_ServesNonHTML(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	})

	handler := HTMLNonceInject(next)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/data", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, `{"ok":true}`, rr.Body.String())
}

func TestHTMLNonceInject_NoNonce(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`<html><body>Hello</body></html>`))
	})

	handler := HTMLNonceInject(next)
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, `<html><body>Hello</body></html>`, rr.Body.String())
}

func TestHTMLNonceInject_WithNonce(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><script src="/app.js"></script></html>`))
		w.WriteHeader(http.StatusOK)
	})

	handler := HTMLNonceInject(next)
	ctx := context.WithValue(context.Background(), nonceContextKey{}, "abc123")
	req := httptest.NewRequestWithContext(ctx, "GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `nonce="abc123"`)
}

func TestHTMLNonceInject_AlreadyHasNonce(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<script nonce="existing">alert(1)</script>`))
		w.WriteHeader(http.StatusOK)
	})

	handler := HTMLNonceInject(next)
	ctx := context.WithValue(context.Background(), nonceContextKey{}, "abc123")
	req := httptest.NewRequestWithContext(ctx, "GET", "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `nonce="existing"`)
}

func TestInjectNonce(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		nonce    string
		expected string
	}{
		{
			name:     "adds nonce to script tag",
			body:     `<script src="/app.js"></script>`,
			nonce:    "abc123",
			expected: `<script nonce="abc123" src="/app.js"></script>`,
		},
		{
			name:     "no script tags",
			body:     `<div>Hello</div>`,
			nonce:    "abc123",
			expected: `<div>Hello</div>`,
		},
		{
			name:     "multiple script tags",
			body:     `<script src="/a.js"></script><script src="/b.js"></script>`,
			nonce:    "abc123",
			expected: `<script nonce="abc123" src="/a.js"></script><script nonce="abc123" src="/b.js"></script>`,
		},
		{
			name:     "preserves existing nonce",
			body:     `<script nonce="existing" src="/app.js"></script>`,
			nonce:    "abc123",
			expected: `<script nonce="existing" src="/app.js"></script>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := injectNonce([]byte(tt.body), tt.nonce)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestNonceInjectWriterWriteHeader(t *testing.T) {
	w := &nonceInjectWriter{
		ResponseWriter: httptest.NewRecorder(),
		nonce:          "test-nonce",
	}

	w.WriteHeader(http.StatusOK)
	assert.True(t, w.committed)
}

func TestNonceInjectWriterWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	w := &nonceInjectWriter{
		ResponseWriter: rec,
		nonce:          "test-nonce",
		buf:            []byte{},
	}

	n, err := w.Write([]byte(`<script src="/app.js"></script>`))
	assert.NoError(t, err)
	assert.Equal(t, 31, n)

	w.WriteHeader(http.StatusOK)
	assert.Contains(t, rec.Body.String(), `nonce="test-nonce"`)
}

func TestNonceInjectWriterWriteHeaderAfterCommitted(t *testing.T) {
	rec := httptest.NewRecorder()
	w := &nonceInjectWriter{
		ResponseWriter: rec,
		nonce:          "test-nonce",
		committed:      true,
	}

	w.WriteHeader(http.StatusOK)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, rec.Body.String())
}

func TestNonceInjectWriterWriteAfterCommitted(t *testing.T) {
	rec := httptest.NewRecorder()
	w := &nonceInjectWriter{
		ResponseWriter: rec,
		nonce:          "test-nonce",
		committed:      true,
	}

	n, err := w.Write([]byte("hello"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", rec.Body.String())
}

func TestNonceInjectWriterWriteErrorAfterCommitted(t *testing.T) {
	failingWriter := &failingResponseWriter{err: errors.New("write failed")}
	w := &nonceInjectWriter{
		ResponseWriter: failingWriter,
		nonce:          "test-nonce",
		committed:      true,
	}

	n, err := w.Write([]byte("data"))
	assert.Equal(t, 0, n)
	assert.Error(t, err)
	assert.Equal(t, "write response: write failed", err.Error())
}
