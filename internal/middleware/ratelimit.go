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

// AuthRateLimit enforces per-IP rate limiting for auth endpoints (5 attempts/minute, burst 5)
func AuthRateLimit(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/api/v1/login" && r.URL.Path != "/api/v1/register" {
				next.ServeHTTP(w, r)
				return
			}
			ip := getClientIP(r)
			v, ok := AuthRateLimiterInstance.visitors.Load(ip)
			if !ok {
				limiter := rate.NewLimiter(5.0/60.0, 5)
				AuthRateLimiterInstance.visitors.Store(ip, &authVisitor{limiter: limiter, lastSeen: time.Now()})
				v, _ = AuthRateLimiterInstance.visitors.Load(ip)
			}
			av := v.(*authVisitor)
			av.lastSeen = time.Now()
			if !av.limiter.Allow() {
				log.Warn("Auth rate limit exceeded", zap.String("path", sanitize.LogString(r.URL.Path)), zap.String("ip", sanitize.LogString(ip)))
				w.Header().Set("X-Captcha-Required", "true")
				http.Error(w, "Превышен лимит запросов для авторизации", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
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
			userID, _ := r.Context().Value(UserIDKey).(string)
			if userID == "" {
				next.ServeHTTP(w, r)
				return
			}
			if r.URL.Path == "/health" || r.URL.Path == "/api/v1/auth/refresh" {
				next.ServeHTTP(w, r)
				return
			}
			v, ok := UserRateLimiterInstance.visitors.Load(userID)
			if !ok {
				limiter := rate.NewLimiter(100, 200)
				UserRateLimiterInstance.visitors.Store(userID, &userVisitor{limiter: limiter, lastSeen: time.Now()})
				v, _ = UserRateLimiterInstance.visitors.Load(userID)
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
