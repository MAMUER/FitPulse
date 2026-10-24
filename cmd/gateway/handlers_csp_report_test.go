package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCspReportHandler(t *testing.T) {
	g := newTestGateway()

	// non-post
	req := httptest.NewRequest(http.MethodGet, "/api/security/csp-report", nil)
	w := httptest.NewRecorder()
	g.cspReportHandler(w, req)
	require.Equal(t, http.StatusMethodNotAllowed, w.Code)

	// bad json
	req = httptest.NewRequest(http.MethodPost, "/api/security/csp-report", bytes.NewBufferString("bad"))
	w = httptest.NewRecorder()
	g.cspReportHandler(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// report-uri format
	req = httptest.NewRequest(http.MethodPost, "/api/security/csp-report", bytes.NewBufferString(`{"csp-report":{"blocked-uri":"http://x","violated-directive":"script-src"}}`))
	req.Header.Set("User-Agent", "test")
	w = httptest.NewRecorder()
	g.cspReportHandler(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// reporting api format
	req = httptest.NewRequest(http.MethodPost, "/api/security/csp-report", bytes.NewBufferString(`{"type":"csp-violation","body":{"blocked-uri":"http://y","violated-directive":"style-src"}}`))
	w = httptest.NewRecorder()
	g.cspReportHandler(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}
