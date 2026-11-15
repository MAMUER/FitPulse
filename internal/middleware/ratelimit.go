package middleware

import (
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/MAMUER/project/internal/sanitize"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type userVisitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type authVisitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// rateLimiter manages rate limiting state for IP-based limiting
type rateLimiter struct {
	visitors sync.Map
}

// userRateLimiter manages rate limiting state for user-based limiting
type userRateLimiter struct {
	visitors sync.Map
}

// authRateLimiter manages rate limiting state for IP-based auth endpoints limiting
type authRateLimiter struct {
	visitors sync.Map
}

// Package-level singletons initialized at startup
var RateLimiterInstance = &rateLimiter{}

var UserRateLimiterInstance = &userRateLimiter{}

var AuthRateLimiterInstance = &authRateLimiter{}

func init() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			RateLimiterInstance.visitors.Range(func(key, value interface{}) bool {
				v := value.(*visitor)
				if time.Since(v.lastSeen) > 10*time.Minute {
					RateLimiterInstance.visitors.Delete(key)
				}
				return true
			})
			UserRateLimiterInstance.visitors.Range(func(key, value interface{}) bool {
				v := value.(*userVisitor)
				if time.Since(v.lastSeen) > 10*time.Minute {
					UserRateLimiterInstance.visitors.Delete(key)
				}
				return true
			})
			AuthRateLimiterInstance.visitors.Range(func(key, value interface{}) bool {
				v := value.(*authVisitor)
				if time.Since(v.lastSeen) > 10*time.Minute {
					AuthRateLimiterInstance.visitors.Delete(key)
				}
				return true
			})
		}
	}()
}

func resetRateLimiters() {
	RateLimiterInstance.visitors.Range(func(key, value interface{}) bool {
		RateLimiterInstance.visitors.Delete(key)
		return true
	})
	UserRateLimiterInstance.visitors.Range(func(key, value interface{}) bool {
		UserRateLimiterInstance.visitors.Delete(key)
		return true
	})
	AuthRateLimiterInstance.visitors.Range(func(key, value interface{}) bool {
		AuthRateLimiterInstance.visitors.Delete(key)
		return true
	})
}

// ResetRateLimiterForIP removes rate limit entries for the given IP.
func ResetRateLimiterForIP(ip string) {
	RateLimiterInstance.visitors.Delete(ip)
	AuthRateLimiterInstance.visitors.Delete(ip)
}

// ResetUserRateLimiterForID removes rate limit entries for the given user ID.
func ResetUserRateLimiterForID(userID string) {
	UserRateLimiterInstance.visitors.Delete(userID)
}

// CaptchaValidator reports whether token is a valid captcha solution for clientIP.
// The gateway implements it so a rate-limited client can prove it is human
// instead of forcing the limit to be raised for everyone.
type CaptchaValidator interface {
	ValidateCaptcha(token, clientIP string) bool
}

// AuthRateLimit enforces per-IP rate limiting for auth endpoints (5 attempts/minute, burst 5).
// When a CaptchaValidator is supplied, a client that exceeds the limit can still
// proceed by presenting a valid X-Captcha-Token; the 429 response advertises
// X-Captcha-Required so the UI knows to render the challenge.
func AuthRateLimit(log *zap.Logger, captcha ...CaptchaValidator) func(http.Handler) http.Handler {
	var validator CaptchaValidator
	if len(captcha) > 0 {
		validator = captcha[0]
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorizeAuthRequest(w, r, next, log, validator)
		})
	}
}

// authorizeAuthRequest applies the auth rate limit to a single request. A
// request over the limit is served when it carries a valid captcha token,
// otherwise it is rejected with 429 and X-Captcha-Required.
func authorizeAuthRequest(
	w http.ResponseWriter,
	r *http.Request,
	next http.Handler,
	log *zap.Logger,
	validator CaptchaValidator,
) {
	if r.URL.Path != "/api/v1/login" && r.URL.Path != "/api/v1/register" {
		next.ServeHTTP(w, r)
		return
	}

	ip := getClientIP(r)
	visitor := authVisitorFor(ip)
	visitor.lastSeen = time.Now()

	if visitor.limiter.Allow() {
		next.ServeHTTP(w, r)
		return
	}
	if captchaVerified(validator, r, ip) {
		next.ServeHTTP(w, r)
		return
	}

	log.Warn(
		"Auth rate limit exceeded",
		zap.String("path", sanitize.LogString(r.URL.Path)),
		zap.String("ip", sanitize.LogString(ip)),
	)
	w.Header().Set("X-Captcha-Required", "true")
	http.Error(w, "Превышен лимит запросов для авторизации", http.StatusTooManyRequests)
}

// authVisitorFor returns the per-IP auth limiter, creating it on first use.
func authVisitorFor(ip string) *authVisitor {
	visitor, ok := AuthRateLimiterInstance.visitors.Load(ip)
	if !ok {
		created := &authVisitor{
			limiter:  rate.NewLimiter(5.0/60.0, 5),
			lastSeen: time.Now(),
		}
		actual, _ := AuthRateLimiterInstance.visitors.LoadOrStore(ip, created)
		visitor = actual
	}
	return visitor.(*authVisitor)
}

// captchaVerified reports whether the request carries a captcha token accepted
// by the configured validator.
func captchaVerified(validator CaptchaValidator, r *http.Request, ip string) bool {
	if validator == nil {
		return false
	}
	return validator.ValidateCaptcha(r.Header.Get("X-Captcha-Token"), ip)
}

// RateLimit enforces per-IP rate limiting (10 r/s, burst 50)
func RateLimit(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" || r.URL.Path == "/api/v1/auth/refresh" {
				next.ServeHTTP(w, r)
				return
			}
			ip := getClientIP(r)
			v, ok := RateLimiterInstance.visitors.Load(ip)
			if !ok {
				limiter := rate.NewLimiter(10, 50)
				RateLimiterInstance.visitors.Store(ip, &visitor{limiter: limiter, lastSeen: time.Now()})
				v, _ = RateLimiterInstance.visitors.Load(ip)
			}
			vis := v.(*visitor)
			vis.lastSeen = time.Now()
			if !vis.limiter.Allow() {
				log.Warn("Rate limit exceeded", zap.String("path", sanitize.LogString(r.URL.Path)), zap.String("ip", sanitize.LogString(ip)))
				w.Header().Set("X-Captcha-Required", "true")
				http.Error(w, "Превышен лимит запросов", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// UserRateLimit enforces per-user rate limiting (100 r/s, burst 200)
func UserRateLimit(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(UserIDKey).(string)
			if !ok || userID == "" {
				next.ServeHTTP(w, r)
				return
			}
			if r.URL.Path == "/health" || r.URL.Path == "/api/v1/auth/refresh" {
				next.ServeHTTP(w, r)
				return
			}
			v, loaded := UserRateLimiterInstance.visitors.Load(userID)
			if !loaded {
				limiter := rate.NewLimiter(100, 200)
				stored := &userVisitor{limiter: limiter, lastSeen: time.Now()}
				UserRateLimiterInstance.visitors.Store(userID, stored)
				v = stored
			}
			vis := v.(*userVisitor)
			vis.lastSeen = time.Now()
			if !vis.limiter.Allow() {
				log.Warn("User rate limit exceeded", zap.String("user_id", sanitize.LogString(userID)), zap.String("path", sanitize.LogString(r.URL.Path)))
				http.Error(w, "Превышен лимит запросов для пользователя", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
