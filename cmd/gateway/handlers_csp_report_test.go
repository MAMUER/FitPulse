package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSPReportHandler_MethodNotAllowed(t *testing.T) {
	g := &gateway{log: nil}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/security/csp-report", nil)
	g.cspReportHandler(w, r)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestCSPReportHandler_InvalidJSON(t *testing.T) {
	g := &gateway{log: nil}
	w := httptest.NewRecorder()
	body := bytes.NewReader([]byte("not-json"))
	r := httptest.NewRequest(http.MethodPost, "/api/security/csp-report", body)
	g.cspReportHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "{}", w.Body.String())
}

func TestCSPReportHandler_ValidReport(t *testing.T) {
	g := &gateway{log: nil}
	report := CSPReportBody{
		Report: CSPViolationBody{
			BlockedURI:        "https://evil.com/script.js",
			ViolatedDirective: "script-src",
		},
	}
	body, _ := json.Marshal(report)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/security/csp-report", bytes.NewReader(body))
	r.Header.Set("User-Agent", "Mozilla/5.0")
	r.RemoteAddr = "127.0.0.1"
	g.cspReportHandler(w, r)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "{}", w.Body.String())
}

func TestCSPReportBody_Unmarshal(t *testing.T) {
	body := `{"type":"csp-report","csp-report":{"document-uri":"https://example.com","blocked-uri":"https://evil.com","violated-directive":"script-src"}}`
	var report CSPReportBody
	err := json.Unmarshal([]byte(body), &report)
	require.NoError(t, err)
	assert.Equal(t, "https://evil.com", report.Report.BlockedURI)
	assert.Equal(t, "script-src", report.Report.ViolatedDirective)
}

func TestStringsContainsIgnoreCase(t *testing.T) {
	assert.True(t, strings.Contains(strings.ToLower("Hello"), strings.ToLower("hello")))
	assert.False(t, strings.Contains(strings.ToLower("Hello"), strings.ToLower("world")))
}
