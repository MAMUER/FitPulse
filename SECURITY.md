# FitPulse — Security Policy

## Поддерживаемые версии

| Версия | Поддержка          |
|--------|--------------------|
| 2.0.x  | :white_check_mark: |
| < 2.0  | :x:                |

## Сообщение об уязвимости

Мы принимаем сообщения об уязвимостях серьёзно. Если вы обнаружили уязвимость безопасности в FitPulse, пожалуйста, следуйте этой инструкции:

### Конфиденциальность

**Пожалуйста, не создавайте публичные issue для сообщений об уязвимостях.** Это позволит нам исправить уязвимость до того, как она станет известна злоумышленникам.

### Как сообщить

1. **Email**: Отправьте письмо на `mihnikolaenko12@yandex.ru` или создайте приватный advisory в репозитории.

2. **Информация для предоставления**:
   - Тип уязвимости (XSS, SQL Injection, CSRF, Authentication Bypass)
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

### Высокие

- XSS (Cross-Site Scripting)
- CSRF (Cross-Site Request Forgery)
- Недостатки контроля доступа
- Небезопасная десериализация

### Средние

- Missing security headers
- Weak cryptography
- Information disclosure
- Session management issues

### Низкие

- Missing rate limiting
- Verbose error messages
- Missing CSP directives

## Меры безопасности в FitPulse

### Аутентификация и авторизация

- **JWT**: ES256, access token TTL 15 минут
- **Refresh Token**: opaque, TTL 7 дней, rotation при каждом использовании
- **Хеширование паролей**: Argon2id (memory 64 MB, iterations 3, parallelism 4). Согласно современным рекомендациям OWASP и RFC 9106, минимальный порог памяти для Argon2id должен составлять 64 МБ для устойчивости к GPU-атакам.
- **2FA**: TOTP (Google Authenticator) с backup-кодами
- **Сессии**: принудительная инвалидация при logout, отдельные хранилища для критических действий
- **Авторизация**: серверная проверка ролей через прямой запрос к БД

### Защита API

- **CSP**: строгая nonce-based политика для всех ответов
- **Rate limiting**: per-IP (10 r/s, burst 50), per-user (100 r/s, burst 200), sliding window
- **Маскировка версий**: NGINX `server_tokens off`, удаление заголовков Server/X-Powered-By
- **Обработка ошибок**: кастомные HTML-страницы, замена 403 на 404
- **Подпись ответов**: HMAC-SHA256 для критических JSON (login, register, profile, biometrics, plans). Ключи хранятся в Vault/AWS KMS с автоматической ротацией (см. runbooks/secrets.md).

### Безопасность данных

**At rest:**

- PostgreSQL: `pgcrypto` для чувствительных полей (PII, токены)
- Шифрование tablespace на уровне ОС (fsCrypt)
- Резервные копии: AES-256, ключ в Vault

**In transit:**

- TLS 1.3 для всех внешних эндпоинтов
- mTLS для внутренних gRPC-коммуникаций между микросервисами (Linkerd с встроенным mTLS или Istio + cert-manager)
- Certificate pinning неприменим для SPA в мобильном браузере. Вместо этого: HSTS + Certificate Transparency logs.

### CI/CD безопасность

- **SAST**: gosec
- **Vulnerability scanning**: govulncheck, Trivy
- **Secrets scanning**: TruffleHog, Gitleaks
- **Image signing**: cosign
- **SBOM generation**: syft (SPDX, CycloneDX)

### Инфраструктура

- **Сетевая сегментация**: Kubernetes Network Policies (dmz/app/data/monitoring)
- **RBAC**: минимальные права, отдельные ServiceAccount на сервис
- **Secrets**: хранение в Vault, автоматическая ротация каждые 90 дней
- **WAF**: Nginx + ModSecurity или managed WAF
- **Observability**: структурированное логирование (slog), Prometheus метрики, OpenTelemetry traces

## Процесс исправления

1. **Получение отчёта** → Валидация и классификация
2. **Исследование** → Определение масштаба и влияния
3. **Исправление** → Разработка патча
4. **Тестирование** → Проверка исправления и регрессионное тестирование
5. **Релиз** → Выпуск обновления с указанием CVE (если применимо)
6. **Post-mortem** → Анализ инцидента и улучшение процессов

## Bug Bounty

На данный момент программа Bug Bounty не активна. Мы благодарим исследователей за ответственное раскрытие уязвимостей.

## Контакты

- **GitHub Security Advisory**: [Create a security advisory](https://github.com/MAMUER/fitpulse/security/advisories)
- **Email**: [mihnikolaenko12@yandex.ru](mailto:mihnikolaenko12@yandex.ru)

---

### Последнее обновление: 2026-07-01
