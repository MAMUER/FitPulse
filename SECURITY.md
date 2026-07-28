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

- **CSP**: строгая nonce-based политика для всех ответов (nonce генерируется через
  `crypto/rand`, 32 байта = 256 бит энтропии, кодируется стандартным base64) +
  `Referrer-Policy: strict-origin-when-cross-origin`,
  `Permissions-Policy: camera=(), microphone=(), geolocation=()`,
  `Cross-Origin-Opener-Policy: same-origin`,
  `Cross-Origin-Embedder-Policy: require-corp` для предотвращения cross-origin
  утечек и изоляции контекста.
  Атрибут `nonce` автоматически впрыскивается middleware `HTMLNonceInject` во все `<script>` теги HTML-ответа.
  Нарушения CSP логируются в ELK: директивы `report-uri /api/security/csp-report` и `report-to csp-endpoint` (`Report-To` header), обработчик `cspReportHandler` пишет структурированные `CSP_VIOLATION` события в zap.
  **Статус**: реализовано в `internal/middleware/security_headers.go` и `internal/middleware/nonce_inject.go`, эндпоинт `POST /api/security/csp-report` в `cmd/gateway`.
- **Subresource Integrity (SRI)**: не применяется. Все фронтенд-ресурсы (JS/CSS/шрифты) находятся локально в проекте (`/static/...`), внешние CDN отсутствуют. Подмена ресурсов исключается CSP nonce-based + логикой деплоя.
- **Rate limiting**: per-IP (10 r/s, burst 50), per-user (100 r/s, burst 200), sliding window; для auth endpoints отдельно: 5 attempts/minute per IP для `/login` и `/register` для защиты от brute-force атак (OWASP Authentication Cheat Sheet).
- **Маскировка версий**: NGINX `server_tokens off`, удаление заголовков Server/X-Powered-By
- **Обработка ошибок**: кастомные HTML-страницы, замена 403 на 404

### Безопасность данных

**At rest:**

- PostgreSQL: `pgsodium` (libsodium).
  Детерминированный AEAD `crypto_aead_det_encrypt` применяется только для полей, где требуется точный lookup без расшифровки (токены верификации).
  Для PII (email, full_name, nickname) используется рандомизированное шифрование + blind index (HMAC-индекс для поиска).
  Ключ импортируется в keyring `pgsodium.key` из `DB_ENCRYPTION_KEY` при старте `user-service` (`ensurePgsodiumKey`); legacy-данные, зашифрованные через `pgcrypto`, автоматически перекодируются (`reencryptPIIFromPgcrypto`).
  TOTP-секреты и refresh-токены носимых устройств — envelope encryption AES-256-GCM на уровне приложения (`internal/crypto`).
  Реализовано в `cmd/user-service/main.go`, `cmd/device-aggregator/main.go`, `internal/db/pgsodium.go`; схема — `db/migrations/V1__full_schema.sql`; образ БД заменён на `pgsodium/pgsodium:pg18`.
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

#### Принятые риски Trivy misconfiguration

В `trivy.yaml` определены исключения для правил, которые являются ложноположительными для специфичных рабочих нагрузок:

| Правило | Файл | Обоснование |
| ------- | ---- | ----------- |
| `KSV-0121` | `configs/monitoring/node-exporter/daemonset.yaml` | HostPath `/proc`, `/sys`, `/` необходимы node-exporter'у для сбора метрик хоста. Без них мониторинг невозможен. |
| `KSV-0010` | `configs/monitoring/node-exporter/daemonset.yaml` | `hostPID: true` требуется для доступа к `/proc/[pid]` всех процессов хоста. |
| `KSV-0009` | `configs/k8s/base/ingress-nginx/deployment.yaml` | `hostNetwork: true` необходим ingress-nginx на bare-metal/VPS для приёма трафика на порты 80/443 без внешнего балансировщика. |
| `KSV-0109` | `configs/k8s/base/ingress-nginx/configmap.yaml` | Ложноположительное: ключ `server-tokens: "false"` — это не секрет, а настройка скрытия версии nginx в заголовках ответа. |
| `KSV-0117` | `configs/k8s/base/ingress-nginx/deployment.yaml` | Принятый риск: ingress-nginx обязан слушать привилегированные порты 80/443 для обработки HTTP/HTTPS трафика. Без этого работа контроллера невозможна. |
| `KSV-01010` | `configs/k8s/base/ingress-nginx/configmap.yaml`, `configs/k8s/base/deployments/valkey-config.yaml`, `configs/k8s/base/configmap.yaml` | Ложноположительное: в ConfigMap только общедоступные параметры (HSTS, `valkey.conf`, порты). Секреты хранятся в Kubernetes Secrets, не в ConfigMap. |
| `KSV-0116` | `configs/k8s/base/local-path-provisioner.yaml`, `configs/monitoring/node-exporter/daemonset.yaml` | Принятый риск: оба компонента работают с root GID по дизайну. local-path-provisioner требует root для управления правами на хостовой ФС при создании PV. node-exporter требует root для доступа к `/proc`, `/sys` и `/host/root`. |
| `KSV-0105` | `configs/monitoring/node-exporter/daemonset.yaml` | Принятый риск: node-exporter по дизайну работает с UID 0 для доступа к хостовым `/proc`, `/sys` и `/host/root`. Без root доступ сборка метрик невозможна. |
| `KSV-0020` | `monitoring/grafana`, `monitoring/fluent-bit`, `jobs/seed-admin`, `jobs/migrate-db`, `deployments/rabbitmq-statefulset`, `deployments/postgres` | Официальные образы используют низкие UID по дизайну (Grafana 472, Fluent-bit 1000, PostgreSQL 999, RabbitMQ 1000, Flyway 1000). |
| `KSV-0021` | `monitoring/grafana`, `monitoring/fluent-bit`, `jobs/seed-admin`, `jobs/migrate-db`, `deployments/rabbitmq-statefulset`, `deployments/postgres` | Официальные образы используют низкие GID по дизайну (Grafana 472, Fluent-bit 1000, PostgreSQL 999, RabbitMQ 1000). |
| `KSV-0039` | `.trivyignore` (global ignore) | Глобально игнорируется: Trivy file scanner не сопоставляет LimitRange с workload'ами из разных файлов/директорий, хотя политики существуют для `fitness-platform-production`, `ingress-nginx` и `local-path-storage`. |
| `KSV-0040` | `.trivyignore` (global ignore) | Глобально игнорируется: Trivy file scanner не сопоставляет ResourceQuota с workload'ами из разных файлов/директорий, хотя политики существуют для `fitness-platform-production`, `ingress-nginx` и `local-path-storage`. |

### Kubescape — принятые исключения

| Правило | Манифест | Обоснование |
| ------- | -------- | ----------- |
| `C-0013` | `configs/k8s/base/local-path-provisioner.yaml` | Принятый риск: local-path-provisioner использует `busybox:1.37.0` (явный тег, не `:latest`) для helper pods. Контроль срабатывает из-за привилегированного доступа к хостовой ФС, который необходим для создания PersistentVolumes. |
| `C-0017` | `configs/k8s/base/local-path-provisioner.yaml` | Принятый риск: local-path-provisioner требует доступа к привилегированным портам хоста для управления директориями PersistentVolumes. |
| `C-0055` | `configs/k8s/base/local-path-provisioner.yaml` | Принятый риск: local-path-provisioner создает helper pods с elevated privileges для настройки прав на директориях хоста. Это стандартный паттерн для local-path provisioner безCSI-интеграции. |
| `C-0034` | `configs/k8s/base/local-path-provisioner.yaml` | Принятый риск: local-path-provisioner использует hostPath volumes для доступа к локальному хранилищу узлов. Без этого работа provisioner невозможна. |
| `C-0045` | `configs/k8s/base/local-path-provisioner.yaml` | Принятый риск: helper pods local-path-provisioner требуют привилегированного контекста безопасности для управления правами на хостовой ФС. |
| `C-0048` | `configs/k8s/base/local-path-provisioner.yaml` | Принятый риск: local-path-provisioner создает helper pods для настройки директорий на узлах. Это стандартный паттерн для storage provisioners без прямого доступа к hostPath. |
| `C-0056` | `configs/monitoring/fluent-bit/daemonset.yaml` | Принятый риск: fluent-bit требует доступа к `/var/log`, `/var/lib/docker/containers`, `/run/log` для сбора логов хоста и контейнеров. Без этого централизованное логирование невозможно. |
| `C-0018` | `configs/monitoring/fluent-bit/daemonset.yaml` | Принятый риск: fluent-bit запускается с привилегиями для чтения логов всех pods в namespace. Это необходимо для работы как cluster-wide logging agent. |
| `GHSA-qwww-vcr4-c8h2` | `web/package-lock.json` | Ложноположительное: уязвимость касается только unstable RSC API, которые не используются в проекте (нет директив `use server`/`use client` в `web/src`). |
| `KSV-0125` | `configs/monitoring/node-exporter/daemonset.yaml` | Ложноположительное: официальный образ `prom/node-exporter` из `docker.io` (trusted). |
| `KSV-0125` | `configs/monitoring/grafana/deployment.yaml` | Ложноположительное: официальный образ `grafana/grafana` из `docker.io` (trusted). |
| `KSV-0125` | `configs/monitoring/fluent-bit/daemonset.yaml` | Ложноположительное: официальный образ `fluent/fluent-bit` из `docker.io` (trusted). |
| `KSV-0125` | `configs/k8s/base/local-path-provisioner.yaml` | Ложноположительное: официальный образ `rancher/local-path-provisioner` из `docker.io` (trusted). |
| `KSV-0125` | `configs/monitoring/alertmanager/deployment.yaml` | Ложноположительное: официальный образ `prom/alertmanager` из `docker.io` (trusted). |
| `KSV-0125` | `configs/monitoring/prometheus/deployment.yaml` | Ложноположительное: официальный образ `prom/prometheus` из `docker.io` (trusted). |
| `KSV-0023` | `configs/monitoring/node-exporter/daemonset.yaml` | Принятый риск: HostPath `/proc`, `/sys`, `/` необходимы node-exporter'у для сбора метрик хоста. Без них мониторинг невозможен. |
| `KSV-0023` | `configs/monitoring/fluent-bit/daemonset.yaml` | Принятый риск: HostPath `/var/log`, `/var/lib/docker/containers`, `/run/log` необходимы fluent-bit'у для сбора логов хоста и контейнеров. Без них логирование невозможно. |
| `KSV-0012` | `configs/monitoring/node-exporter/daemonset.yaml` | Принятый риск: node-exporter требует root для доступа к `/proc` и `/sys` хоста. |
| `KSV-0012` | `configs/k8s/base/local-path-provisioner.yaml` | Принятый риск: local-path-provisioner требует root и `DAC_OVERRIDE` для управления правами на PersistentVolumes. |
| `KSV-0022` | `configs/k8s/base/local-path-provisioner.yaml` | Принятый риск: `DAC_OVERRIDE` необходим для управления правами на директории хоста при создании PersistentVolumes. |
| `KSV-0049` | `configs/k8s/base/local-path-provisioner.yaml` | Исправлено: в ClusterRole `local-path-provisioner-role` удалены лишние права `create`, `update`, `patch`, `delete` для configmaps. Provisioner только читает `local-path-config`. |
| `KSV-0048` | `configs/k8s/base/local-path-provisioner.yaml` | Принятый риск: local-path-provisioner создает helper pods для настройки директорий на узлах. Стандартный паттерн storage provisioner без прямого hostPath. |
| `KSV-0042` | `configs/k8s/base/local-path-provisioner.yaml` | Принятый риск: доступ к `pods/log` требуется для диагностики helper pods при создании PV. Без этого отладка проблем невозможна. |
| `KSV-0113` | `configs/k8s/base/rbac/rbac.yaml`, `configs/k8s/overlays/production/ingress-nginx-tls-role.yaml` | Принятый риск: сервисы читают secrets (`app-secrets`, `fittpulse-duckdns-org-tls`) и configmaps (`app-config`) в `fitness-platform-production`. Доступ ограничен `resourceNames` и verbs `get`/`list`/`watch`. |

### gosec — принятые исключения

| Правило | Файл | Обоснование |
| --------- | ------ | ------------- |
| `G101` | `cmd/gateway/main.go:203` | Ложноположительное: строки — публичные URL Google OAuth endpoints (`https://accounts.google.com/o/oauth2/auth`, `https://oauth2.googleapis.com/token`), известные всем разработчикам. Не являются credentials. |
| `G101` | `cmd/gateway/helpers.go:94` | Ложноположительное: ключи мапы — пользовательские сообщения об ошибках (gRPC status text), а не пароли/токены/секреты. |
| `G505` | `cmd/device-aggregator/providers/garmin.go:7` | Принятый риск: `crypto/sha1` требуется для HMAC-SHA1 подписи в OAuth 1.0a протоколе Garmin Health API. Это upstream-требование интеграции, а не слабое хеширование паролей. |

### CodeQL — логирование пользовательского ввода (go/log-injection)

Все точки логирования пользовательского ввода защищены функцией `sanitize.LogString()` из `internal/sanitize`, которая удаляет управляющие символы (переводы строк, возвраты каретки и другие ASCII 0x00-0x1F, 0x7F) для предотвращения инъекций в логи.

CodeQL не распознаёт кастомную функцию санитизации при межпакетном анализе, поэтому на уже защищённых строках добавлены комментарии `// CodeQL ignore: go/log-injection - sanitize.LogString() used`.

Защита реализована через `sanitize.LogString()` вместо игнорирования правила глобально. Комментарии добавлены только туда, где санитизация уже применена, но CodeQL не видит связь с пользовательским вводом. Это позволяет сохранить высокий уровень безопасности и не скрывать реальные уязвимости.

| Файл | Строки | Статус |
| --- | --- | --- |
| `cmd/device-aggregator/webhooks.go` | 43, 44, 91 | Исправлено: добавлен `sanitize.LogString()` для полей `type`, `user_id`, `action` из вебхуков Fitbit/Withings. |
| `cmd/classifier/main.go` | 500 | Исправлено: добавлен `sanitize.LogString()` для `r.URL.Path` в middleware логирования HTTP запросов. |
| `cmd/gateway/handlers_auth.go` | 216, 220 | Исправлено: добавлен `sanitize.LogString()` для `userID` и `r.URL.Path` при проверке критической сессии. |
| `cmd/gateway/handlers_csp_report.go` | 67-76 | Исправлено: добавлен `sanitize.LogString()` для всех полей CSP отчёта и заголовков запроса. |
| `internal/telemetry/http.go` | 23 | Исправлено: добавлен `sanitize.LogString()` для `r.URL.Path` в OpenTelemetry логировании. |
| `cmd/device-connector/main.go` | 181, 214-215, 239-240, 299-300, 374-375, 395-396 | Уже защищено `sanitize.LogString()`. Добавлены `// CodeQL ignore: go/log-injection` комментарии, так как CodeQL не распознаёт кастомный санитайзер. |

### Semgrep — безопасная установка инструментов CI

Запрещён паттерн `curl | bash` / `wget | sh` в GitHub Actions. Все установки выполняются через скачивание скрипта во временный файл с последующим выполнением и очисткой:

| Инструмент | Файл | Исправление |
| --- | --- | --- |
| Kubescape | `.github/workflows/ci.yml:464` | Заменено на `curl -fsSL ... -o /tmp/install-kubescape.sh && sudo bash /tmp/install-kubescape.sh` |
| Syft | `.github/workflows/ci.yml:756` | Заменено на `curl -fsSL ... -o /tmp/install.sh && bash /tmp/install.sh` |

### Инфраструктура

- **Сетевая сегментация**: Kubernetes Network Policies (dmz/app/data/monitoring)
- **RBAC**: минимальные права, отдельные ServiceAccount на сервис
  - gateway-sa, user-service-sa, biometric-service-sa, training-service-sa
  - device-connector-sa, classifier-sa, ml-generator-sa
  - app-service-account (для Jobs: migrate-db, seed-admin)
  - Каждая Role ограничена `resourceNames` на конкретные secrets и configmaps (например, `app-secrets`, `app-config`, `db-migrations`, `fittpulse-duckdns-org-tls`).
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
