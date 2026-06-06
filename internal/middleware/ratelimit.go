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

// rateLimiter manages rate limiting state
type rateLimiter struct {
	visitors sync.Map
}

// Package-level singleton initialized at startup
//
//nolint:gochecknoglobals
var rateLimiterInstance = &rateLimiter{}

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
		}
	}()
}

func RateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		ip := r.RemoteAddr
		v, ok := rateLimiterInstance.visitors.Load(ip)
		if !ok {
			limiter := rate.NewLimiter(10, 20) // 10 requests/sec, burst 20
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
