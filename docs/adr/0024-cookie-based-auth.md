# ADR 0024: Миграция аутентификации на HttpOnly cookie

## Статус

Принято

## Контекст

До этого момента JWT токены хранились в `localStorage` фронтенда и передавались через заголовок `Authorization: Bearer`. Это создавало риск кражи токенов при XSS-атаке, так как любой JavaScript-код на странице имеет доступ к `localStorage`.

Для FitPulse это критично, потому что:
- система обрабатывает персональные данные и медицинскую информацию
- JWT токен даёт полный доступ к аккаунту на 15 минут
- refresh_token даёт доступ на 7 дней

## Решение

Перевести аутентификацию на HttpOnly/Secure/SameSite-Strict cookie:

1. **Access token cookie** (`fitpulse-access-token`):
   - `HttpOnly: true` — недоступен для JavaScript, защита от XSS
   - `Secure: true` в production, `false` в локальной разработке
   - `SameSite: Strict` — защита от CSRF
   - `MaxAge: 900` (15 минут)
   - `Path: /`

2. **Refresh token cookie** (`fitpulse-refresh-token`):
   - `HttpOnly: true`, `Secure`, `SameSite: Strict`
   - `MaxAge: 7 * 24 * 60 * 60` (7 дней)
   - `Path: /`

3. **Backend changes**:
   - `internal/middleware/middleware.go` — `AuthMiddleware` принимает токен из `Authorization: Bearer` или из cookie `fitpulse-access-token`
   - `cmd/gateway/handlers_auth.go` — login, 2FA verify, Google callback, refresh, logout handlers выдают/очищают cookie через `http.SetCookie`
   - `cmd/gateway/main.go` — helper `isSecureRequest(r)` определяет, использовать ли `Secure` флаг (по `r.TLS` или `X-Forwarded-Proto`)

4. **Frontend changes**:
   - `web/src/utils/backendRequest.js` — убран `localStorage`, добавлен `credentials: 'include'`, refresh через `POST /api/v1/auth/refresh` без тела
   - `web/src/services/api.js` — убрано чтение/запись токенов в `localStorage`
   - `web/src/hooks/useAuth.js` — проверки переведены с `data?.access_token` на `data?.status === 'ok'`

5. **Tests**:
   - `cmd/gateway/handlers_auth_test.go` — проверяет cookie `fitpulse-access-token` (HttpOnly, SameSite=Strict)
   - `cmd/gateway/handlers_test.go` — проверяет cookie в success сценарии

6. **Fallback**: токен также принимается из заголовка `Authorization: Bearer` для совместимости с legacy-клиентами и Postman.

## Последствия

- **Плюсы**: защита от XSS-украживания токенов; соответствие best practices для SPA с персональными данными.
- **Нейтрально**: требуется `credentials: 'include'` во всех fetch-запросах; CSRF защита через SameSite=Strict.
- **Риски**: при локальной разработке по HTTP cookie без `Secure` флага; смягчается через `isSecureRequest()`.

## Рассмотренные альтернативы

- **Оставить localStorage**: проще, но уязвимо к XSS.
- **httpOnly + Authorization Bearer**: частичная защита, но без автоматической отправки cookie.
- **Session в БД**: избыточно для JWT-based архитектуры.

## Реализация

- `internal/middleware/middleware.go`
- `cmd/gateway/handlers_auth.go`
- `web/src/utils/backendRequest.js`
- `web/src/services/api.js`
- `web/src/hooks/useAuth.js`
- `cmd/gateway/handlers_auth_test.go`
- `cmd/gateway/handlers_test.go`
- `docs/API.md`
- `docs/adr/0014-ui-specification.md`
