package middleware

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

// RemoveServerHeader удаляет заголовок Server
// Требование #5: Маскировка версий серверного ПО
func RemoveServerHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Del("Server")
		next.ServeHTTP(w, r)
	})
}

// SecurityHeaders добавляет заголовки безопасности
// Требование #5: Удаление информации о версии
// Требование #12: Строгая Content Security Policy (nonce-based)
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Требование #5: Удаляем все заголовки с версиями ПО
		w.Header().Del("Server")
		w.Header().Del("X-Powered-By")
		w.Header().Del("X-AspNet-Version")
		w.Header().Del("X-Go-Powered-By")

		// Защита от XSS в старых браузерах
		w.Header().Set("X-XSS-Protection", "1; mode=block")

		// Защита от MIME-sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Защита от кликджекинга
		w.Header().Set("X-Frame-Options", "DENY")

		// Политика реферера
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		nonce := generateNonce()
		ctx := r.Context()
		ctx = context.WithValue(ctx, nonceContextKey{}, nonce)

		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'nonce-"+nonce+"'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data: https:; "+
				"font-src 'self'; "+
				"connect-src 'self'; "+
				"media-src 'self'; "+
				"object-src 'none'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'; "+
				"report-to csp-endpoint; "+
				"report-uri /api/security/csp-report;",
		)

		// Permissions Policy — запрет доступа к аппаратным средствам
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		w.Header().Set("Cross-Origin-Embedder-Policy", "require-corp")

		// HSTS для HTTPS
		if r.TLS != nil {
			w.Header().Set(
				"Strict-Transport-Security",
				"max-age=63072000; includeSubDomains; preload",
			)
		}

		// Запрет кеширования для авторизованных страниц
		if r.Header.Get("Authorization") != "" {
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		}

		// Report-To: именованная конечная точка для CSP-нарушений (ELK ingestion)
		w.Header().Set("Report-To",
			`{"group":"csp-endpoint","max_age":31536000,"endpoints":[{"url":"/api/security/csp-report"}]}`,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type nonceContextKey struct{}

// GetNonce извлекает CSP nonce из контекста запроса
func GetNonce(r *http.Request) string {
	v, _ := r.Context().Value(nonceContextKey{}).(string)
	return v
}

func generateNonce() string {
	// 32 байта = 256 бит криптографически стойкой энтропии (требование: минимум 128 бит)
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand не должен падать; при фатальной ошибке лучше сломать запрос, чем nonce ""
		panic("middleware: failed to generate CSP nonce: " + err.Error())
	}
	return base64.StdEncoding.EncodeToString(b)
}

// LogoutHeaders добавляет заголовки для принудительной инвалидации сессии
// Требование #1: Явное указание браузеру на удаление cookies (session и refresh_token)
func LogoutHeaders() http.Header {
	h := make(http.Header)
	h.Add("Set-Cookie", "session=; Max-Age=0; Path=/; HttpOnly; Secure; SameSite=Strict")
	h.Add("Set-Cookie", "refresh_token=; Max-Age=0; Path=/; HttpOnly; Secure; SameSite=Strict")
	h.Set("Cache-Control", "no-store, no-cache, must-revalidate")
	h.Set("Pragma", "no-cache")
	return h
}
