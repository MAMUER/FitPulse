package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
)

// CaptchaProvider defines the interface for CAPTCHA verification.
type CaptchaProvider interface {
	Verify(responseToken, clientIP string) (bool, error)
	Name() string
}

// turnstileConfig holds Cloudflare Turnstile configuration.
type turnstileConfig struct {
	secretKey string
	timeout   time.Duration
}

// newTurnstileProvider creates a Cloudflare Turnstile CAPTCHA provider.
func newTurnstileProvider(secretKey string) CaptchaProvider {
	return &turnstileConfig{
		secretKey: secretKey,
		timeout:   10 * time.Second,
	}
}

func (c *turnstileConfig) Name() string {
	return "cloudflare-turnstile"
}

// Verify validates a Turnstile response token with Cloudflare.
func (c *turnstileConfig) Verify(responseToken, clientIP string) (bool, error) {
	if responseToken == "" {
		return false, nil
	}

	resp, err := http.Post("https://challenges.cloudflare.com/turnstile/v0/siteverify",
		"application/x-www-form-urlencoded",
		strings.NewReader(fmt.Sprintf("secret=%s&response=%s&remoteip=%s",
			c.secretKey, responseToken, clientIP)))
	if err != nil {
		return false, fmt.Errorf("turnstile verify request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Success     bool     `json:"success"`
		ChallengeTS string   `json:"challenge_ts"`
		Hostname    string   `json:"hostname"`
		ErrorCodes  []string `json:"error-codes"`
		Action      string   `json:"action"`
		Cdata       string   `json:"cdata"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("turnstile response parse: %w", err)
	}

	if !result.Success {
		return false, fmt.Errorf("turnstile failed: %v", result.ErrorCodes)
	}

	return true, nil
}

// captchaMiddleware adds CAPTCHA verification when rate limit is exceeded.
type captchaMiddleware struct {
	provider CaptchaProvider
	log      *zap.Logger
}

// captchaResponse represents the JSON response when CAPTCHA is required.
type captchaResponse struct {
	Error           string `json:"error"`
	CaptchaRequired bool   `json:"captcha_required"`
	SiteKey         string `json:"site_key,omitempty"`
}

// WriteCaptchaRequired writes a 429 response with CAPTCHA challenge.
func (m *captchaMiddleware) WriteCaptchaRequired(w http.ResponseWriter, siteKey string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusTooManyRequests)
	json.NewEncoder(w).Encode(captchaResponse{
		Error:           "Rate limit exceeded. Please complete CAPTCHA.",
		CaptchaRequired: true,
		SiteKey:         siteKey,
	})
}

// VerifyCaptchaToken verifies the CAPTCHA token from request.
func (m *captchaMiddleware) VerifyCaptchaToken(r *http.Request, clientIP string) (bool, error) {
	token := r.Header.Get("X-Captcha-Token")
	if token == "" {
		return false, nil
	}
	valid, err := m.provider.Verify(token, clientIP)
	if err != nil {
		m.log.Warn("CAPTCHA verification failed", zap.Error(err))
	}
	return valid, err
}

// loadCaptchaProvider initializes CAPTCHA provider from environment.
func loadCaptchaProvider() (CaptchaProvider, string, error) {
	provider := os.Getenv("CAPTCHA_PROVIDER")
	switch provider {
	case "cloudflare", "":
		secretKey := os.Getenv("CLOUDFLARE_TURNSTILE_SECRET_KEY")
		if secretKey == "" {
			return nil, "", errors.New("CLOUDFLARE_TURNSTILE_SECRET_KEY is required for cloudflare provider")
		}
		siteKey := os.Getenv("CLOUDFLARE_TURNSTILE_SITE_KEY")
		if siteKey == "" {
			return nil, "", errors.New("CLOUDFLARE_TURNSTILE_SITE_KEY is required for cloudflare provider")
		}
		return newTurnstileProvider(secretKey), siteKey, nil
	case "hcaptcha":
		secretKey := os.Getenv("HCAPTCHA_SECRET_KEY")
		if secretKey == "" {
			return nil, "", errors.New("HCAPTCHA_SECRET_KEY is required for hcaptcha provider")
		}
		siteKey := os.Getenv("HCAPTCHA_SITE_KEY")
		if siteKey == "" {
			return nil, "", errors.New("HCAPTCHA_SITE_KEY is required for hcaptcha provider")
		}
		return newHCaptchaProvider(secretKey), siteKey, nil
	case "none":
		return nil, "", nil
	default:
		return nil, "", fmt.Errorf("unsupported CAPTCHA provider: %s", provider)
	}
}

// hCaptcha provider implementation.
type hCaptchaConfig struct {
	secretKey string
	timeout   time.Duration
}

func newHCaptchaProvider(secretKey string) CaptchaProvider {
	return &hCaptchaConfig{
		secretKey: secretKey,
		timeout:   10 * time.Second,
	}
}

func (c *hCaptchaConfig) Name() string {
	return "hcaptcha"
}

func (c *hCaptchaConfig) Verify(responseToken, clientIP string) (bool, error) {
	if responseToken == "" {
		return false, nil
	}

	resp, err := http.Post("https://api.hcaptcha.com/siteverify",
		"application/x-www-form-urlencoded",
		strings.NewReader(fmt.Sprintf("secret=%s&response=%s&remoteip=%s",
			c.secretKey, responseToken, clientIP)))
	if err != nil {
		return false, fmt.Errorf("hcaptcha verify request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result struct {
		Success     bool     `json:"success"`
		ChallengeTS string   `json:"challenge_ts"`
		Hostname    string   `json:"hostname"`
		ErrorCodes  []string `json:"error-codes"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("hcaptcha response parse: %w", err)
	}

	if !result.Success {
		return false, fmt.Errorf("hcaptcha failed: %v", result.ErrorCodes)
	}

	return true, nil
}

// CaptchaMiddleware returns a chi middleware that enforces CAPTCHA verification.
func (g *gateway) CaptchaMiddleware(next http.Handler) http.Handler {
	if g.captchaProvider == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Captcha-Token")
		if token != "" {
			clientIP := getClientIP(r)
			valid, err := g.captchaProvider.Verify(token, clientIP)
			if err != nil {
				g.log.Warn("CAPTCHA verification failed", zap.Error(err))
			}
			if valid {
				next.ServeHTTP(w, r)
				return
			}
		}
		mw := &captchaMiddleware{
			provider: g.captchaProvider,
			log:      g.log.Logger,
		}
		mw.WriteCaptchaRequired(w, g.captchaSiteKey)
	})
}

// getClientIP extracts client IP from request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For (set by ingress/WAF)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	// Fallback to RemoteAddr
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		return ip[:idx]
	}
	return ip
}
