package middleware

import (
	"net/http"
)

// CORSConfig содержит конфигурацию CORS.
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig возвращает конфигурацию CORS по умолчанию.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-Correlation-ID", "X-Captcha-Token"},
		MaxAge:         300,
	}
}

// CORS возвращает middleware для обработки CORS.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			allowed := false
			for _, allowedOrigin := range cfg.AllowedOrigins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				if cfg.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}

			if len(cfg.AllowedMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", joinStrings(cfg.AllowedMethods))
			}
			if len(cfg.AllowedHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", joinStrings(cfg.AllowedHeaders))
			}
			if cfg.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", http.StatusText(cfg.MaxAge))
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func joinStrings(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for i := 1; i < len(ss); i++ {
		result += ", " + ss[i]
	}
	return result
}
