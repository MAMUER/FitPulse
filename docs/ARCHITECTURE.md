# FitPulse — Полная архитектура и операционная документация

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

- Quorum queues (Raft consensus) для отказоустойчивости (classic mirrored queues deprecated)
- DLQ: `<queue-name>.dlq` для анализа ошибок
- TTL на сообщениях: 24 часа для сообщений уведомлений

### 1.2 Logging Stack: ELK (Elasticsearch, Logstash, Kibana)

```yaml
component: "ELK Stack"
purpose: "Централизованное хранение и анализ логов"

retention:
  hot: "90 дней"
  cold: "Архивация в S3 (1 год)"

requirements:
  - "Structured JSON logging (обязательные поля: timestamp, level, correlationId, userId)"
  - "Индексация по service, action, error_code для быстрого поиска"
  - "Role-based access в Kibana: dev → read-only, security → full access"
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

requirements:
  - "Service discovery через Kubernetes annotations"
  - "Recording rules для pre-aggregated метрик"
  - "Alertmanager интеграция с Slack/PagerDuty"
```

---

## 2. Матрица конфигураций по окружениям

|Параметр|Dev|Test|Staging|Prod|
|---|---|---|---|---|
|**K8s pods per service**|1|2|3|5+ (HPA: min=5, max=20)|
|**PostgreSQL topology**|1 инстанс (локальный, PG 18)|1 primary + 1 replica|1 primary + 2 replicas|1 primary + 3 replicas (1 sync + 2 async, PG 16)|
|**Valkey topology**|1 узел (Valkey 9)|3 узла (Sentinel)|3 узла (Sentinel)|6 узлов (Cluster mode, 3 master + 3 replica)|
|**GPU resources**|CPU only|1× NVIDIA T4|2× NVIDIA T4|4+× NVIDIA A10 (ML inference)|
|**Monitoring stack**|Базовый (логи в консоль)|ELK + Prometheus (full)|Полный + алерты в Slack|Полный + on-call ротация + PagerDuty|
|**Backup strategy**|Нет|Ежедневно (pg_dump)|Каждые 12 часов (WAL-архивация)|Каждые 6 часов (WAL) + PITR|
|**SSL/TLS**|Self-signed|Let's Encrypt (авто-ротация)|Corporate CA|Corporate CA + HSM|
|**Access control**|Локальный доступ|VPN|VPN + 2FA (TOTP)|2FA + IP whitelist + Hardware token|

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
    description: "ELK, Prometheus, Grafana"
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
- mTLS для gRPC-коммуникации между микросервисами (istio/linkerd)
- HSTS + Certificate Transparency logs вместо certificate pinning (SPA в браузере не поддерживает кастомный пиннинг сертификатов)

### 4.4 Управление зависимостями

|Инструмент|Функция|
|---|---|
|Dependabot|Еженедельный скан, авто-PR для минорных обновлений|
|Snyk|Интеграция в CI/CD, блокировка мержа при critical CVE|

**Политики**:

- Critical CVE: патч в течение 24 часов
- High CVE: патч в течение 7 дней
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

## 5. Порядок выпуска версий (Release Pipeline)

### 5.1 Девять этапов релиза

#### Этап 1: Разработка (Development)

- Ветка: `feature/*`
- Действия:
  - Разработка в изолированной ветке
  - Pre-commit hooks: lint, format, secret scan

#### Этап 2: Code Review

- Требования:
  - Minimum 2 approving reviews
  - SAST scan: SonarQube (quality gate: no critical issues)
  - Dependency scan: Snyk/Dependabot
- Артефакты:
  - Approved PR с changelog

#### Этап 3: CI Build

- Jobs:
  - Unit tests (coverage ≥95%)
  - Integration tests (TestContainers)
  - Contract tests (Pact)
  - Container scan: trivy/grype (no critical CVE)
  - Build multi-arch image (amd64 + arm64)
- Output: Immutable image tag: `sha256:abc123`

#### Этап 4: Deploy Test

- Environment: `test`
- Automation: fully automated
- Verification:
  - Smoke tests: health checks, basic flows
  - API contract validation

#### Этап 5: Deploy Production

- Environment: `production`
- Действия:
  - UAT: тестирование продуктовой командой
  - Performance tests: k6 (p95 < 3s)
  - Security scan: OWASP ZAP full scan
  - Chaos test: случайное убийство 1 пода
- Approval: Product Owner + Tech Lead sign-off

#### Этап 6: Release Candidate

- Артефакты:
  - Git tag: `v2.1.0-rc1`
  - Changelog: auto-generated + manual review
  - Migration plan: Flyway scripts + rollback instructions
  - Runbook: шаги деплоя + отката

#### Этап 7: Deploy Production (Canary + Rolling)

**Canary фаза**:

```yaml
traffic: "10%"
duration: "1 hour"

success_criteria:
  - "Error rate < 1%"
  - "p95 latency < 3s"
  - "No critical logs"
```

**Rolling фаза**:

```yaml
batches: "30% → 60% → 100%"
interval: "30 minutes между батчами"
health_check: "readiness probe + synthetic transactions"
```

#### Этап 8: Post-Deploy Monitoring

- Duration: 24 hours
- Metrics watch:
  - Error rate (per endpoint)
  - p95/p99 latency
  - DB connection pool usage
  - ML model confidence drift
- Alert thresholds: см. раздел "Наблюдаемость"

#### Этап 9: Автоматический откат (Rollback Trigger)

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
flyway undo -target=previous_version

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

- Ежеквартальный внешний пентест
- Ежемесячный внутренний скан (OWASP ZAP)
- Remediation SLA: critical 24h, high 7d

### 6.6 Соответствие (Compliance)

**Требование**: Полное соответствие 152-ФЗ (персональные данные)

**Реализация**:

- Хранение ПДн только на территории РФ (Yandex Cloud / Selectel)
- Шифрование ПДн в покое и при передаче
- Механизм выполнения прав субъекта ПДн (удаление, экспорт)
- Регистрация в Роскомнадзоре (оператор ПДн)

**Проверка**:

- Ежегодный аудит на соответствие 152-ФЗ
- DPIA (Data Protection Impact Assessment) для новых фич

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

- [ ] Матрица окружений применена ко всем компонентам
- [ ] RabbitMQ настроен с persistent queues и DLQ
- [ ] ELK Stack: 90 дней хранения, JSON-логи, RBAC в Kibana
- [ ] Prometheus: service discovery, recording rules, Alertmanager

### Наблюдаемость

- [ ] Все сервисы логируют в обязательном JSON-формате
- [ ] Реализованы 6 обязательных Prometheus-метрик
- [ ] Настроены алерты с эскалацией по уровням SEV

### Безопасность

- [ ] Network Policies разделяют зоны dmz/app/data/monitoring
- [ ] RBAC: минимальные права, отдельные ServiceAccount
- [ ] Шифрование: TDE/БД, volumes, secrets
- [ ] mTLS для внутренних gRPC-вызовов
- [ ] WAF настроен с базовым набором правил

### Релизный процесс

- [ ] Пайплайн включает все 9 этапов
- [ ] Canary-деплой с критериями успеха/отката
- [ ] Автоматический rollback при error rate > 5% или p95 > 10s

### Приемка

- [ ] Определены метрики для 99.9% availability
- [ ] Настроены нагрузочные тесты для проверки p95 < 5s
- [ ] План Chaos Engineering для проверки восстановления < 5 мин
- [ ] Пентест запланирован до релиза
- [ ] Реализованы механизмы соответствия 152-ФЗ

### Документация

- [ ] ADR для всех архитектурных решений
- [ ] Runbook для эксплуатации и отката
- [ ] OpenAPI-спецификация актуальна и покрыта тестами
