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

1. **Email**: Отправьте письмо на `mihnikolaenko12@yandex.ru` (будет заменён на `security@fitpulse.app` в Phase 2) или создайте приватный advisory в репозитории.
**Важно**: Для шифрования чувствительных данных используйте наш PGP-ключ (fingerprint будет опубликован в Phase 2).

2. **Информация для предоставления**:
   - Тип уязвимости (XSS, SQL Injection, CSRF, Authentication Bypass или иная)
   - Подробное описание шагов для воспроизведения
   - Версия FitPulse, где обнаружена уязвимость
   - Возможные последствия эксплуатации
   - Рекомендации по исправлению (если есть)

3. **Время ответа**:
   - Первоначальный ответ: в течение 48 часов
   - План исправления: в течение 7 дней
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

### Низкая опасность

- Missing rate limiting
- Verbose error messages
- Missing CSP directives

## Меры безопасности в FitPulse

### Аутентификация и авторизация

- **JWT**: ES256, access token TTL 15 минут
- **Refresh Token**: opaque, TTL 7 дней, rotation при каждом использовании
- **Хеширование паролей**: Argon2id (memory 64 MB, iterations 3, parallelism 1)
- **2FA**: TOTP (Google Authenticator) с backup-кодами
- **Сессии**: принудительная инвалидация при logout, отдельные хранилища для критических действий
- **Авторизация**: серверная проверка ролей через прямой запрос к БД

### Защита API

- **CSP**: строгая nonce-based политика для всех ответов
- **Rate limiting**: per-IP (10 r/s, burst 50), per-user (100 r/s, burst 200), sliding window; для auth endpoints отдельно: 5 attempts/minute per IP для `/login` и `/register` для защиты от brute-force атак (OWASP Authentication Cheat Sheet).
- **Маскировка версий**: NGINX `server_tokens off`, удаление заголовков Server/X-Powered-By
- **Обработка ошибок**: кастомные HTML-страницы, замена 403 на 404
- **Подпись ответов**: HMAC-SHA256 для критических JSON (login, register, profile, biometrics, plans).

### Безопасность данных

**At rest:**

- PostgreSQL: `pgcrypto` для чувствительных полей (PII, токены)
- Шифрование tablespace на уровне ОС (dm-crypt/LUKS для `/var/lib/rancher/k3s/storage`, настраивается через `configs/k8s/scripts/configure-storage-encryption.sh`; `storage-class-encrypted.yaml` для PVC)
- Резервные копии: AES-256

**In transit:**

- TLS 1.3 для всех внешних эндпоинтов (terminated на host Nginx)
- mTLS для внутренних gRPC-коммуникаций между микросервисами (Linkerd с встроенным mTLS или Istio + cert-manager)
- HSTS + OCSP Must-Staple + Certificate Transparency: Let's Encrypt сертификаты автоматически логируются в CT-логи; `ssl_trusted_certificate` configured в `deploy/lb/production.conf`; верификация CT и OCSP в CI/CD шаге "Verify Certificate Transparency and OCSP Stapling".
- L7 WAF:
  1. Host Nginx + ModSecurity (module `ngx_http_modsecurity_module.so`) + OWASP CRS v4 (`deploy/lb/modsecurity.conf`, rules in `/opt/modsecurity-crs/`). Включает правила для SQLi, XSS, request smuggling, кастомные исключения для `/health`. Устанавливается через `deploy/lb/install-crs.sh` в CI/CD (`provision-k8s-vps` job).
  2. In-cluster ingress-nginx c `enable-modsecurity` подготовкой в ConfigMap (`configs/k8s/base/ingress-nginx/configmap.yaml`). Фактический ModSecurity module требует кастомной сборки образа контроллера (`Phase 2`). Пока primary WAF остаётся host Nginx.

### CI/CD безопасность

- **SAST**: gosec
- **Vulnerability scanning**: govulncheck, Trivy
- **Secrets scanning**: TruffleHog, Gitleaks
- **Image signing**: cosign
- **SBOM generation**: syft (SPDX, CycloneDX)

### Инфраструктура

- **Сетевая сегментация**: Kubernetes Network Policies (dmz/app/data/monitoring)
- **RBAC**: минимальные права, отдельные ServiceAccount на сервис
  - gateway-sa, user-service-sa, biometric-service-sa, training-service-sa
  - device-connector-sa, classifier-sa, ml-generator-sa
  - Per-service Roles с минимальным набором разрешений (get configmaps/secrets)
- **Secrets**: JWT, API keys и TLS private keys.
- **WAF**:
  1. Host Nginx + ModSecurity (module `ngx_http_modsecurity_module.so`) + OWASP CRS v4 (`deploy/lb/modsecurity.conf`, rules in `/opt/modsecurity-crs/`). Включает правила для SQLi, XSS, request smuggling, кастомные исключения для `/health`. Устанавливается через `deploy/lb/install-crs.sh` в CI/CD (`provision-k8s-vps` job).
  2. In-cluster ingress-nginx (Namespace `ingress-nginx`, NodePort 30080) с `enable-modsecurity` подготовкой в ConfigMap (`configs/k8s/base/ingress-nginx/configmap.yaml`). Фактический ModSecurity module требует кастомной сборки образа контроллера (`Phase 2`). Пока primary WAF остаётся host Nginx.
- **Observability**: структурированное логирование (slog), Prometheus метрики, OpenTelemetry traces

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
