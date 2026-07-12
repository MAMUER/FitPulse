package middleware

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRemoveServerHeader(t *testing.T) {
	handler := RemoveServerHeader(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("Server") != "" {
		t.Error("Server header should be removed")
	}
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	headers := rec.Header()
	if headers.Get("X-Content-Type-Options") != "nosniff" {
		t.Error("Missing X-Content-Type-Options header")
	}
	if headers.Get("X-Frame-Options") != "DENY" {
		t.Error("Missing X-Frame-Options header")
	}
	if headers.Get("X-XSS-Protection") != "1; mode=block" {
		t.Error("Missing X-XSS-Protection header")
	}
	if headers.Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Error("Missing Referrer-Policy header")
	}
	if headers.Get("Content-Security-Policy") == "" {
		t.Error("Missing Content-Security-Policy header")
	}
}

func TestSecurityHeadersWithTLS(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	req.TLS = &tls.ConnectionState{} // simulate HTTPS
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Header().Get("Strict-Transport-Security") == "" {
		t.Error("Missing HSTS header when TLS")
	}
}

func TestNonceIsRandomBase64AndAtLeast128Bits(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 100; i++ {
		n := generateNonce()
		if _, dup := seen[n]; dup {
			t.Fatalf("nonce collision across 100 generations: %q", n)
		}
		seen[n] = struct{}{}

		decoded, err := base64.StdEncoding.DecodeString(n)
		if err != nil {
			t.Fatalf("nonce is not valid standard base64: %q (%v)", n, err)
		}
		// 32 bytes decoded = 256 bits >= 128 bits requirement
		if len(decoded) < 16 {
			t.Fatalf("nonce entropy too low: %d bytes (<16)", len(decoded))
		}
	}
}

func TestCSPPolicyIncludesNonceAndReportURI(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequestWithContext(context.Background(), "GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	csp := rec.Header().Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("missing CSP header")
	}
	if !strings.Contains(csp, "nonce-") {
		t.Error("CSP missing nonce directive")
	}
	if !strings.Contains(csp, "report-uri") {
		t.Error("CSP missing report-uri directive")
	}
	if !strings.Contains(csp, "report-to") {
		t.Error("CSP missing report-to directive")
	}
	if strings.Contains(csp, "unsafe-inline") {
		// Извлекаем только значение директивы script-src и проверяем, что в ней нет unsafe-inline.
		start := strings.Index(csp, "script-src")
		if start >= 0 {
			rest := csp[start:]
			end := strings.Index(rest, ";")
			if end >= 0 {
				scriptSrc := rest[:end]
				if strings.Contains(scriptSrc, "unsafe-inline") {
					t.Error("CSP must not contain unsafe-inline in script-src")
				}
			}
		}
	}
	if rec.Header().Get("Report-To") == "" {
		t.Error("missing Report-To header for report-to group")
	}
}

func TestNonceInjectAddsNonceToAllScriptTags(t *testing.T) {
	html := `<!DOCTYPE html><html><head></head><body>` +
		`<script src="/static/js/a.js"></script>` +
		`<script type="module" src="/static/js/b.js"></script>` +
		`<script>var x=1;</script>` +
		`</body></html>`

	handler := HTMLNonceInject(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(html))
	}))

	nonce := "test-nonce-123"
	ctx := context.WithValue(context.Background(), nonceContextKey{}, nonce)
	req := httptest.NewRequestWithContext(ctx, "GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	scripts := strings.Split(body, "<script")
	for _, s := range scripts[1:] {
		if !strings.Contains(s, `nonce="`+nonce+`"`) {
			t.Errorf("script tag missing nonce: %q", s)
		}
	}
}
