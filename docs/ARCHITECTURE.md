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
├── deploy/
│   └── lb/
│       ├── production.conf           # Host NGINX конфигурация
│       └── install-crs.sh            # ModSecurity CRS установка
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
├── pkg/                              # Публичные пакеты
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

**Опции**:

- Nginx + ModSecurity (open-source, CRS ruleset)
- AWS WAF / Cloudflare (managed rules + custom)

**Правила**:

- SQL injection, XSS, path traversal блокировка
- Rate limiting: 100 req/min per IP для анонимных пользователей
- Geo-blocking: доступ только из разрешённых регионов (опционально)

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
Миграция: `db/migrations/V20__add_biometric_dedup.sql`.
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
- [x] WAF настроен с базовым набором правил (ModSecurity CRS v4)

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
