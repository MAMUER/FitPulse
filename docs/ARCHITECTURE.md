# FitPulse — Полная архитектура и операционная документация

## Структура проекта

```text
.
├── api/
│   ├── proto/
│   │   ├── user.proto
│   │   ├── biometric.proto
│   │   ├── training.proto
│   │   └── ml.proto
│   └── gen/                          # сгенерированные .go файлы (committed в репозиторий)
├── cmd/
│   ├── gateway/                      # HTTP/gRPC gateway
│   ├── user-service/                 # Users, auth, profile
│   ├── biometric-service/            # Biometric data ingestion
│   ├── training-service/             # Training plans
│   ├── device-connector/             # External device sync (Fitbit/Garmin/Withings)
│   ├── device-aggregator/            # OAuth/webhook aggregator for devices
│   ├── classifier/                   # Classifier service
│   ├── ml_generator/                 # ML plan generator service (Python/FastAPI)
│   └── data-processor/               # Background data processing (in repo, not deployed standalone)
├── configs/
│   └── k8s/
│       ├── base/
│       │   ├── deployments/          # Deployment манифесты
│       │   ├── services/             # Service манифесты
│       │   ├── ingress-nginx/        # NGINX Ingress Controller
│       │   ├── configmap.yaml
│       │   ├── limit-range.yaml
│       │   ├── local-path-provisioner.yaml
│       │   ├── namespace.yaml
│       │   ├── resource-quota.yaml
│       │   ├── serviceaccount.yaml
│       │   ├── storage-class-encrypted.yaml
│       │   └── kustomization.yaml
│       ├── overlays/
│       │   └── production/
│       │       └── kustomization.yaml
│       └── scripts/                  # Helper scripts for k8s bootstrap
├── db/
│   └── migrations/                   # SQL миграции (версионированные)
│       └── V1__full_schema.sql       # Consolidated idempotent schema
├── docs/                             # Документация
├── internal/
│   ├── apperrors/                    # Application error types
│   ├── auth/                         # JWT, TOTP, refresh tokens
│   ├── biometric/                    # Domain, repository, service (biometric)
│   ├── cache/                        # Valkey cache abstraction
│   ├── config/                       # Configuration loader
│   ├── crypto/                       # Encryption utilities
│   ├── db/                           # Database connection, PGP encryption
│   ├── domain/                       # Shared domain models
│   ├── email/                        # Email sender, templates
│   ├── grpc/                         # gRPC server/client utilities, mTLS
│   ├── logger/                       # Structured logging
│   ├── metrics/                      # Prometheus metrics
│   ├── middleware/                    # HTTP middleware (auth, rate limit, etc.)
│   ├── queue/                        # RabbitMQ publisher/consumer
│   ├── repository/                   # Generic repositories
│   ├── sanitize/                     # HTML/XSS sanitization
│   ├── telemetry/                    # OpenTelemetry tracing
│   ├── totp/                         # TOTP 2FA
│   └── validator/                    # Request validators
├── models/                           # ML модели
├── scripts/                          # Вспомогательные скрипты
├── web/                              # SPA фронтенд
│   ├── index.html                    # Основное SPA (auth + views)
│   └── templates/
│       └── confirm.html              # Шаблон страницы подтверждения email
│   ├── static/
│       │   ├── css/
│       │   │   ├── main.css
│       │   │   └── modules.css
│       │   ├── fonts/
│       │   │   ├── fonts.css
│       │   │   └── *.woff2
│       │   ├── js/
│       │   │   ├── api.js
│       │   │   ├── app.js
│       │   │   └── modules.js
│       │   └── errors/               # HTML шаблоны ошибок
├── .github/
│   └── workflows/
│       └── ci.yml                   # Полный CI/CD пайплайн
├── Makefile
├── go.mod
├── go.sum
└── README.md
```

---

## 1. Компоненты инфраструктуры

### 1.1 Message Broker: RabbitMQ

```yaml
component: "RabbitMQ"
purpose: "Асинхронные сообщения между микросервисами"

use_cases:
  - "Очереди уведомлений (email, push)"
  - "Фоновая обработка биометрических данных"
  - "Событийная синхронизация между сервисами"

requirements:
  - "Durability: persistent queues + mirrored queues"
  - "Dead letter queues для обработки failed messages"
  - "Monitoring: queue depth, consumer lag, message rates"
```

**Конфигурация**:

- DLQ: `<queue-name>.dlq` для анализа ошибок (реализовано в `internal/queue/dlq.go`)
- TTL на сообщениях: 24 часа для сообщений уведомлений (реализовано в `internal/queue/dlq.go`)
- Persistent queues: `durable=true` при объявлении очередей
- Мониторинг: queue depth, consumer lag, message rates через Prometheus

### 1.2 Logging Stack: Fluent Bit

```yaml
component: "Fluent Bit"
purpose: "Сбор и форматирование логов из подов Kubernetes"

implementation:
  - "Fluent Bit DaemonSet на каждом узле (fluent/fluent-bit:2.2.2)"
  - "Сбор stdout/stderr контейнеров из /var/log/containers/*.log"
  - "Парсинг docker/json_logs контейнеров"
  - "Добавление Kubernetes метаданных (namespace, pod, container)"
  - "Вывод в stdout в формате JSON lines"

current_state:
  - "Fluent Bit DaemonSet развёрнут через configs/monitoring/fluent-bit/"
  - "Output: stdout только, без центрального хранилища"
  - "Kubernetes фильтр для обогащения логов метаданными"
  - "HTTP health endpoint на порту 2020"
```

**JSON-формат логов (обязательный)**:

```json
{
  "timestamp": "2026-04-02T14:30:00Z",
  "level": "INFO",
  "service": "biometric-service",
  "correlationId": "abc123-def456",
  "userId": "user-789",
  "action": "BIOMETRIC_DATA_RECEIVED",
  "durationMs": 125,
  "message": "Received 10 biometric records",
  "context": {
    "endpoint": "/api/v1/biometric",
    "method": "POST",
    "statusCode": 200,
    "userAgent": "FitnessApp/2.1.0",
    "ip": "192.168.1.100"
  }
}
```

### 1.3 Metrics Stack: Prometheus + Grafana

```yaml
component: "Prometheus + Grafana"
purpose: "Сбор, хранение и визуализация метрик"

implementation:
  - "Prometheus развёрнут в Kubernetes (configs/monitoring/prometheus/)"
  - "Grafana с provisioned дашборадами"
  - "Alertmanager с базовыми алертами (вебхук)"
  - "Service discovery через Kubernetes annotations"

current_state:
  - "Scrape configs настроены для всех сервисов"
  - "Alertmanager: вебхук на localhost:9093"
```

---

## 2. Матрица конфигураций по окружениям

|Параметр|Dev|Test|Staging|Prod|
|---|---|---|---|---|
|**K8s pods per service**|1|1|1–2|1–3 (HPA при нагрузке)|
|**PostgreSQL topology**|1 инстанс (postgres:18-alpine)|1 primary|1 primary + 1 replica|1 primary + 1–2 replicas (postgres:18 + pgsodium:pg18)|
|**Valkey topology**|1 узел (valkey:9-alpine)|1 узел (standalone)|1 узел (standalone)|1 узел (standalone, Sentinel Phase 2)|
|**RabbitMQ**|1 узел (rabbitmq:4.3-management-alpine, classic queues)|1 узел|1 узел|1 узел (quorum queues Phase 2)|
|**GPU resources**|CPU only|CPU only|CPU only|CPU only (ML inference на CPU)|
|**Monitoring stack**|Console logs|Prometheus + Grafana|Prometheus + Grafana + Alertmanager|Prometheus + Grafana + Alertmanager (Slack Phase 2)|
|**Backup strategy**|Нет|Еженедельно (pg_dump)|Ежедневно (pg_dump)|Ежедневно (pg_dump) + WAL (Phase 2)|
|**SSL/TLS**|Self-signed|Self-signed / Let's Encrypt|Let's Encrypt (авто-ротация)|Let's Encrypt / Corporate CA|
|**Access control**|Локальный доступ|VPN|VPN + 2FA (TOTP)|2FA + IP whitelist|

---

## 3. Наблюдаемость: логи, метрики, алерты

### 3.1 Обязательные поля логирования

Все сервисы должны логировать в следующем формате:

|Поле|Тип|Описание|
|---|---|---|
|`timestamp`|ISO8601|Время события в UTC|
|`level`|enum|DEBUG/INFO/WARN/ERROR/FATAL|
|`service`|string|Имя микросервиса|
|`correlationId`|UUID|ID для трассировки запроса по сервисам|
|`userId`|string\|null|ID пользователя (если аутентифицирован)|
|`action`|string|Семантическое имя действия (UPPER_SNAKE_CASE)|

### 3.2 Prometheus-метрики (обязательный набор)

```yaml
prometheus_metrics:
  - name: "request_duration_seconds"
    type: "Histogram"
    labels: ["service", "endpoint", "method", "status"]
    buckets: [0.1, 0.25, 0.5, 1, 2.5, 5, 10]
  
  - name: "error_total"
    type: "Counter"
    labels: ["service", "error_code", "endpoint"]
  
  - name: "classification_confidence"
    type: "Gauge"
    labels: ["model_version", "class"]
    description: "Уверенность ML-модели в определении состояния пользователя"
  
  - name: "db_connection_pool_usage"
    type: "Gauge"
    labels: ["service", "pool_name"]
  
  - name: "notification_queue_depth"
    type: "Gauge"
    labels: ["queue_name", "priority"]
  
  - name: "biometric_sync_lag_seconds"
    type: "Gauge"
    labels: ["device_type", "user_segment"]
    description: "Задержка между получением данных с устройства и обработкой"
```

### 3.3 Минимальный набор алертов

#### Критические (SEV-1)

|Алерт|Условие|Каналы|
|---|---|---|
|`ServiceDown`|`up{job=~'fitness-.*'} == 0` за 2 мин|Slack + Grafana OnCall|
|`DBConnectionPoolExhausted`|`db_connection_pool_usage > 0.9` за 1 мин|Grafana OnCall|
|`BackupFailed`|`backup_success{type='full'} == 0`|Grafana OnCall|

#### Предупреждения (SEV-3)

|Алерт|Условие|Каналы|
|---|---|---|
|`HighErrorRate`|`rate(error_total[5m]) / rate(request_total[5m]) > 0.01` за 5 мин|Slack|
|`HighLatency`|`histogram_quantile(0.95, ...) > 5` за 10 мин|Slack|
|`LowMLConfidence`|`classification_confidence < 0.7` за 15 мин|Slack|

**Политика эскалации**:

- `SEV-1`: немедленно → Grafana OnCall → on-call engineer → Tech Lead → CTO
- `SEV-2`: 15 мин → Slack → on-call engineer → Tech Lead
- `SEV-3`: 1 час → Slack → on-call engineer
- `SEV-4`: 24 часа → ticket queue

---

## 4. Безопасность развертывания и управление обновлениями

### 4.1 Сетевая сегментация (Network Policies)

```yaml
zones:
  dmz:
    description: "Внешний трафик (Ingress, WAF)"
    allowed_ingress: ["internet"]
    allowed_egress: ["app-zone"]
  
  app-zone:
    description: "Микросервисы приложения"
    allowed_ingress: ["dmz", "monitoring-zone"]
    allowed_egress: ["data-zone", "monitoring-zone"]
  
  data-zone:
    description: "БД, кэш, очереди"
    allowed_ingress: ["app-zone"]
    allowed_egress: ["none"]
  
  monitoring-zone:
    description: "Prometheus, Grafana, Alertmanager"
    allowed_ingress: ["vpn-users"]
    allowed_egress: ["all"]

verification:
  - "NetworkPolicy audit: kube-bench, kube-hunter"
  - "Penetration test: изоляция зон, попытки lateral movement"
```

### 4.2 RBAC и привилегии

```yaml
implementation: "Kubernetes RBAC + ServiceAccount per service"

principles:
  - "Principle of least privilege: каждый сервис имеет минимальные права"
  - "No cluster-admin для приложений"
  - "Separate ServiceAccount для CI/CD и runtime"

verification:
  - "Audit RBAC: kubectl auth can-i --list"
  - "CIS Kubernetes Benchmark via kube-bench"
```

### 4.3 Шифрование

**At rest**:

- PostgreSQL: `pgcrypto` для чувствительных полей (PII, токены) + шифрование tablespace на уровне ОС (dm-crypt/LUKS)
- Volumes: шифрование

**In transit**:

- TLS 1.3 минимум для всех внешних эндпоинтов
- mTLS для gRPC-коммуникации между микросервисами (TLS 1.3, сертификаты в Kubernetes Secret)
- HSTS + OCSP Stapling + CT logs (pinning не применяется для SPA)

### 4.4 Управление зависимостями

|Инструмент|Функция|
|---|---|
|Dependabot|Еженедельный скан, авто-PR для минорных обновлений|
|Snyk|Интеграция в CI/CD, блокировка мержа при critical CVE|

**Политики** (best effort, без юридических гарантий):

- Critical CVE: патч в течение 1–3 рабочих дней
- High CVE: патч в течение 3–7 рабочих дней
- Запрет на использование пакетов с известными уязвимостями (blacklist)

### 4.5 Аудит администраторов

```yaml
implementation: "Audit Service + ELK"

logged_actions:
  - "Login to production environment"
  - "Database schema changes"
  - "Secret rotation / credential access"
  - "Deployment / rollback operations"
  - "RBAC policy changes"

retention: "1 год (соответствие 152-ФЗ)"
access: "Только роль 'auditor', read-only"

verification:
  - "Ежеквартальный review логов аудита"
  - "Compliance check: соответствие внутренним политикам"
```

### 4.6 WAF (Web Application Firewall)

**Текущая реализация**:

- Ingress NGINX Controller с `hostNetwork: true` на портах 80/443
- ModSecurity + OWASP CRS v4 (managed via ConfigMap)
- cert-manager для автоматического управления TLS-сертификатами (Let's Encrypt)
- Automated CRS updates через Kubernetes CronJob (еженедельно)

**Правила**:

- SQL injection, XSS, path traversal блокировка
- Rate limiting: 100 req/min per IP для анонимных пользователей
- Geo-blocking: доступ только из разрешённых регионов (опционально)
- Health checks bypass WAF (`/health` endpoint)

### 4.7 Ротация секретов

```yaml
policy:
  - "Динамические секреты для БД: короткоживущие креды"
  - "Сервисы получают новые ключи без перезапуска (hot reload)"

verification:
  - "Compliance check: аудит политик"
  - "Тест отката: восстановление работы при компрометации ключа"
```

---

## 4.x User Service (user-service)

### 4.x.1 Назначение

gRPC-сервис для управления пользователями, аутентификацией и профилями. Отвечает за:
- Регистрацию и подтверждение email
- Логин по паролю (Argon2id) и Google OAuth
- Выдачу JWT access/refresh токенов
- Управление профилями, целями и противопоказаниями
- 2FA через TOTP с резервными кодами
- Управление устройствами пользователя
- Хранение состава тела, менструальных циклов, состояний здоровья
- Пригласительные коды для регистрации тренеров/клиентов
- Шифрование PII через pgsodium AEAD

### 4.x.2 gRPC методы

| RPC | описание |
|---|---|
| `Register` | Регистрация пользователя |
| `RegisterWithInvite` | Регистрация по invite-коду |
| `ConfirmEmail` | Подтверждение email |
| `Login` | Логин по паролю |
| `AuthenticateGoogle` | Авторизация через Google |
| `RefreshToken` | Обновление access token |
| `GetProfile` | Получить профиль |
| `GetUserByEmail` | Получить пользователя по email |
| `UpdateProfile` | Обновить профиль |
| `ChangePassword` | Сменить пароль |
| `ChangeNickname` | Сменить никнейм |
| `UploadProfilePhoto` | Загрузить фото профиля |
| `RemoveProfilePhoto` | Удалить фото профиля |
| `ListDevices` | Список устройств |
| `AddDevice` | Добавить устройство |
| `RemoveDevice` | Удалить устройство |
| `SyncDeviceData` | Синхронизировать данные устройства |
| `GetTrainingStats` | Статистика тренировок |
| `GetAchievements` | Достижения пользователя |
| `ListUsers` | Список пользователей (пагинация) |
| `ValidateInviteCode` | Проверить invite-код |
| `SetupTOTP` | Настроить 2FA |
| `ConfirmTOTP` | Подтвердить включение 2FA |
| `VerifyTOTP` | Проверить TOTP код |
| `DisableTOTP` | Отключить 2FA |
| `ListHealthConditions` | Список состояний здоровья |
| `UpsertHealthCondition` | Создать/обновить состояние здоровья |
| `DeleteHealthCondition` | Удалить состояние здоровья |
| `ListBodyComposition` | Список записей состава тела |
| `CreateBodyComposition` | Создать запись состава тела |
| `ListMenstrualCycles` | Список менструальных циклов |
| `CreateMenstrualCycle` | Создать менструальный цикл |
| `UpdateMenstrualCycle` | Обновить менструальный цикл |
| `DeleteMenstrualCycle` | Удалить менструальный цикл |
| `SyncFloData` | Синхронизация с Flo |
| `SyncOKOKData` | Синхронизация с OKOK |

### 4.x.3 Конфигурация

| Переменная | Default | Описание |
|---|---|---|
| `USER_SERVICE_PORT` | `50051` | Порт gRPC сервера |
| `USER_SERVICE_METRICS_PORT` | `9096` | Порт metrics-сервера |
| `DB_HOST`, `DB_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `DB_SSLMODE` | — | PostgreSQL подключение |
| `JWT_PRIVATE_KEY_PEM` | — | PEM-ключ для JWT |
| `TOTP_ENCRYPTION_KEY` | — | Ключ для шифрования TOTP секретов |
| `BASE_URL` | `https://localhost:8443` | Базовый URL для ссылок верификации |
| `GOOGLE_CLIENT_ID` | — | Google OAuth Client ID |
| `DB_ENCRYPTION_KEY` | — | Ключ для pgsodium PII шифрования |

### 4.x.4 Безопасность

- Пароли: Argon2id (m=65536, t=3, p=1), salt 16 байт, hash 32 байта
- PII шифрование: pgsodium AEAD (`email_encrypted`, `full_name_encrypted`, `nickname_encrypted`, `token_encrypted`, `totp_secret_encrypted`)
- Blind indexes: `email_hash`, `full_name_hash`, `nickname_hash` для поиска по зашифрованным полям
- JWT: ES256, 15 минут access, 7 дней refresh
- 2FA: TOTP (10 резервных кодов, хешированных через SHA256)
- Google OAuth: автоматическая привязка/создание пользователя
- Email верификация: токены с сроком 24 часа

### 4.x.5 Graceful Shutdown

- `signal.NotifyContext` для SIGINT/SIGTERM
- Graceful shutdown gRPC сервера (`GracefulStop`) и metrics-сервера (таймаут 10 секунд)

### 4.x.6 Метрики

- gRPC server interceptor: `metrics.UnaryServerInterceptor("user-service")`
- HTTP endpoint: `:9096/metrics` (Prometheus `promhttp.Handler`)

### 4.x.7 Middleware

- gRPC recovery interceptor (`middleware.RecoveryGRPC`)
- Correlation ID interceptor (`middleware.CorrelationIDGRPC`)
- Telemetry interceptor (`telemetry.ServerHandlerOption`)

### 4.x.8 Особенности

- Логгер: `internal/logger` с полем `service: "user-service"`
- PII миграция: автоматическое перекодирование pgcrypto → pgsodium при старте
- Backfill PII: заполнение шифрованных полей для существующих plaintext записей
- Транзакционная целостность: `CreateMenstrualCycle` и `UpdateMenstrualCycle` используют транзакции
- Invite-коды: хранятся в БД, поддерживают role/specialty/max_uses

### 4.x.9 Интеграционные тесты

Пропущены (`t.Skip`). Запуск: `go test ./cmd/user-service/...`

---

## 4.x Биометрический сервис (biometric-service)

### 4.x.1 Назначение

gRPC-сервис для приёма, валидации, дедупликации и хранения биометрических данных (пульс, SpO2, температура, артериальное давление, шаги, HRV). Публикует события в RabbitMQ для асинхронной ML-обработки.

### 4.x.2 gRPC-авторизация

Все методы требуют JWT access token (ES256) в gRPC metadata:

```text
authorization: Bearer <access_token>
```

Interceptor: `middleware.GRPCAuthInterceptor` (`internal/middleware/grpc_auth.go`).
Токен валидируется по JWKS публичному ключу из `JWT_PUBLIC_KEY_PEM_FILE`.

### 4.x.3 Health Check

Динамический health check раз в 10 секунд:
- Пингует PostgreSQL (`db.PingContext`)
- Пингует RabbitMQ (`queue.Publisher.Ping()`)
- gRPC health protocol возвращает `SERVING` / `NOT_SERVING`

### 4.x.4 Метрики

- gRPC interceptor: `metrics.UnaryServerInterceptor("biometric-service")` — `grpc_requests_total`, `grpc_request_duration_seconds`, `grpc_errors_total`
- HTTP endpoint: `:9090/metrics` (Prometheus `promhttp.Handler`)
- Бизнес-метрики: `biometric_sync_lag_seconds`

### 4.x.5 Дедупликация

Уникальное ограничение на `(user_id, metric_type, timestamp, device_type)`.
Миграция: `db/migrations/V1__full_schema.sql`.
Вставки используют `ON CONFLICT DO NOTHING`.

### 4.x.6 Валидация метрик

| metric_type | диапазон |
|---|---|
| `heart_rate` | 30–220 |
| `spo2` | 70–100 |
| `temperature` | 35.5–38.5 °C |
| `blood_pressure_systolic` | 80–200 |
| `blood_pressure_diastolic` | 50–130 |
| `steps` | 0–100000 |
| `hrv` | 0–200 |

### 4.x.7 gRPC методы

| RPC | описание |
|---|---|
| `AddRecord` | Добавить одну запись |
| `BatchAddRecords` | Пакетная вставка с транзакцией |
| `GetRecords` | Получить записи с фильтрацией по `from`/`to` и пагинацией `limit`/`offset` |
| `GetLatest` | Последняя запись по типу |
| `UpdateRecord` | Обновить запись по `id` |
| `DeleteRecord` | Удалить запись по `id` |

### 4.x.8 Интеграционные тесты

Используют Testcontainers (PostgreSQL + RabbitMQ) через `internal/testcontainers`.
Запуск: `go test ./cmd/biometric-service/...` (без `-short`).

---

## 4.x Training Service (training-service)

### 4.x.1 Назначение

gRPC-сервис для управления тренировочными планами. Отвечает за:
- Генерацию персонализированных планов тренировок на основе классификации состояния пользователя
- Хранение планов в PostgreSQL (`training_plans`, `training_plan_weeks`, `training_plan_days`, `training_exercises`)
- Отслеживание выполнения тренировок (`workout_completions`)
- Начисление достижений (`user_achievements`)
- Публикацию событий о генерации планов в RabbitMQ

### 4.x.2 gRPC методы

| RPC | описание |
|---|---|
| `GeneratePlan` | Сгенерировать тренировочный план |
| `GetPlan` | Получить план по ID |
| `ListPlans` | Список планов пользователя |
| `CompleteWorkout` | Отметить тренировку выполненной |
| `GetProgress` | Прогресс пользователя |

### 4.x.3 Конфигурация

| Переменная | Default | Описание |
|------------|---------|----------|
| `TRAINING_SERVICE_PORT` | `50053` | Порт gRPC сервера |
| `TRAINING_SERVICE_METRICS_PORT` | `9095` | Порт metrics-сервера |
| `DB_HOST`, `DB_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `DB_SSLMODE` | — | PostgreSQL подключение |
| `RABBITMQ_URL` | — | RabbitMQ URL (опционально) |
| `BIOMETRIC_SERVICE_ADDR` | — | Адрес biometric-service (не используется напрямую) |

### 4.x.4 Генерация плана

`GeneratePlan` выполняет:
1. Валидацию запроса
2. Удаление существующего активного плана пользователя
3. Подготовку данных плана из запроса
4. Расчёт дат начала и конца
5. Сохранение плана и деталей в транзакции PostgreSQL
6. Публикацию события `plan_generated` в RabbitMQ

### 4.x.5 Достижения

Автоматическое начисление достижений при выполнении тренировок:
- `first_workout` — после 1 выполненной тренировки
- `ten_workouts` — после 10 выполненных тренировок
- `fifty_workouts` — после 50 выполненных тренировок

### 4.x.6 Graceful Shutdown

- `signal.NotifyContext` для SIGINT/SIGTERM
- Graceful shutdown gRPC сервера (`GracefulStop`) и metrics-сервера (таймаут 10 секунд)

### 4.x.7 Метрики

- gRPC server interceptor: `metrics.UnaryServerInterceptor("training-service")`
- HTTP endpoint: `:9095/metrics` (Prometheus `promhttp.Handler`)

### 4.x.8 Middleware

- gRPC recovery interceptor (`middleware.RecoveryGRPC`)
- Correlation ID interceptor (`middleware.CorrelationIDGRPC`)
- Telemetry interceptor (`telemetry.ServerHandlerOption`)

### 4.x.9 Интеграционные тесты

Пропущены (`t.Skip`). Запуск: `go test ./cmd/training-service/...`

### 4.x.10 Особенности

- Логгер: `internal/logger` с полем `service: "training-service"`
- Транзакционная целостность: `GeneratePlan` использует единую транзакцию для плана и деталей
- RabbitMQ опционален: сервис работает без очереди, но логирует предупреждение

---

## 4.x Классификатор состояний (classifier)

### 4.x.1 Назначение

HTTP-сервис для классификации физиологического состояния пользователя по 6 зонам на основе биометрических данных (пульс, HRV, SpO2, температура, АД, сон). Использует rule-based модель (замена реальной ML-модели в Phase 1). Gateway вызывает его для `POST /api/v1/ml/classify`.

### 4.x.2 Endpoints

| Endpoint | Назначение |
|----------|-----------|
| `POST /classify` | Классификация состояния |
| `GET /health` | Health check |
| `GET /metrics` | Prometheus метрики |
| `GET /classes` | Список поддерживаемых классов |
| `GET /model-info` | Информация о модели |

### 4.x.3 Конфигурация

| Переменная | Default | Описание |
|------------|---------|----------|
| `CLASSIFIER_PORT` | `8001` | Порт сервера |
| `CLASSIFIER_METRICS_PORT` | `9091` | Порт metrics-сервера |

### 4.x.4 Формат запроса `POST /classify`

```json
{
  "physiological_data": {
    "heart_rate": 140.0,
    "heart_rate_variability": 50.0,
    "spo2": 98.0,
    "temperature": 36.6,
    "blood_pressure_systolic": 120.0,
    "blood_pressure_diastolic": 80.0,
    "sleep_hours": 7.0
  },
  "user_profile": {
    "age": 30,
    "fitness_level": "intermediate",
    "goals": ["endurance"]
  }
}
```

### 4.x.5 Формат ответа

```json
{
  "status": "success",
  "state": "endurance_basic",
  "confidence": 0.87,
  "recommendation": ["Бег в аэробной зоне", "Велосипед (средняя интенсивность)"],
  "fatigue_level": 0.3,
  "motivation_score": 0.7,
  "recovery_quality": 0.7,
  "predicted_class": "endurance_basic",
  "predicted_class_ru": "Базовая выносливость E1-E2",
  "probabilities": {
    "recovery": 0.02,
    "endurance_basic": 0.87,
    "endurance_threshold": 0.02,
    "power_hiit": 0.02,
    "overtraining": 0.05,
    "illness": 0.02
  },
  "description": "Работа ниже лактатного порога...",
  "hr_range": "65-80% HRmax",
  "personalized_notes": "Учитывая цель похудения..."
}
```

### 4.x.6 Классы состояний

| # | Класс (slug) | Название RU | Ключевые правила |
|---|--------------|-------------|------------------|
| 0 | `recovery` | Восстановление | HRV > 50 И HR < 65% HRmax |
| 1 | `endurance_basic` | Базовая выносливость E1-E2 | HR 65–80% HRmax, HRV 50–80 |
| 2 | `endurance_threshold` | Пороговая выносливость E3 | HR 80–90% HRmax |
| 3 | `power_hiit` | Силовая/HIIT | HR > 90% HRmax |
| 4 | `overtraining` | Перетренированность | HRV < 30 И HR < 60% HRmax |
| 5 | `illness` | Заболевание | Температура > 37.5°C |

### 4.x.7 Middleware

- Recovery middleware (`middleware.RecoveryMiddleware`)
- Request ID (`middleware.RequestID`)
- CORS (`corsMiddleware`)
- Логирование с Prometheus-метриками (`classifierLoggingMiddleware`)

### 4.x.8 Graceful Shutdown

- `signal.NotifyContext` для SIGINT/SIGTERM
- Graceful shutdown основного и metrics серверов (таймаут 10 секунд)

### 4.x.9 Метрики

- `http_request_duration_seconds{method, path}`
- `http_requests_total{method, path, status}`
- `error_total{service="classifier", error_type}`
- `classification_confidence{model_version="rule-based", class}`

### 4.x.10 Валидация

Валидация входных данных с диапазонами:
- `heart_rate`: 20–250
- `heart_rate_variability`: 0–300
- `spo2`: 70–100
- `temperature`: 30–45
- `blood_pressure_systolic`: 60–250
- `blood_pressure_diastolic`: 40–150
- `sleep_hours`: 0–24

Нулевые значения считаются не указанными и пропускаются.

### 4.x.11 Интеграционные тесты

Реальные e2e-тесты с поднятым HTTP-сервером:
- Health check
- Классификация с валидными данными
- Обработка невалидного JSON
- Валидационные ошибки

Запуск: `go test ./cmd/classifier/...` (без `-short`).

### 4.x.12 Особенности

- Логгер: `internal/logger` с полем `service: "classifier"`
- Gateway трансформирует ответ в контракт API: добавляет `status`, `state`, `fatigue_level`, `motivation_score`, `recovery_quality`
- Mapping метрик: `hrv` → `heart_rate_variability`, `systolic_pressure` → `blood_pressure_systolic` и т.д.

---

## 4.x Device Aggregator (device-aggregator)

### 4.x.1 Назначение

HTTP-сервис для управления OAuth-подключениями носимых устройств (Fitbit, Garmin, Withings). Отвечает за:
- OAuth 2.0 flow для Fitbit и Withings
- OAuth 1.0a flow для Garmin
- Шифрование и хранение refresh-токенов в БД
- Управление подключениями (подключение/отключение)
- Обработка webhook-уведомлений от провайдеров
- Список подключённых провайдеров для пользователя

### 4.x.2 Endpoints

| Endpoint | Назначение |
|----------|-----------|
| `GET /health` | Health check |
| `GET /metrics` | Prometheus метрики |
| `GET /api/v1/devices/fitbit/auth` | Start Fitbit OAuth |
| `GET /api/v1/devices/fitbit/callback` | Fitbit OAuth callback |
| `POST /api/v1/devices/fitbit/webhook` | Fitbit webhook |
| `POST /api/v1/devices/fitbit/disconnect` | Disconnect Fitbit |
| `GET /api/v1/devices/garmin/auth` | Start Garmin OAuth 1.0a |
| `GET /api/v1/devices/garmin/callback` | Garmin OAuth callback |
| `POST /api/v1/devices/garmin/disconnect` | Disconnect Garmin |
| `GET /api/v1/devices/withings/auth` | Start Withings OAuth |
| `GET /api/v1/devices/withings/callback` | Withings OAuth callback |
| `POST /api/v1/devices/withings/webhook` | Withings webhook |
| `POST /api/v1/devices/withings/disconnect` | Disconnect Withings |
| `GET /api/v1/devices/providers` | List connected providers |

### 4.x.3 Конфигурация

| Переменная | Default | Описание |
|------------|---------|----------|
| `DEVICE_AGGREGATOR_PORT` | `8083` | Порт сервера |
| `DEVICE_AGGREGATOR_METRICS_PORT` | `9093` | Порт metrics-сервера |
| `DB_HOST`, `DB_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `DB_SSLMODE` | — | PostgreSQL подключение |
| `DEVICE_TOKEN_ENCRYPTION_KEY` | — | AES-256-GCM ключ для шифрования refresh-токенов (обязателен) |
| `FITBIT_CLIENT_ID`, `FITBIT_CLIENT_SECRET`, `FITBIT_REDIRECT_URI` | — | Fitbit OAuth credentials |
| `GARMIN_CONSUMER_KEY`, `GARMIN_CONSUMER_SECRET`, `GARMIN_CALLBACK_URL` | — | Garmin OAuth 1.0a credentials |
| `WITHINGS_CLIENT_ID`, `WITHINGS_CLIENT_SECRET`, `WITHINGS_CALLBACK_URL` | — | Withings OAuth credentials |

### 4.x.4 Безопасность

- Refresh-токены шифруются через AES-256-GCM (`internal/crypto`)
- Валидация redirect URI: только HTTPS, только доверенные хосты (`fitbit.com`, `withings.com`, `withings.net`, `duckdns.org`)
- Webhook-подписи: HMAC-SHA256 для Withings
- OAuth state параметр хранится в БД с TTL 10 минут

### 4.x.5 Graceful Shutdown

- `signal.NotifyContext` для SIGINT/SIGTERM
- Graceful shutdown основного и metrics серверов (таймаут 10 секунд)

### 4.x.6 Метрики

- `http_request_duration_seconds{method, path}`
- `http_requests_total{method, path, status}`
- `error_total{service="device-aggregator", error_type}`
- `biometric_sync_lag_seconds{device_type, user_segment}`

### 4.x.7 Тесты

- Unit-тесты для handlers: health, disconnect, OAuth callback, auth start, redirect validation
- Запуск: `go test ./cmd/device-aggregator/...`

### 4.x.8 Особенности

- Логгер: `internal/logger` с полем `service: "device-aggregator"`
- Middleware: recovery, request ID, correlation ID, logging с Prometheus
- Garmin OAuth 1.0a использует `crypto/sha1` для подписи (требование Garmin Health API, см. `SECURITY.md`)

---

## 4.x Device Connector (device-connector)

### 4.x.1 Назначение

HTTP-сервис для регистрации носимых устройств и приёма биометрических данных с них. Отвечает за:
- Регистрацию устройств пользователей (создание `device_id` и `device_token`)
- Аутентификацию устройств по токену
- Валидацию и дедупликацию входящих записей
- Хранение сырых ingest-записей в PostgreSQL (`devices`, `device_ingest_log`)
- Форвардинг валидных записей в `biometric-service` через gRPC

### 4.x.2 Endpoints

| Endpoint | Назначение |
|----------|-----------|
| `GET /health` | Health check |
| `GET /metrics` | Prometheus метрики |
| `POST /api/v1/devices/register` | Регистрация устройства |
| `POST /api/v1/devices/{device_id}/ingest` | Приём данных с устройства |

### 4.x.3 Конфигурация

| Переменная | Default | Описание |
|------------|---------|----------|
| `DEVICE_CONNECTOR_PORT` | `8082` | Порт сервера |
| `DEVICE_CONNECTOR_METRICS_PORT` | `9094` | Порт metrics-сервера |
| `DB_HOST`, `DB_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `DB_SSLMODE` | — | PostgreSQL подключение |
| `BIOMETRIC_SERVICE_ADDR` | `localhost:50052` | Адрес biometric-service gRPC |

### 4.x.4 Формат запроса `POST /api/v1/devices/register`

```json
{
  "device_type": "fitbit",
  "user_id": "user-123"
}
```

### 4.x.5 Формат ответа регистрации

```json
{
  "device_id": "uuid",
  "device_type": "fitbit",
  "user_id": "user-123",
  "device_token": "uuid"
}
```

### 4.x.6 Формат запроса `POST /api/v1/devices/{device_id}/ingest`

```json
{
  "device_type": "fitbit",
  "device_token": "uuid",
  "sync_interval_ms": 5000,
  "records": [
    {
      "metric_type": "heart_rate",
      "value": 70.0,
      "timestamp": "2024-01-01T00:00:00Z",
      "quality": "good"
    }
  ]
}
```

### 4.x.7 Формат ответа ingest

```json
{
  "total_received": 10,
  "duplicates": 2,
  "forwarded": 8,
  "failed": 0
}
```

### 4.x.8 Валидация записей

- `metric_type` не может быть пустым
- `value` не может быть отрицательным
- Для `heart_rate`: диапазон 30–220
- Для `spo2`: диапазон 70–100
- Для остальных метрик: проверка по `metricSyncRules`

### 4.x.9 Дедупликация

Дубликаты определяются по триплету `(device_id, timestamp, metric_type)` через таблицу `device_ingest_log`.

### 4.x.10 gRPC форвардинг

Валидные записи форвардятся в `biometric-service` через `AddRecord` с предварительной валидацией `validator.ValidateBiometricRecord`.

### 4.x.11 Graceful Shutdown

- `signal.NotifyContext` для SIGINT/SIGTERM
- Graceful shutdown основного и metrics серверов (таймаут 10 секунд)

### 4.x.12 Метрики

- `http_request_duration_seconds{method, path}`
- `http_requests_total{method, path, status}`
- `error_total{service="device-connector", error_type}`
- gRPC client metrics через `metrics.UnaryClientInterceptor`

### 4.x.13 Middleware

- Recovery middleware
- Request ID
- Correlation ID
- Logging с Prometheus

### 4.x.14 Тесты

- Unit-тесты: `isValidDeviceType`, `metricSyncRules`, `healthHandler`, `registerDeviceHandler`, `ingestInputs`, `validateIngestRecord`, `authenticateDevice`
- Запуск: `go test ./cmd/device-connector/...`

### 4.x.15 Особенности

- Логгер: `internal/logger` с полем `service: "device-connector"`
- Поддерживаемые типы устройств: `fitbit`, `garmin`, `withings`
- Токены устройств хранятся в базе в открытом виде (в production требует шифрования)

---

## 4.x Background Data Processor (data-processor)

### 4.x.1 Назначение

Фоновый сервис для потребления биометрических событий из RabbitMQ (`biometric_events`) и сохранения их в PostgreSQL. Обеспечивает:
- Асинхронную запись биометрических данных с валидацией диапазонов
- Prometheus-метрики обработки сообщений
- Graceful shutdown с ожиданием завершения in-flight сообщений
- Health check и metrics endpoints

### 4.x.2 Endpoints

| Endpoint | Назначение |
|----------|-----------|
| `GET /health` | Health check (JSON `{"status":"healthy"}`) |
| `GET /metrics` | Prometheus метрики |

### 4.x.3 Конфигурация

| Переменная | Default | Описание |
|------------|---------|----------|
| `DATA_PROCESSOR_PORT` | `8084` | Порт health-сервера |
| `DATA_PROCESSOR_METRICS_PORT` | `9092` | Порт metrics-сервера |
| `DB_HOST`, `DB_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `DB_SSLMODE` | — | PostgreSQL подключение |
| `RABBITMQ_URL` | — | RabbitMQ подключение (обязателен) |

### 4.x.4 Graceful Shutdown

- `signal.NotifyContext` для SIGINT/SIGTERM
- Ожидание завершения текущих сообщений (таймаут 30 секунд)
- Graceful shutdown health и metrics серверов

### 4.x.5 Валидация событий

Перед записью в БД проверяются:
- `user_id` и `metric_type` не пустые
- `value >= 0`
- `value` в допустимых диапазонах для каждого типа метрики (heart_rate: 30–220, spo2: 70–100 и т.д.)

### 4.x.6 Метрики

- `error_total{service="data-processor", error_type="parse_error|validation_error|insert_error"}`
- `queue_messages_total{queue, status}` (через `internal/queue`)

### 4.x.7 Тесты

- Unit-тесты: парсинг, валидация, getMetricRules, вставка в БД
- Интеграционные тесты с Testcontainers (PostgreSQL + RabbitMQ), пропускаются если Docker недоступен
- Запуск: `go test ./cmd/data-processor/...` (без `-short` для integration тестов)

---

## 5. Порядок выпуска версий (Release Pipeline)

### 5.1 Девять этапов релиза

#### Этап 1: Разработка (Development)

- Ветка: `feature/*`
- Действия:
  - Разработка в изолированной ветке
  - Pre-commit hooks: lint, format, secret scan

#### Этап 2: Code Review

- Требования:
  - Minimum 1 approving review (Dependabot PRs auto-approve)
  - SAST scan: gosec (не SonarQube)
  - Dependency scan: govulncheck + Trivy + Dependabot
- Артефакты:
  - Approved PR с changelog

#### Этап 3: CI Build

- Jobs:
  - Unit tests (`make check`)
  - Security scanning: gosec SAST, govulncheck, Trivy (filesystem + config), Gitleaks, TruffleHog, Syft SBOM
  - Container scan: Trivy image scan (no Grype)
  - Build Docker images (single-arch, не multi-arch)
- Output: Image tag: `ghcr.io/mamuer/project/<service>:<sha>`

#### Этап 4: Deploy Test

- Environment: `test` (k3s on VPS)
- Automation: fully automated via `provision-k8s-vps` job
- Verification:
  - Smoke tests: TestContainers health checks
  - DB migrations applied
  - Seed admin created

#### Этап 5: Deploy Production

- Environment: `production`
- Действия:
  - UAT: тестирование продуктовой командой
  - Performance tests: k6 (p95 < 3s) — **автоматизировано в CI**
  - Security scan: Trivy + Kubescape — **автоматизировано в CI**
- Approval: Product Owner + Tech Lead sign-off (ручное)

#### Этап 6: Release Candidate

- Артефакты:
  - Git tag: `v2.1.0-rc1`
  - Changelog: auto-generated + manual review
  - Migration plan: K8s Job (`migrate-db.yaml`)
  - Runbook: шаги деплоя + отката

#### Этап 7: Deploy Production (Rolling)

**Rolling фаза**:

```yaml
batches: "по одному поду на сервис"
interval: "ручное подтверждение между обновлениями"
health_check: "readiness probe"
```

#### Этап 8: Post-Deploy Monitoring

- Duration: 24 hours
- Metrics watch:
  - Error rate (per endpoint)
  - p95/p99 latency
  - DB connection pool usage
  - ML model confidence drift
- Alert thresholds: см. раздел "Наблюдаемость"

#### Этап 9: Ручной откат (Rollback Trigger)

**Откат срабатывает при**:

- Error rate > 5% в течение 15 минут
- p95 latency > 10s в течение 15 минут
- Critical security vulnerability обнаружена
- Data loss > 0.1%

**Команды**:

```bash
# Kubernetes
kubectl rollout undo deployment/fitness-api -n prod

# Database
kubectl apply -f configs/k8s/base/jobs/migrate-db.yaml -n fitness-platform-production

# Verification
# Smoke tests + synthetic user journey
```

---

## 6. Критерии приемки архитектуры (Definition of Done)

### 6.1 Доступность (Availability)

**Требование**: > 99.9% uptime annually

```text
Calculation: (total_minutes - downtime_minutes) / total_minutes * 100
Monitoring: Prometheus uptime probe + synthetic transactions
```

**Пример**: 99.9% = 365 дней - 43.2 минуты максимум downtime в год.

### 6.2 Производительность (Performance)

**Требование**: p95 latency < 5 seconds для 95% пользовательских запросов

- Measurement: Histogram metrics + RUM (Real User Monitoring)
- Exceptions: ML inference endpoints: p95 < 15s (с уведомлением пользователя)

### 6.3 Масштабируемость (Scalability)

**Требование**: Автомасштабирование: 2× нагрузка → 2× поды за ≤ 3 минуты

- Implementation: Kubernetes HPA + Cluster Autoscaler
- Testing: Load test: k6 с постепенным увеличением RPS

### 6.4 Отказоустойчивость (Resilience)

**Требование**: Восстановление после сбоя < 5 минут

- Verification: Chaos Engineering: регулярные тесты (kill pod, network partition)
- Metrics: MTTR (Mean Time To Recovery) tracked in Grafana

### 6.5 Безопасность (Security)

**Требование**: 0 critical vulnerabilities после penetration test

**Процесс**:

- Ежемесячный внутренний скан (gosec, Trivy, govulncheck)
- Remediation SLA (best effort): critical 1–3 рабочих дней, high 3–7 рабочих дней

### 6.6 Резервное копирование (Backup)

**Требование**: Ежедневное резервное копирование с возможностью восстановления за < 1 час

**Текущее состояние**:
- Ежедневный `pg_dump` через cron job (`backup-postgres.sh`)

### 6.7 Документация (Documentation)

**Требование**: Актуальная документация в репозитории

**Обязательные документы**:

- Architecture Decision Records (ADR)
- Runbook для эксплуатации
- Incident Response Playbook
- API Specification (OpenAPI 3.0.3)

**Политика обновления**: Документация обновляется в том же PR, что и код.

---

## 7. Контрольный список проверки архитектуры

### Инфраструктура

- [x] Матрица окружений применена к основным компонентам
- [x] RabbitMQ настроен с persistent queues и DLQ
- [x] Prometheus: service discovery, дашборды Grafana

### Наблюдаемость

- [x] Все сервисы логируют в обязательном JSON-формате
- [x] Реализованы 6 обязательных Prometheus-метрик

### Безопасность

- [x] Network Policies разделяют зоны dmz/app/data/monitoring
- [x] RBAC: минимальные права, отдельные ServiceAccount
- [x] Шифрование: TDE/БД (pgcrypto), volumes, secrets
- [x] mTLS для внутренних gRPC-вызовов (hand-rolled TLS 1.3)
- [x] WAF настроен с базовым набором правил (Ingress NGINX + ModSecurity + OWASP CRS v4 + cert-manager)

### Релизный процесс

- [x] Пайплайн включает стадии: lint, test, security scan, build, deploy
- [x] Gosec
- [x] Govulncheck + Trivy

### Приемка

- [x] Определены метрики для availability
- [x] Настроены k6 нагрузочные тесты

### Документация

- [x] ADR для архитектурных решений
- [x] Runbook для эксплуатации и отката
- [x] API Specification (Protobuf + docs/API.md)

## 8. Генерация Protobuf (локальная разработка)

При изменении `api/proto/*.proto` сгенерированный Go-код (`api/gen/**`) нужно
пересоздать и закоммитить. Используйте цель `make proto` (см. `Makefile`, цель `proto`).

### Зависимости

- **`protoc`** — компилятор Protocol Buffers.
- **Плагины генерации Go** (должны быть в `PATH`):
  ```bash
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  ```
  Убедитесь, что `$GOPATH/bin` (по умолчанию `~/go/bin`) добавлен в `PATH`,
  иначе `protoc` завершится с ошибкой «protoc-gen-go: plugin not found».

### Установка `protoc` по платформам

```bash
# macOS (Homebrew)
brew install protobuf

# Ubuntu / Debian
sudo apt-get update && sudo apt-get install -y protobuf-compiler

# Windows (Chocolatey)
choco install protoc
```

После установки зависимостей:

```bash
make proto
```

См. также раздел «Протоколы (Protobuf)» в `CONTRIBUTING.md` для правил
версионирования `.proto` файлов.

## 9. Shared library `internal/auth` — ограничения и правила использования

### 9.1 Роль

`internal/auth` — это **общая библиотека JWT-аутентификации**, которая используется
несколькими сервисами (`gateway`, `user-service`, `middleware`). Она предоставляет
криптографические примитивы (ES256 ключи, подпись/валидация JWT, fingerprinting)
и доменные типы для claims.

### 9.2 Структура пакета

```text
internal/auth/
├── claims.go   # Доменные типы (Claims, JWKSKey, JWKSResponse)
└── jwt.go      # Инфраструктурная реализация (ES256, подпись, валидация)
```

- **`claims.go`**: зависит только от `github.com/golang-jwt/jwt/v5` для `RegisteredClaims`.
  Не содержит криптографической логики. Может использоваться в domain-слое.
- **`jwt.go`**: зависит от `crypto/ecdsa`, `crypto/x509`, `encoding/pem`, `jwt/v5`.
  Содержит всю инфраструктурную логику. Должен использоваться только в infra-слое.

### 9.3 Правила использования

1. **Каждый сервис определяет свой порт** в `cmd/<service>/ports/auth.go`:
   ```go
   type TokenProvider interface {
       GenerateAccessToken(userID, email, role string, ttl time.Duration) (string, error)
       GenerateRefreshToken() string
       ValidateAccessToken(token string) (*claims.Claims, error)
       ComputeTokenFingerprint(token string) string
   }
   ```

2. **Адаптер в infra-слое** (`cmd/<service>/infra/jwt_adapter.go`) реализует порт,
   делегируя вызовы в `internal/auth/jwt`. Только композиционный корень (`main.go`)
   знает о существовании `internal/auth`.

3. **Применение/доменный слой НЕ импортирует `internal/auth/jwt`** напрямую.
   Для передачи claims между слоями используется `internal/auth/claims`.

4. **Запрещено** добавлять в `internal/auth` бизнес-логику (например,
   хранение refresh-токенов, проверку ролей, интеграцию с БД). Этот пакет —
   только криптографические утилиты.

### 9.4 Обоснование

Полная гексагональная архитектура требовала бы вынесения JWT-логики в отдельный
auth-сервис. Однако для текущего масштаба проекта это избыточно. Компромисс:
- **`internal/auth`** = shared library с четко очерченной ответственностью.
- **Порты/адаптеры** в каждом сервисе сохраняют возможность тестирования
  (mock-реализации `TokenProvider`) и возможность смены алгоритма/библиотеки
  без изменения domain-слоя.

### 9.5 Миграция с прямого импорта

Если вы видите в коде сервиса прямой импорт `"github.com/MAMUER/project/internal/auth"`,
это технический долг. Правильный паттерн:

```go
// ❌ Bad
import "github.com/MAMUER/project/internal/auth"
token, err := auth.GenerateAccessToken(...)

// ✅ Good
import "github.com/MAMUER/project/cmd/<service>/ports"
import "github.com/MAMUER/project/cmd/<service>/infra"
token, err := s.tokenProvider.GenerateAccessToken(...)
```

## 10. Shared library `internal/config` — типизированная конфигурация и env helpers

### 10.1 Роль

`internal/config` — это **общая библиотека конфигурации**, которая используется
всем сервисами (`gateway`, `user-service`, `biometric-service` и др.). Она предоставляет:
- helpers для чтения env vars с поддержкой `_FILE` суффикса (Docker/Kubernetes secrets)
- typed accessors (`GetEnvInt`, `GetEnvBool`, `GetEnvDuration`, `GetEnvFloat64`, `GetEnvInt64`)
- обязательные env vars с паникой при отсутствии (`GetEnvRequired`)
- типизированные конфигурационные структуры (`CacheConfig`, `JWTConfig`, `ServerConfig`) с валидацией
- centralized constants (`DefaultTimeout`, `MaxBatchSize`, `ValkeyTTLSeconds`, `JWTExpirationHours`, `MinHeartRate`, `MaxHeartRate`, `MinSpO2`, `MaxSpO2`, `CorrelationIDHeader`)

### 10.2 Структура пакета

```text
internal/config/
├── env.go          # GetEnv, GetEnvRequired, GetEnvInt, GetEnvInt64, GetEnvBool, GetEnvDuration, GetEnvFloat64
├── config.go       # CacheConfig, JWTConfig, ServerConfig, Load*, Validate(), LogConfig
├── limits.go       # DefaultTimeout, MaxBatchSize, ValkeyTTLSeconds, JWTExpirationHours, MinHeartRate, MaxHeartRate, MinSpO2, MaxSpO2, CorrelationIDHeader
├── env_test.go
├── config_test.go
└── limits_test.go
```

### 10.3 Environment variable loading

Приоритет источников: `KEY_FILE` > `KEY` > `defaultValue`.

- Если `KEY_FILE` установлен, читается содержимое файла (trim).
- Иначе возвращается значение `KEY`.
- Если ни один источник не найден, возвращается `defaultValue` (если передан).

Пример:

```bash
JWT_PRIVATE_KEY_PEM_FILE=/run/secrets/jwt_private_key.pem
```

```go
privateKey := config.GetEnv("JWT_PRIVATE_KEY_PEM")
```

### 10.4 Typed accessors

| Функция | Тип возврата | Поведение при invalid/empty |
| --- | --- | --- |
| `GetEnv(key, default...)` | `string` | возвращает `defaultValue` |
| `GetEnvRequired(key)` | `string` | паникует, если пусто |
| `GetEnvInt(key, default)` | `int` | возвращает `default` |
| `GetEnvInt64(key, default)` | `int64` | возвращает `default` |
| `GetEnvBool(key, default)` | `bool` | возвращает `default` |
| `GetEnvDuration(key, default)` | `time.Duration` | возвращает `default` |
| `GetEnvFloat64(key, default)` | `float64` | возвращает `default` |

`GetEnvBool` поддерживает: `true`, `false`, `1`, `0`, `yes`, `no`, `on`, `off` (case-insensitive).

### 10.5 Configuration structs

```go
type CacheConfig struct {
    Addr     string
    Password string
    DB       int
}

type JWTConfig struct {
    PrivateKeyPEM string
    PublicKeyPEM  string
}

type ServerConfig struct {
    Addr string
}
```

Каждая структура имеет метод `Validate() error`.

### 10.6 Loaders

```go
func LoadCacheConfig() CacheConfig
func LoadJWTConfig() JWTConfig
func LoadServerConfig(envVar, defaultAddr string) ServerConfig
```

- `LoadJWTConfig` использует `GetEnvRequired`, поэтому при отсутствии `JWT_PRIVATE_KEY_PEM` или `JWT_PUBLIC_KEY_PEM` процесс завершится panic на старте.

### 10.7 Logging configuration

```go
config.LogConfig(log, cfg)
```

Логирует конфигурацию на уровне info, маскируя секреты (`[REDACTED]`).
Поддерживаемые типы: `CacheConfig`, `JWTConfig`, `ServerConfig`.

### 10.8 Constants

| Constant | Значение | Назначение |
| --- | --- | --- |
| `DefaultTimeout` | `5s` | Таймаут внешних вызовов по умолчанию |
| `MaxBatchSize` | `100` | Максимальный размер батча |
| `ValkeyTTLSeconds` | `3600` | TTL записей Valkey по умолчанию |
| `JWTExpirationHours` | `24` | Время жизни JWT по умолчанию |
| `MinHeartRate` | `30` | Минимальный допустимый пульс, bpm |
| `MaxHeartRate` | `220` | Максимальный допустимый пульс, bpm |
| `MinSpO2` | `70` | Минимальный допустимый SpO2, % |
| `MaxSpO2` | `100` | Максимальный допустимый SpO2, % |
| `CorrelationIDHeader` | `X-Correlation-ID` | HTTP-заголовок для корреляции запросов |

### 10.9 Правила использования

1. **Вся конфигурация загружается через typed loaders** (`LoadCacheConfig`, `LoadJWTConfig`, `LoadServerConfig`).
2. **Обязательные переменные** — только через `GetEnvRequired` или loaders, которые его используют.
3. **Валидация** — вызывается сразу после загрузки конфигурации в композиционном корне (`main.go`).
4. **Логирование** — `LogConfig` вызывается один раз после валидации для отладки/аудита.
5. **Секреты** — передаются через `_FILE` или env vars, логируются в маскированном виде.

## 11. Shared library `internal/domain` — доменные модели биометрических данных

### 11.1 Роль

`internal/domain` — это **ядро доменной модели** платформы. Он содержит:
- Типизированные entity для биометрических измерений (`BiometricData`)
- Enum `MetricType` для всех поддерживаемых метрик
- Валидацию инвариантов на уровне домена
- JSON-сериализацию для API

### 11.2 Структура пакета

```text
internal/domain/
├── biometric_data.go      # BiometricData entity, MetricType enum, Validate()
└── biometric_data_test.go # Тесты конструктора, валидации, констант
```

### 11.3 Доменная модель

```go
type BiometricData struct {
    ID         string     `json:"id"`
    UserID     string     `json:"user_id"`
    MetricType string     `json:"metric_type"`
    Value      float64    `json:"value"`
    Timestamp  time.Time  `json:"timestamp"`
    DeviceType string     `json:"device_type"`
    CreatedAt  time.Time  `json:"created_at"`
}
```

### 11.4 MetricType enum

```go
const (
    MetricHeartRate        MetricType = "heart_rate"
    MetricHRV              MetricType = "hrv"
    MetricSpO2             MetricType = "spo2"
    MetricTemperature      MetricType = "temperature"
    MetricBloodPressureSys MetricType = "blood_pressure_systolic"
    MetricBloodPressureDia MetricType = "blood_pressure_diastolic"
    MetricECG              MetricType = "ecg"
    MetricSleepStage       MetricType = "sleep_stage"
    MetricSteps            MetricType = "steps"
    MetricDistance         MetricType = "distance"
    MetricCalories         MetricType = "calories"
    MetricRespiratoryRate  MetricType = "respiratory_rate"
    MetricBloodGlucose     MetricType = "blood_glucose"
    MetricOxygenSaturation MetricType = "oxygen_saturation"
)
```

### 11.5 Валидация

```go
func NewBiometricData(userID, metricType string, value float64, timestamp time.Time, deviceType string) (*BiometricData, error)
func (b *BiometricData) Validate() error
```

Правила:
- `UserID` не может быть пустым
- `MetricType` не может быть пустым
- `Timestamp` не может быть нулевым
- `Value` должен быть >= 0
- `DeviceType` не может быть пустым
- `CreatedAt` устанавливается автоматически в `time.Now()`

### 11.6 Правила использования

1. **Конструктор `NewBiometricData`** — всегда используйте его для создания сущности, он валидирует инварианты.
2. **MetricType** — используйте только константы из пакета, избегайте magic strings.
3. **Репозиторий** — `internal/repository/biometric_repository.go` маппит `BiometricData` в/из PostgreSQL через `database/sql`.
4. **Сериализация** — JSON-теги используются для REST/gRPC responses.

## 12. Shared library `internal/crypto` — шифрование AES-GCM

### 11.1 Роль

`internal/crypto` — это **общая библиотека симметричного шифрования**, которая используется
всем сервисами для защиты чувствительных данных:
- `device-aggregator`: шифрование токенов устройств перед сохранением в БД
- `user-service`: шифрование TOTP-секретов перед сохранением в БД

### 11.2 Структура пакета

```text
internal/crypto/
└── totp_crypto.go   # AES-GCM encryptor (256-bit key)
```

### 11.3 Алгоритм и параметры

| Параметр | Значение |
| --- | --- |
| Алгоритм | AES-256-GCM |
| Размер ключа | 32 байта (256 бит) |
| Nonce | CSPRNG, размер равен `NonceSize()` (обычно 12 байт) |
| AAD | `nil` |

### 11.4 API

```go
type AESGCMEncryptor struct { ... }

func NewAESGCMEncryptor(keyMaterial string) (*AESGCMEncryptor, error)
func (e *AESGCMEncryptor) Encrypt(plaintext []byte) ([]byte, error)
func (e *AESGCMEncryptor) Decrypt(ciphertext []byte) ([]byte, error)
```

- `NewAESGCMEncryptor` принимает ключ в одном из форматов:
  - base64-строка (декодируется автоматически)
  - raw строка длиной 32 байта
- `Encrypt` возвращает `nonce || ciphertext || tag`
- `Decrypt` ожидает тот же формат, проверяет тег GCM

### 11.5 Правила использования

1. **Ключ загружается через `config`** в композиционном корне (`main.go`), передаётся в адаптер через DI.
2. **Никакого stateful init**: нет `Init*()` функций, которые хранят encryptor в пакетном состоянии.
3. **Данные никогда не логируются в открытом виде**: только маскированные или зашифрованные.
4. **Ключ ротируется через замену env var** и перезапуск сервиса.

## 13. Shared library `internal/db` — подключение к PostgreSQL и PII-шифрование

### 13.1 Роль

`internal/db` — это **общая библиотека работы с PostgreSQL**, которая используется
всем сервисами для:
- Создания подключений к PostgreSQL с connection pooling и метриками.
- Шифрования PII-данных через pgsodium (rand AES-GCM + blind index).
- Генерации nonce для шифрования полей.
- Загрузки ключа шифрования `DB_ENCRYPTION_KEY` из окружения.

### 13.2 Структура пакета

```text
internal/db/
├── db.go        # Config, LoadConfig, NewConnection, connection pool metrics
├── db_test.go   # Тесты подключения, пула, blind index, nonce, pgsodium helpers
├── pgp.go       # EncryptionKey, EmailHash
└── pgsodium.go  # PII-шифрование: aegis256 AEAD, blind index, nonce
```

### 13.3 Подключение к PostgreSQL

```go
type Config struct {
    Host     string
    Port     string
    User     string
    Password string
    DBName   string
    SSLMode  string
}
```

- `LoadConfig()` загружает конфигурацию из env vars с поддержкой `_FILE` суффикса.
- `Validate()` проверяет обязательные поля.
- `NewConnection(cfg)` открывает подключение, настраивает пул и запускает метрики.
- Connection pool: `MaxOpenConns=25`, `MaxIdleConns=10`, `ConnMaxLifetime=5m`.
- Метрика `DBConnectionPoolUsage` обновляется каждые 15 секунд.

### 13.4 PII-шифрование (pgsodium)

| Функция | Назначение |
| --- | --- |
| `BlindIndex(plaintext)` | lowercase hex SHA256 для поиска без утечки plaintext |
| `NicknameHash(nickname)` | alias для `BlindIndex` |
| `GenerateNonce()` | случайный 12-байтовый nonce для aegis256 AEAD |
| `PgsodiumRandomEncryptParam($N, $M)` | рандомизированное шифрование с nonce |
| `PgsodiumDecryptParam(ct, nonce, alias)` | расшифровка aegis256 |

### 13.5 Ключи

- `EncryptionKey()` возвращает ключ из `DB_ENCRYPTION_KEY` (64 hex chars или raw).
- `SetPgsodiumKeyID(id)` фиксирует идентификатор ключа в keyring pgsodium.
- `PgsodiumKeyringName()` возвращает имя ключа `fitpulse_pii`.

### 13.6 Правила использования

1. **Конфигурация загружается через `LoadConfig()`** в композиционном корне (`main.go`), затем валидируется.
2. **Ключ pgsodium импортируется один раз** при старте через `ensurePgsodiumKey` (`cmd/user-service/main.go`).
3. **Для всех PII-полей используется рандомизированное шифрование** (`PgsodiumRandomEncryptParam` + `GenerateNonce`) + blind index.
4. **Детерминированное шифрование удалено**: все поля используют nonce + aegis256.
5. **Никаких SQL-инъекций**: все литералы sanitize через `sanitize.String`, параметры передаются через `$N`.

## 14. Shared library `internal/email` — SMTP email sending

### 14.1 Роль

`internal/email` — это **общая библиотека отправки email**, которая используется
сервисами для:
- Отправки писем подтверждения email при регистрации (`user-service`).
- Поддержки TLS для production SMTP серверов (Yandex, Mail.ru, Gmail).
- Контроля дневного лимита и пропуска тестовых доменов.

### 14.2 Структура пакета

```text
internal/email/
├── email.go      # Config, LoadConfig, EmailSender interface, SMTPClient
└── email_test.go # Тесты конфигурации, SMTP, HTML шаблонов
```

### 14.3 Architecture

```go
// EmailSender is the port for sending emails.
type EmailSender interface {
    SendVerificationEmail(ctx context.Context, toEmail, verifyToken, baseURL string) error
}

// SMTPClient is an SMTP implementation of EmailSender.
type SMTPClient struct { ... }
```

- `EmailSender` — порт (интерфейс), который используется в доменных сервисах.
- `SMTPClient` — адаптер, реализующий отправку через `net/smtp`.

### 14.4 Configuration

```go
type Config struct {
    Host            string
    Port            int
    User            string
    Password        string
    From            string
    UseTLS          bool
    DailyLimit      int      // 0 = unlimited
    SkipSendDomains []string
}
```

- `LoadConfig()` загружает конфигурацию из env vars с поддержкой `_FILE` суффикса.
- `Validate()` проверяет обязательные поля и корректность порта.
- Environment variables: `SMTP_HOST`, `SMTP_PORT`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM`, `SMTP_TLS`, `EMAIL_DAILY_LIMIT`, `EMAIL_SKIP_DOMAINS`.

### 14.5 Правила использования

1. **В сервисах используйте только порт `EmailSender`**, никогда не импортируйте `SMTPClient` напрямую в доменный слой.
2. **Конфигурация загружается через `LoadConfig()`** в композиционном корне (`main.go`), затем валидируется.
3. **Контекст обязателен**: `SendVerificationEmail(ctx, ...)` поддерживает отмену и таймауты.
4. **Thread-safe daily limit**: `dailySent` защищен `sync.Mutex`, безопасен для concurrent use.
5. **Skip domains** — используются для тестовых окружений, возвращают ошибку `skipped: test domain ...`.
6. **TLS**: для production SMTP серверов используйте `UseTLS=true` с портом 465/587.
