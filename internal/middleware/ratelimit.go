package middleware

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type userVisitor struct {
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

// Package-level singletons initialized at startup
//
//nolint:gochecknoglobals
var rateLimiterInstance = &rateLimiter{}

//nolint:gochecknoglobals
var userRateLimiterInstance = &userRateLimiter{}

func init() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rateLimiterInstance.visitors.Range(func(key, value interface{}) bool {
				v := value.(*visitor)
				if time.Since(v.lastSeen) > 10*time.Minute {
					rateLimiterInstance.visitors.Delete(key)
				}
				return true
			})
			userRateLimiterInstance.visitors.Range(func(key, value interface{}) bool {
				v := value.(*userVisitor)
				if time.Since(v.lastSeen) > 10*time.Minute {
					userRateLimiterInstance.visitors.Delete(key)
				}
				return true
			})
		}
	}()
}

// RateLimit enforces per-IP rate limiting (10 r/s, burst 20)
func RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == "/api/v1/auth/refresh" {
			next.ServeHTTP(w, r)
			return
		}
		ip := r.RemoteAddr
		v, ok := rateLimiterInstance.visitors.Load(ip)
		if !ok {
			limiter := rate.NewLimiter(10, 20)
			rateLimiterInstance.visitors.Store(ip, &visitor{limiter: limiter, lastSeen: time.Now()})
			v, _ = rateLimiterInstance.visitors.Load(ip)
		}
		vis := v.(*visitor)
		vis.lastSeen = time.Now()
		if !vis.limiter.Allow() {
			http.Error(w, "Превышен лимит запросов", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// UserRateLimit enforces per-user rate limiting (100 r/s, burst 20)
func UserRateLimit(next http.Handler) http.Handler {
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
		v, ok := userRateLimiterInstance.visitors.Load(userID)
		if !ok {
			limiter := rate.NewLimiter(100, 20)
			userRateLimiterInstance.visitors.Store(userID, &userVisitor{limiter: limiter, lastSeen: time.Now()})
			v, _ = userRateLimiterInstance.visitors.Load(userID)
		}
		vis := v.(*userVisitor)
		vis.lastSeen = time.Now()
		if !vis.limiter.Allow() {
			http.Error(w, "Превышен лимит запросов для пользователя", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
