# FitPulse — Security Policy

## Поддерживаемые версии

| Версия | Поддержка          |
|--------|--------------------|
| 1.0.x  | :white_check_mark: |
| < 1.0  | :x:                |

## Сообщение об уязвимости

Мы принимаем сообщения об уязвимостях серьёзно. Если вы обнаружили уязвимость безопасности в FitPulse, пожалуйста, следуйте этой инструкции:

### Конфиденциальность

**Пожалуйста, не создавайте публичные issue для сообщений об уязвимостях.** Это позволит нам исправить уязвимость до того, как она станет известна злоумышленникам.

### Как сообщить

1. **Email**: Отправьте письмо на `mihnikolaenko12@yandex.ru` или создайте приватный advisory в репозитории.

2. **Информация для предоставления**:
   - Тип уязвимости (XSS, SQL Injection, CSRF, Authentication Bypass или иная)
   - Подробное описание шагов для воспроизведения
   - Версия FitPulse, где обнаружена уязвимость
   - Возможные последствия эксплуатации
   - Рекомендации по исправлению (если есть)

3. **Время ответа** (best effort, без юридических гарантий — проект поддерживается добровольцами без команды 24/7):
   - Первоначальный ответ: в течение 48 часов
   - План исправления: в течение 7 рабочих дней
   - Исправление: в течение 30 дней для критических уязвимостей

## Типы уязвимостей

### Критические

- Удалённое выполнение кода (RCE)
- SQL-инъекции с доступом к данным
- Аутентификация/авторизация bypass
- Утечка чувствительных данных (PII, пароли, токены)

### Высокая опасность

- XSS (Cross-Site Scripting)
- CSRF (Cross-Site Request Forgery)
- Недостатки контроля доступа
- Небезопасная десериализация

### Средняя опасность

- Missing security headers
- Weak cryptography
- Information disclosure
- Session management issues

#### Осознанные исключения: HMAC-SHA1 в Garmin OAuth 1.0a

В `cmd/device-aggregator/providers/garmin.go` используется `crypto/sha1` для формирования `oauth_signature` по спецификации **OAuth 1.0a**. Garmin Health API строго требует `oauth_signature_method: HMAC-SHA1`; замена алгоритма приведёт к отклонению всех запросов авторизации и синхронизации данных.

Использование SHA1 ограничено только подписью исходящих запросов к внешнему сервису Garmin. Криптографически значимые данные (пароли, refresh-токены, TOTP-секреты, PII) шифруются по современным стандартам: Argon2id, AES-256-GCM, pgsodium/libsodium.

Исключение зафиксировано в `.golangci.yml`:

```yaml
- path: cmd/device-aggregator/providers/garmin\.go
  linters:
    - gosec
  text: G505
```

При отключении интеграции с Garmin или переходе на их OAuth 2.0 (если появится) это исключение должно быть удалено.

### Низкая опасность

- Missing rate limiting
- Verbose error messages
- Missing CSP directives

## Меры безопасности в FitPulse

### Аутентификация и авторизация

- **JWT (Access Token)**: короткоживущий токен для аутентификации API. Реализована защита от replay attacks через короткое время жизни. Подробности реализации (алгоритм подписи, TTL, JWKS endpoint): [API Reference → Аутентификация](docs/API.md#аутентификация).
- **Refresh Token**: реализована Refresh Token Rotation и Reuse Detection для защиты от session hijacking. При попытке повторного использования отозванного токена вся сессия принудительно завершается.
- **Хеширование паролей**: Argon2id (memory 64 MB, iterations 3, parallelism 1)
- **2FA**: TOTP (стандарт RFC 6238) с резервными кодами восстановления
- **Сессии**: принудительная инвалидация при logout, отдельные хранилища для критических действий
- **Авторизация**: серверная проверка ролей через прямой запрос к БД

### Защита API

- **CSP**: строгая nonce-based политика для всех ответов (nonce генерируется через `crypto/rand`, 32 байта = 256 бит энтропии, кодируется стандартным base64) + `Referrer-Policy: strict-origin-when-cross-origin`, `Permissions-Policy: camera=(), microphone=(), geolocation=()`, `Cross-Origin-Opener-Policy: same-origin`, `Cross-Origin-Embedder-Policy: require-corp` для предотвращения cross-origin утечек и изоляции контекста. Атрибут `nonce` автоматически впрыскивается middleware `HTMLNonceInject` во все `<script>` теги HTML-ответа. Нарушения CSP логируются в ELK: директивы `report-uri /api/security/csp-report` и `report-to csp-endpoint` (`Report-To` header), обработчик `cspReportHandler` пишет структурированные `CSP_VIOLATION` события в zap. **Статус**: реализовано в `internal/middleware/security_headers.go` и `internal/middleware/nonce_inject.go`, эндпоинт `POST /api/security/csp-report` в `cmd/gateway`.
- **Subresource Integrity (SRI)**: не применяется. Все фронтенд-ресурсы (JS/CSS/шрифты) находятся локально в проекте (`/static/...`), внешние CDN отсутствуют. Подмена ресурсов исключается CSP nonce-based + логикой деплоя.
- **Rate limiting**: per-IP (10 r/s, burst 50), per-user (100 r/s, burst 200), sliding window; для auth endpoints отдельно: 5 attempts/minute per IP для `/login` и `/register` для защиты от brute-force атак (OWASP Authentication Cheat Sheet).
- **Маскировка версий**: NGINX `server_tokens off`, удаление заголовков Server/X-Powered-By
- **Обработка ошибок**: кастомные HTML-страницы, замена 403 на 404

### Безопасность данных

**At rest:**

- PostgreSQL: `pgsodium` (libsodium). Детерминированный AEAD `crypto_aead_det_encrypt` применяется только для полей, где требуется точный lookup без расшифровки (токены верификации). Для PII (email, full_name, nickname) используется рандомизированное шифрование + blind index (HMAC-индекс для поиска). Ключ импортируется в keyring `pgsodium.key` из `DB_ENCRYPTION_KEY` при старте `user-service` (`ensurePgsodiumKey`); legacy-данные, зашифрованные через `pgcrypto`, автоматически перекодируются (`reencryptPIIFromPgcrypto`). TOTP-секреты и refresh-токены носимых устройств — envelope encryption AES-256-GCM на уровне приложения (`internal/crypto`). Реализовано в `cmd/user-service/main.go`, `cmd/device-aggregator/main.go`, `internal/db/pgsodium.go`; схема — `db/migrations/V1__full_schema.sql`; образ БД заменён на `pgsodium/pgsodium:pg18`.
- Шифрование tablespace на уровне ОС (dm-crypt/LUKS для `/var/lib/rancher/k3s/storage`, настраивается через `configs/k8s/scripts/configure-storage-encryption.sh`; `storage-class-encrypted.yaml` для PVC)
- Резервные копии: AES-256

**In transit:**

- TLS 1.3 для всех внешних эндпоинтов (terminated на host Nginx)
- mTLS для внутренних gRPC-коммуникаций между микросервисами (TLS 1.3, mutual auth, сертификаты в Kubernetes Secret)
- HSTS + OCSP Stapling (`ssl_stapling on; ssl_stapling_verify on;`) + Certificate Transparency: Let's Encrypt сертификаты логируются в CT-логи; `ssl_trusted_certificate` и OCSP настроены в Ingress NGINX через cert-manager; верификация CT и OCSP в CI/CD шаге.
- L7 WAF: См. раздел "Инфраструктура" → "WAF"

### CI/CD безопасность

- **SAST**: gosec (глубокий анализ логики кода)
- **Vulnerability / Secrets / Misconfiguration scanning**: Trivy (единый сканер для репозитория `scan-type: fs` со `scanners: vuln,secret,misconfig` и для образов `scanners: vuln,secret`, плюс `scan-type: config` для IaC).
- **SBOM generation**: syft (SPDX, CycloneDX)
- **Image signing**: cosign

### Инфраструктура

- **Сетевая сегментация**: Kubernetes Network Policies (dmz/app/data/monitoring)
- **RBAC**: минимальные права, отдельные ServiceAccount на сервис
  - gateway-sa, user-service-sa, biometric-service-sa, training-service-sa
  - device-connector-sa, classifier-sa, ml-generator-sa
  - Per-service Roles с жестким ограничением `resourceNames` для чтения только специфичных секретов
- **Secrets**: JWT, API keys и TLS private keys хранятся в Kubernetes Secrets.
- **WAF**:
   1. Ingress NGINX Controller (`hostNetwork: true`, порты 80/443) + ModSecurity + OWASP CRS v4. Правила для SQLi, XSS, request smuggling, кастомные исключения для `/health`. Конфигурация в `configs/k8s/base/ingress-nginx/`. CRS rules автоматически обновляются через CronJob (`configs/k8s/base/jobs/update-modsecurity-crs.yaml`).
   2. cert-manager в кластере управляет TLS-сертификатами (Let's Encrypt). ClusterIssuer `letsencrypt-prod` для автоматического выпуска и продления сертификатов.
- **Observability**: структурированное логирование (zap), Prometheus метрики, OpenTelemetry traces

## Процесс исправления

1. **Получение отчёта** → Валидация и классификация
2. **Исследование** → Определение масштаба и влияния
3. **Исправление** → Разработка патча
4. **Тестирование** → Проверка исправления и регрессионное тестирование
5. **Релиз** → Выпуск обновления с указанием CVE (если применимо)
6. **Post-mortem** → Анализ инцидента и улучшение процессов

## Bug Bounty

FitPulse — бесплатный open-source проект без бюджета на вознаграждения.  
Программа Bug Bounty **не активна в денежном выражении**, но мы принимаем добровольные сообщения об уязвимостях и публично атрибутируем исследователей.

Мы благодарим исследователей за ответственное раскрытие уязвимостей.  

Подробности: scope, severity tiers, правила disclosure — в файле [BUG_BOUNTY_SCOPE.md](BUG_BOUNTY_SCOPE.md).

## Контакты

- **GitHub Security Advisory**: [Create a security advisory](https://github.com/MAMUER/fitpulse/security/advisories)
- **Email**: [mihnikolaenko12@yandex.ru](mailto:mihnikolaenko12@yandex.ru)

---

### Последнее обновление: 2026-07-01
