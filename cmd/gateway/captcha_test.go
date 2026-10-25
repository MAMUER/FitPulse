package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type captchaMockProvider struct {
	valid     bool
	err       error
	verifyIP  string
	verifyTkn string
}

func (m *captchaMockProvider) Verify(responseToken, clientIP string) (bool, error) {
	m.verifyIP = clientIP
	m.verifyTkn = responseToken
	return m.valid, m.err
}

func (m *captchaMockProvider) Name() string {
	return "captcha-mock"
}

type captchaFailingWriter struct{}

func (captchaFailingWriter) Header() http.Header        { return http.Header{} }
func (captchaFailingWriter) Write([]byte) (int, error)  { return 0, errors.New("write failed") }
func (captchaFailingWriter) WriteHeader(statusCode int) {}

type captchaRewriteTransport struct {
	orig http.RoundTripper
	host string
}

func (rt *captchaRewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.Host, "challenges.cloudflare.com") ||
		strings.Contains(req.URL.Host, "api.hcaptcha.com") {
		req.URL.Scheme = "http"
		req.URL.Host = rt.host
	}
	return rt.orig.RoundTrip(req)
}

func captchaInstallRewriteTransport(t *testing.T, host string) {
	t.Helper()
	orig := http.DefaultTransport
	http.DefaultTransport = &captchaRewriteTransport{orig: orig, host: host}
	t.Cleanup(func() { http.DefaultTransport = orig })
}

func captchaTestGateway(provider CaptchaProvider, siteKey string) *gateway {
	g := newTestGateway()
	g.captchaProvider = provider
	g.captchaSiteKey = siteKey
	return g
}

func captchaMiddlewareWithLog(provider CaptchaProvider) *captchaMiddleware {
	return &captchaMiddleware{provider: provider, log: zap.NewNop()}
}

func captchaVerifyHTTPCases(t *testing.T, provider CaptchaProvider, verifyPath, logPrefix string) {
	t.Helper()

	t.Run("success", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, verifyPath, r.URL.Path)
			require.NoError(t, r.ParseForm())
			assert.Equal(t, "sekret", r.FormValue("secret"))
			assert.Equal(t, "token-abc", r.FormValue("response"))
			assert.Equal(t, "10.0.0.1", r.FormValue("remoteip"))
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success":     true,
				"challengeTs": "2024-01-01T00:00:00Z",
				"hostname":    "example.com",
				"action":      "login",
			})
		}))
		defer server.Close()
		captchaInstallRewriteTransport(t, server.Listener.Addr().String())

		valid, err := provider.Verify("token-abc", "10.0.0.1")
		assert.True(t, valid)
		assert.NoError(t, err)
	})

	t.Run("verification failed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success":     false,
				"error-codes": []string{"invalid-input-response"},
			})
		}))
		defer server.Close()
		captchaInstallRewriteTransport(t, server.Listener.Addr().String())

		valid, err := provider.Verify("token-abc", "10.0.0.1")
		assert.False(t, valid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), logPrefix+" failed")
		assert.Contains(t, err.Error(), "invalid-input-response")
	})

	t.Run("response read error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Length", "100")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"success":true}`))
		}))
		defer server.Close()
		captchaInstallRewriteTransport(t, server.Listener.Addr().String())

		valid, err := provider.Verify("token-abc", "10.0.0.1")
		assert.False(t, valid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), logPrefix+" response read")
	})

	t.Run("response parse error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `not-json`)
		}))
		defer server.Close()
		captchaInstallRewriteTransport(t, server.Listener.Addr().String())

		valid, err := provider.Verify("token-abc", "10.0.0.1")
		assert.False(t, valid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), logPrefix+" response parse")
	})

	t.Run("request error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		addr := server.Listener.Addr().String()
		server.Close()
		captchaInstallRewriteTransport(t, addr)

		valid, err := provider.Verify("token-abc", "10.0.0.1")
		assert.False(t, valid)
		require.Error(t, err)
		assert.Contains(t, err.Error(), logPrefix+" verify request")
	})
}

func TestNewTurnstileProvider(t *testing.T) {
	p := newTurnstileProvider("secret-key")
	assert.Equal(t, "cloudflare-turnstile", p.Name())
	cfg, ok := p.(*turnstileConfig)
	require.True(t, ok)
	assert.Equal(t, "secret-key", cfg.secretKey)
	assert.Equal(t, 10*time.Second, cfg.timeout)
}

func TestNewHCaptchaProvider(t *testing.T) {
	p := newHCaptchaProvider("secret-key")
	assert.Equal(t, "hcaptcha", p.Name())
	cfg, ok := p.(*hCaptchaConfig)
	require.True(t, ok)
	assert.Equal(t, "secret-key", cfg.secretKey)
	assert.Equal(t, 10*time.Second, cfg.timeout)
}

func TestTurnstileVerify_EmptyToken(t *testing.T) {
	valid, err := newTurnstileProvider("secret").Verify("", "10.0.0.1")
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestTurnstileVerify_HTTP(t *testing.T) {
	captchaVerifyHTTPCases(t, newTurnstileProvider("sekret"), "/turnstile/v0/siteverify", "turnstile")
}

func TestHCaptchaVerify_EmptyToken(t *testing.T) {
	valid, err := newHCaptchaProvider("secret").Verify("", "10.0.0.1")
	assert.False(t, valid)
	assert.NoError(t, err)
}

func TestHCaptchaVerify_HTTP(t *testing.T) {
	captchaVerifyHTTPCases(t, newHCaptchaProvider("sekret"), "/siteverify", "hcaptcha")
}

func TestLoadCaptchaProvider_Cloudflare(t *testing.T) {
	t.Setenv("CAPTCHA_PROVIDER", "cloudflare")
	t.Setenv("CLOUDFLARE_TURNSTILE_SECRET_KEY", "cf-secret")
	t.Setenv("CLOUDFLARE_TURNSTILE_SITE_KEY", "cf-site")
	t.Setenv("HCAPTCHA_SECRET_KEY", "")
	t.Setenv("HCAPTCHA_SITE_KEY", "")

	provider, siteKey, err := loadCaptchaProvider()
	require.NoError(t, err)
	assert.Equal(t, "cf-site", siteKey)
	require.NotNil(t, provider)
	assert.Equal(t, "cloudflare-turnstile", provider.Name())
}

func TestLoadCaptchaProvider_DefaultIsCloudflare(t *testing.T) {
	t.Setenv("CAPTCHA_PROVIDER", "")
	t.Setenv("CLOUDFLARE_TURNSTILE_SECRET_KEY", "cf-secret")
	t.Setenv("CLOUDFLARE_TURNSTILE_SITE_KEY", "cf-site")
	t.Setenv("HCAPTCHA_SECRET_KEY", "")
	t.Setenv("HCAPTCHA_SITE_KEY", "")

	provider, siteKey, err := loadCaptchaProvider()
	require.NoError(t, err)
	assert.Equal(t, "cf-site", siteKey)
	require.NotNil(t, provider)
	assert.Equal(t, "cloudflare-turnstile", provider.Name())
}

func TestLoadCaptchaProvider_MissingCloudflareSecret(t *testing.T) {
	t.Setenv("CAPTCHA_PROVIDER", "cloudflare")
	t.Setenv("CLOUDFLARE_TURNSTILE_SECRET_KEY", "")
	t.Setenv("CLOUDFLARE_TURNSTILE_SITE_KEY", "cf-site")

	provider, siteKey, err := loadCaptchaProvider()
	require.Error(t, err)
	assert.Nil(t, provider)
	assert.Empty(t, siteKey)
	assert.Contains(t, err.Error(), "CLOUDFLARE_TURNSTILE_SECRET_KEY is required")
}

func TestLoadCaptchaProvider_MissingCloudflareSiteKey(t *testing.T) {
	t.Setenv("CAPTCHA_PROVIDER", "cloudflare")
	t.Setenv("CLOUDFLARE_TURNSTILE_SECRET_KEY", "cf-secret")
	t.Setenv("CLOUDFLARE_TURNSTILE_SITE_KEY", "")

	provider, siteKey, err := loadCaptchaProvider()
	require.Error(t, err)
	assert.Nil(t, provider)
	assert.Empty(t, siteKey)
	assert.Contains(t, err.Error(), "CLOUDFLARE_TURNSTILE_SITE_KEY is required")
}

func TestLoadCaptchaProvider_HCaptcha(t *testing.T) {
	t.Setenv("CAPTCHA_PROVIDER", "hcaptcha")
	t.Setenv("CLOUDFLARE_TURNSTILE_SECRET_KEY", "")
	t.Setenv("CLOUDFLARE_TURNSTILE_SITE_KEY", "")
	t.Setenv("HCAPTCHA_SECRET_KEY", "hc-secret")
	t.Setenv("HCAPTCHA_SITE_KEY", "hc-site")

	provider, siteKey, err := loadCaptchaProvider()
	require.NoError(t, err)
	assert.Equal(t, "hc-site", siteKey)
	require.NotNil(t, provider)
	assert.Equal(t, "hcaptcha", provider.Name())
}

func TestLoadCaptchaProvider_MissingHCaptchaSecret(t *testing.T) {
	t.Setenv("CAPTCHA_PROVIDER", "hcaptcha")
	t.Setenv("HCAPTCHA_SECRET_KEY", "")
	t.Setenv("HCAPTCHA_SITE_KEY", "hc-site")

	provider, siteKey, err := loadCaptchaProvider()
	require.Error(t, err)
	assert.Nil(t, provider)
	assert.Empty(t, siteKey)
	assert.Contains(t, err.Error(), "HCAPTCHA_SECRET_KEY is required")
}

func TestLoadCaptchaProvider_MissingHCaptchaSiteKey(t *testing.T) {
	t.Setenv("CAPTCHA_PROVIDER", "hcaptcha")
	t.Setenv("HCAPTCHA_SECRET_KEY", "hc-secret")
	t.Setenv("HCAPTCHA_SITE_KEY", "")

	provider, siteKey, err := loadCaptchaProvider()
	require.Error(t, err)
	assert.Nil(t, provider)
	assert.Empty(t, siteKey)
	assert.Contains(t, err.Error(), "HCAPTCHA_SITE_KEY is required")
}

func TestLoadCaptchaProvider_None(t *testing.T) {
	t.Setenv("CAPTCHA_PROVIDER", "none")

	provider, siteKey, err := loadCaptchaProvider()
	require.NoError(t, err)
	assert.Nil(t, provider)
	assert.Empty(t, siteKey)
}

func TestLoadCaptchaProvider_Unsupported(t *testing.T) {
	t.Setenv("CAPTCHA_PROVIDER", "recaptcha")

	provider, siteKey, err := loadCaptchaProvider()
	require.Error(t, err)
	assert.Nil(t, provider)
	assert.Empty(t, siteKey)
	assert.Contains(t, err.Error(), "unsupported CAPTCHA provider: recaptcha")
}

func TestWriteCaptchaRequired(t *testing.T) {
	m := captchaMiddlewareWithLog(nil)

	w := httptest.NewRecorder()
	m.WriteCaptchaRequired(w, "site-key-123")

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var resp captchaResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "Rate limit exceeded. Please complete CAPTCHA.", resp.Error)
	assert.True(t, resp.CaptchaRequired)
	assert.Equal(t, "site-key-123", resp.SiteKey)
}

func TestWriteCaptchaRequired_EmptySiteKeyOmitted(t *testing.T) {
	m := captchaMiddlewareWithLog(nil)

	w := httptest.NewRecorder()
	m.WriteCaptchaRequired(w, "")

	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	var resp captchaResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.Equal(t, "", resp.SiteKey)
	assert.NotContains(t, w.Body.String(), "site_key")
}

func TestWriteCaptchaRequired_EncodeError(t *testing.T) {
	m := captchaMiddlewareWithLog(nil)

	assert.NotPanics(t, func() {
		m.WriteCaptchaRequired(captchaFailingWriter{}, "site-key-123")
	})
}

func TestVerifyCaptchaToken_NoToken(t *testing.T) {
	provider := &captchaMockProvider{valid: true}
	m := captchaMiddlewareWithLog(provider)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", nil)
	valid, err := m.VerifyCaptchaToken(req, "10.0.0.1")
	assert.False(t, valid)
	assert.NoError(t, err)
	assert.Empty(t, provider.verifyTkn)
}

func TestVerifyCaptchaToken_Valid(t *testing.T) {
	provider := &captchaMockProvider{valid: true}
	m := captchaMiddlewareWithLog(provider)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", nil)
	req.Header.Set("X-Captcha-Token", "token-xyz")
	valid, err := m.VerifyCaptchaToken(req, "10.0.0.9")
	assert.True(t, valid)
	assert.NoError(t, err)
	assert.Equal(t, "token-xyz", provider.verifyTkn)
	assert.Equal(t, "10.0.0.9", provider.verifyIP)
}

func TestVerifyCaptchaToken_ProviderError(t *testing.T) {
	provider := &captchaMockProvider{valid: false, err: errors.New("provider boom")}
	m := captchaMiddlewareWithLog(provider)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", nil)
	req.Header.Set("X-Captcha-Token", "token-xyz")
	valid, err := m.VerifyCaptchaToken(req, "10.0.0.9")
	assert.False(t, valid)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "provider boom")
	assert.Equal(t, "token-xyz", provider.verifyTkn)
}

func TestCaptchaMiddleware_DisabledProvider(t *testing.T) {
	g := captchaTestGateway(nil, "")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("passed"))
	})

	h := g.CaptchaMiddleware(next)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "passed", w.Body.String())
}

func TestCaptchaMiddleware_ValidToken(t *testing.T) {
	g := captchaTestGateway(&captchaMockProvider{valid: true}, "site-key")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("passed"))
	})

	h := g.CaptchaMiddleware(next)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("X-Captcha-Token", "token-xyz")
	req.Header.Set("X-Forwarded-For", "203.0.113.7, 10.0.0.1")
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "passed", w.Body.String())
}

func TestCaptchaMiddleware_InvalidToken(t *testing.T) {
	g := captchaTestGateway(&captchaMockProvider{valid: false}, "site-key")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("passed"))
	})

	h := g.CaptchaMiddleware(next)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("X-Captcha-Token", "token-xyz")
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)

	var resp captchaResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	assert.True(t, resp.CaptchaRequired)
	assert.Equal(t, "site-key", resp.SiteKey)
}

func TestCaptchaMiddleware_ProviderErrorFallsBackToCaptcha(t *testing.T) {
	g := captchaTestGateway(&captchaMockProvider{valid: false, err: errors.New("provider boom")}, "site-key")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("passed"))
	})

	h := g.CaptchaMiddleware(next)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	req.Header.Set("X-Captcha-Token", "token-xyz")
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestCaptchaMiddleware_MissingToken(t *testing.T) {
	g := captchaTestGateway(&captchaMockProvider{valid: true}, "site-key")
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("passed"))
	})

	h := g.CaptchaMiddleware(next)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	h.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	assert.Contains(t, w.Body.String(), "captcha_required")
}

type closeFailingBodyTransport struct{}

func (c *closeFailingBodyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       closeFailingBody{ReadCloser: &closeFailingReader{}},
		Header:     http.Header{},
	}, nil
}

type closeFailingReader struct{}

func (c *closeFailingReader) Read(p []byte) (int, error) { return 0, io.EOF }
func (c *closeFailingReader) Close() error                { return errors.New("close failed") }

type closeFailingBody struct{ io.ReadCloser }

func (c closeFailingBody) Close() error { return errors.New("close failed") }

func TestPostCaptcha_BodyCloseError(t *testing.T) {
	origTransport := http.DefaultTransport
	http.DefaultTransport = &closeFailingBodyTransport{}
	t.Cleanup(func() { http.DefaultTransport = origTransport })

	_, err := postCaptcha("test", "http://example.com/verify", "secret", "token", "1.2.3.4")
	require.NoError(t, err)
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		expected   string
	}{
		{name: "x-forwarded-for first entry", xff: "1.1.1.1, 2.2.2.2", remoteAddr: "10.0.0.5:1234", expected: "1.1.1.1"},
		{name: "x-forwarded-for single entry", xff: "  3.3.3.3 ", remoteAddr: "10.0.0.5:1234", expected: "3.3.3.3"},
		{name: "x-real-ip", xri: "4.4.4.4", remoteAddr: "10.0.0.5:1234", expected: "4.4.4.4"},
		{name: "remote addr with port", remoteAddr: "10.0.0.5:1234", expected: "10.0.0.5"},
		{name: "remote addr without port", remoteAddr: "10.0.0.5", expected: "10.0.0.5"},
		{name: "empty remote addr", remoteAddr: "", expected: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}
			req.RemoteAddr = tt.remoteAddr

			assert.Equal(t, tt.expected, getClientIP(req))
		})
	}
}
