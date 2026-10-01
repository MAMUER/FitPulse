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
			if isOriginAllowed(cfg.AllowedOrigins, origin) {
				setCORSHeaders(w, cfg, origin)
			}
			if isPreflight(r.Method) {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isOriginAllowed(allowedOrigins []string, origin string) bool {
	for _, allowedOrigin := range allowedOrigins {
		if allowedOrigin == "*" || allowedOrigin == origin {
			return true
		}
	}
	return false
}

func setCORSHeaders(w http.ResponseWriter, cfg CORSConfig, origin string) {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	if cfg.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
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
}

func isPreflight(method string) bool {
	return method == http.MethodOptions
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
