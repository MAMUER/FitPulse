package middleware

import (
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
// Требование #12: Строгая Content Security Policy
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

		// Требование #12: Строгая Content Security Policy
		// Унифицировано с NGINX CSP (deploy/lb/nginx.conf)
		// style-src 'self' 'unsafe-inline' — допустимо, т.к. inline-styles не исполняют код
		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; "+
				"script-src 'self' https://cdn.jsdelivr.net; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:; "+
				"font-src 'self'; "+
				"connect-src 'self'; "+
				"media-src 'self'; "+
				"object-src 'none'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self'; "+
				"upgrade-insecure-requests;",
		)

		// Permissions Policy — запрет доступа к аппаратным средствам
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

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

		next.ServeHTTP(w, r)
	})
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
