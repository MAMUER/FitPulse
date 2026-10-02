# Yandex ID Verification

## 25. Production Verification Yandex ID

### 25.1 Текущий статус Phase 1

Yandex ID **уже реализован** в Phase 1:

- `cmd/gateway/handlers_auth.go:1150-1370` — handlers для `/auth/yandex`, `/auth/yandex/callback`
- `cmd/gateway/main.go:335-372` — регистрация routes
- Consent screen: **testing** (ограничение 100 пользователей, токены истекают через 7 дней)
- Scopes: `openid`, `profile`, `email` (минимальные)

### 25.2 Что требуется для Production

Чтобы разблокировать полноценный вход для всех пользователей через Yandex ID с production-статусом consent screen, требуется:

- подтверждённый домен, принадлежащий проекту;
- живой homepage на этом домене;
- privacy policy и terms of service на том же домене;
- branding verification пройдена.

**Связанные разделы**:

- Раздел 9 (Security Reporting) — OAuth integration требует готовой инфраструктуры
- Раздел 6 (Infrastructure as Code) — TLS через cert-manager, Ingress
- [DPA с Yandex ID](../compliance/DPA_YANDEX_ID.md) — трансграничная передача ПДн

### 25.3 Предпосылки

- DNS-записи домена указывают на VPS / внешний load balancer, где поднят кластер k3s.
- В Yandex Developer Console:
  - добавлен `fittpulse.duckdns.org` в **Authorized domains** для проекта `fitpulse-yandex-app`;
  - настроен OAuth 2.0 Client ID (Web application) с authorised redirect URIs:
    - `https://fittpulse.duckdns.org/api/v1/auth/yandex/callback`
  - в consent screen указаны:
    - Homepage: `https://fittpulse.duckdns.org`
    - Privacy Policy: `https://fittpulse.duckdns.org/privacy`
    - Terms of Service: `https://fittpulse.duckdns.org/terms`
- На VPS / в k8s:
  - cert-manager выписывает TLS-сертификат для `fittpulse.duckdns.org`;
  - ingress/routes проксируют `/`, `/privacy`, `/terms` на gateway;
  - SPA на React отдаёт главную страницу и юридические страницы.

### 25.4 Задачи

| Этап | Задача | Срок | Приоритет |
| --- | --- | --- | --- |
| 2 | Подготовить инфраструктуру: TLS через cert-manager, Ingress/Route для `fittpulse.duckdns.org` | 2–3 дня | P0 |
| 3 | Обновить `SECURITY.md`, CI конфиги, deployment manifests под новый домен | 1 день | P1 |
| 4 | В Yandex Developer Console обновить authorized domains, consent screen URLs, redirect URIs | 1 день | P0 |
| 5 | Пройти branding verification (логотип 120×120, скриншоты, описание) | 1–2 дня | P0 |
| 6 | Перевести consent screen из testing в production | 1 день | P0 |

### 25.5 Acceptance Criteria

- `https://fittpulse.duckdns.org`, `/privacy`, `/terms` доступны из внешней сети по HTTPS без авторизации.
- Yandex ID consent screen находится в статусе **production**.
- Вход через Yandex работает для любых пользователей без ограничения в 100 аккаунтов и без 7-дневного истечения токена.
- DPA с Yandex ID согласована и доступна: `docs/compliance/DPA_YANDEX_ID.md`

### 25.6 Риски и mitigation

| Риск | Вероятность | Воздействие | Mitigation |
| --- | --- | --- | --- |
| Домен не прошёл верификацию | Средняя | Высокое | Предварительно submit в Yandex, подготовить все материалы (логотип, скриншоты, описание) |
| Проблемы с DNS propagation | Средняя | Среднее | Использовать TTL 300s на время переезда, мониторить `dig`/`nslookup` |
| Просрочение сертификата cert-manager | Низкая | Низкое | cert-manager автоматически продлевает; настроить алерты за 7 дней до истечения |
| Трансграничная передача ПДн | Средняя | Высокое | DPA с Yandex, explicit consent пользователя, см. `docs/compliance/DPA_YANDEX_ID.md` |
