package main

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCSPReportHandler_MethodNotAllowed(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/api/v1/csp-report", nil)

	g.cspReportHandler(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestCSPReportHandler_BadRequest(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/csp-report", bytes.NewReader([]byte(`{invalid`)))
	req.Header.Set("Content-Type", "application/json")

	g.cspReportHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSPReportHandler_ReadBodyError(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/csp-report", &errorReader{err: io.ErrUnexpectedEOF})
	req.Header.Set("Content-Type", "application/json")

	g.cspReportHandler(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestCSPReportHandler_InvalidJSON(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/csp-report", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")

	g.cspReportHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSPReportHandler_SuccessReportingAPI(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	reqBody := []byte(`{"type":"csp-violation","url":"https://example.com","user_agent":"Mozilla","body":{"document-uri":"https://example.com","blocked-uri":"https://evil.com","violated-directive":"script-src"}}`)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/csp-report", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0")

	g.cspReportHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestCSPReportHandler_SuccessReportURI(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	reqBody := []byte(`{"csp-report":{"document-uri":"https://example.com","blocked-uri":"https://evil.com","violated-directive":"script-src"}}`)
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/csp-report", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	g.cspReportHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestCSPReportHandler_EmptyBody(t *testing.T) {
	g := newTestGateway()

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/csp-report", bytes.NewReader([]byte(``)))
	req.Header.Set("Content-Type", "application/json")

	g.cspReportHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCSPReportHandler_LargeBody(t *testing.T) {
	g := newTestGateway()

	largeBody := make([]byte, 65*1024)
	for i := range largeBody {
		largeBody[i] = 'a'
	}

	w := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), "POST", "/api/v1/csp-report", bytes.NewReader(largeBody))
	req.Header.Set("Content-Type", "application/json")

	g.cspReportHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
