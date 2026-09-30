# Google OAuth Verification

## 25. Production Verification Google OAuth

### 25.1 Текущий статус Phase 1

Google OAuth 2.0 **уже реализован** в Phase 1:

- `cmd/gateway/handlers_auth.go:1150-1370` — handlers для `/auth/google/login`, `/auth/google/callback`
- `cmd/gateway/main.go:335-372` — регистрация routes
- Consent screen: **testing** (ограничение 100 пользователей, токены истекают через 7 дней)
- Scopes: `openid`, `profile`, `email` (минимальные)

### 25.2 Что требуется для Production

Чтобы разблокировать полноценный вход для всех пользователей через Google OAuth 2.0 с production-статусом consent screen, требуется:

- подтверждённый домен, принадлежащий проекту;
- живой homepage на этом домене;
- privacy policy и terms of service на том же домене;
- branding verification пройдена.

**Связанные разделы**:

- Раздел 9 (Security Reporting) — OAuth integration требует готовой инфраструктуры
- Раздел 6 (Infrastructure as Code) — TLS через cert-manager, Ingress
- [DPA с Google OAuth](../compliance/DPA_GOOGLE_OAUTH.md) — трансграничная передача ПДн в США

### 25.3 Предпосылки

- DNS-записи домена указывают на VPS / внешний load balancer, где поднят кластер k3s.
- В Google Cloud Console:
  - добавлен `fittpulse.ru` в **Authorized domains** для проекта `fitpulse-1780824080979`;
  - настроен OAuth 2.0 Client ID (Web application) с authorised redirect URIs:
    - `https://fittpulse.ru/api/v1/auth/google/callback`
  - в consent screen указаны:
    - Homepage: `https://fittpulse.ru`
    - Privacy Policy: `https://fittpulse.ru/privacy`
    - Terms of Service: `https://fittpulse.ru/terms`
- На VPS / в k8s:
  - cert-manager выписывает TLS-сертификат для `fittpulse.ru`;
  - ingress/routes проксируют `/`, `/privacy`, `/terms` на gateway;
  - SPA на React отдаёт главную страницу и юридические страницы.

### 25.4 Задачи

| Этап | Задача | Срок | Приоритет |
| --- | --- | --- | --- |
| 2 | Подготовить инфраструктуру: TLS через cert-manager, Ingress/Route для `fittpulse.ru` | 2–3 дня | P0 |
| 3 | Обновить `SECURITY.md`, CI конфиги, deployment manifests под новый домен | 1 день | P1 |
| 4 | В Google Cloud Console обновить authorized domains, consent screen URLs, redirect URIs | 1 день | P0 |
| 5 | Пройти branding verification (логотип 120×120, скриншоты, описание) | 1–2 дня | P0 |
| 6 | Перевести consent screen из testing в production | 1 день | P0 |

### 25.5 Acceptance Criteria

- `https://fittpulse.ru`, `/privacy`, `/terms` доступны из внешней сети по HTTPS без авторизации.
- Google OAuth consent screen находится в статусе **production**.
- Вход через Google работает для любых пользователей без ограничения в 100 аккаунтов и без 7-дневного истечения токена.
- DPA с Google OAuth согласована и доступна: `docs/compliance/DPA_GOOGLE_OAUTH.md`

### 25.6 Риски и mitigation

| Риск | Вероятность | Воздействие | Mitigation |
| --- | --- | --- | --- |
| Домен не прошёл верификацию | Средняя | Высокое | Предварительно submit в Google, подготовить все материалы (логотип, скриншоты, описание) |
| Проблемы с DNS propagation | Средняя | Среднее | Использовать TTL 300s на время переезда, мониторить `dig`/`nslookup` |
| Просрочение сертификата cert-manager | Низкая | Низкое | cert-manager автоматически продлевает; настроить алерты за 7 дней до истечения |
| Трансграничная передача ПДн | Средняя | Высокое | DPA с Google, explicit consent пользователя, см. `docs/compliance/DPA_GOOGLE_OAUTH.md` |
