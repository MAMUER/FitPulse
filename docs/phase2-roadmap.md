# Phase 2 Backlog

> Детализационный бэклог инфраструктуры и масштабирования.

## 1. Секрет-хранилище: HashiCorp Vault

### 1.1 Контекст

Текущий подход — Kubernetes Secrets + CI-секреты — не покрывает требования к ротации, динамическим учётным данным и audit trail на уровне хранилища.

Vault будет центральным хранилищем всех паролей, секретов (JWT_SECRET, database credentials, API keys, TLS private keys и т.д.) с автоматической ротацией **каждые 30 дней** для ключей и credentials приложений.

### 1.2 Задачи

1. Развёртывание Vault на отдельном инстансе (или managed)
2. Kubernetes auth method: сервисы получают динамические credentials
3. Автоматическая ротация PostgreSQL и RabbitMQ паролей (30 дней)
4. Интеграция с CI/CD: `VAULT_ADDR`, `VAULT_TOKEN` через GitHub OIDC
5. Автоматическая ротация всех секретов (JWT_SECRET, API keys, TLS keys) раз в 30 дней
6. Бэкап Vault storage (Shamir secret shares + sealed keys)

### 1.3 Acceptance Criteria

- Все секреты POSTGRES_PASSWORD, JWT_SECRET и т.д. живут в Vault
- Автоматическая ротация всех секретов (включая database credentials, JWT_SECRET, API keys, TLS keys) раз в 30 дней
- При компрометации pod можно отозвать доступ за < 5 минут
- Vault audit log отправляется в ELK

---

## 2. mTLS и Service Mesh

### 2.1 Контекст

Текущая архитектура использует plain HTTP/gRPC между сервисами. В Phase 2 требуется:

- mutual TLS для всех внутренних коммуникаций
- авторизация на уровне сервиса (SPIFFE ID)
- observability и telemetry через mesh

### 2.2 Задачи

1. Выбор: Istio (полноценный mesh) или lightweight вариант (SPIRE для SPIFFE ID / cert-manager + linkerd)
2. Настройка `cert-manager` + внутреннего CA
3. Включение strict mTLS для всех namespace (через `linkerd` или `istio`)
4. Добавление `PeerAuthentication` и `AuthorizationPolicy`
5. Включение tracing (Jaeger/Zipkin) через mesh sidecar

### 2.3 Acceptance Criteria

- Все connection между сервисами используют TLS 1.3
- egress/ingress traffic control через AuthorizationPolicy
- Внешний доступ к сервисам возможен только через Gateway

---

## 3. PostgreSQL High Availability

### 3.1 Контекст

Single PostgreSQL инстанс сейчас работает на том же VPS что и приложение. Phase 2 требует:

- автоматическое переключение при отказе
- read replicas для отдачи аналитической нагрузки
- PITR для восстановления на произвольный момент

### 3.2 Задачи

1. Развёртывание Patroni + etcd (или managed Aurora/CloudSQL)
2. Настройка 1 primary + 2 synchronous replicas
3. Настройка pg_basebackup + WAL-архивации в S3
4. Настройка HAProxy/ProxySQL как единой точки дохода
5. Интеграция с мониторингом: `pg_stat_replication`, `pg_stat_activity`

### 3.3 Acceptance Criteria

- RTO < 30 секунд при отказе primary
- RPO = 0 только при multi-AZ; для single-VPS: RPO < 1 мин (WAL shipping). RPO=0 с синхронными репликами увеличивает write latency при cross-AZ.
- Автоматическое восстановление из бэкапа протестировано

---

## 4. Valkey 9 (previously Redis)

### 4.1 Контекст

Valkey используется для кэширования и сессий

### 4.2 Задачи

1. Развёртывание Valkey 9 (3 мастера + 3 реплики)
2. Настройка persistence (AOF + RDB)
3. Интеграция с application: connection pooling, sentinel-режим
4. Мониторинг: memory usage, hit rate, latency

### 4.3 Acceptance Criteria

- Высокая доступность: автоматическое failover при отказе мастера
- Поддержка dataset > 50GB
- Бэккап раз в 6 часов в S3
- Совместимость с Valkey CLI и go-redis клиентом

---

## 5. Compliance: 152-ФЗ

### 5.1 Контекст

152-ФЗ «О персональных данных» требует:

- шифрование at rest и in transit
- хранение данных на территории РФ
- аудит действий с ПДн
- механизмы реализации прав субъекта (доступ, удаление)

### 5.2 Задачи

1. Выбор площадки для хранения данных: Yandex Cloud, Selectel, или own datacenter
2. Шифрование БД (pgcrypto/TDE) и объектного хранилища
3. Включение audit log: PostgreSQL pgaudit, application audit, ELK retention 3 года
4. Реализация API для субъекта ПДн: export / delete
5. Подготовка документации (Политика обработки ПДн, Инструкция по работе с инцидентами)

### 5.3 Acceptance Criteria

- Все персональные данные (email, biometric) шифруются в БД и в transit
- Audit log доступен для запросов Роскомнадзора
- Утверждена локальная политика безопасности

---

## 6. Backup & Recovery Strategy

### 6.1 Контекст

Phase 1 имела базовый бэкап. Phase 2 включает:

- инкрементальные WAL-бэкапы
- PITR
- тестирование восстановления раз в квартал

### 6.2 Задачи

1. Автоматические полные бэкапы раз в день + WAL-архивация
2. Шифрование бэкапов (AES-256, ключ в Vault)
3. Тестовый стенд восстановления (restore-to-clone)
4. Scheduled Chaos tests: отключение primary БД, Valkey master, Vault

### 6.3 Acceptance Criteria

- Восстановление за < 1 час
- Бэкапы реплицируются в 2 географических зоны
- Recovery drill — раз в квартал с публичным отчётом

---

## 7. Observability & SLO

### 7.1 Контекст

Phase 1: Prometheus + Grafana + ELK. Phase 2 расширяет:

- SLI/SLO/SLA дашборды
- алертинг по Grafana OnCall (open-source) или Alertmanager + Telegram/Slack webhook
- distributed tracing

### 7.2 Задачи

1. Определение SLI/SLO:
   - Availability: 99.9%
   - Latency p95: < 2s
   - Error budget: 0.1%
2. Настройка Alertmanager со стратегией эскалации
3. Включение Jaeger/Zipkin для gRPC и HTTP
4. Дашборды: SLO burn rate, RED metrics

---

## 8. Infrastructure as Code (Total rewrite)

### 8.1 Контекст

Phase 1 использует Kustomize + inline-скрипты для k3s. Phase 2 требует:

- увеличение ресурсов VPS (обязательно, для поддержки HA-компонентов, Vault, Istio и бóльшего числа подов)
- Terraform для управления инфраструктурой
- ArgoCD или Flux для GitOps
- единый репозиторий конфигураций

### 8.2 Задачи

1. Terraform модули: VPS, K8s cluster, DB, Valkey 9, Vault
2. ArgoCD для declarative deploy'а
3. Sealed Secrets или External Secrets Operator для секретов
4. Policy as Code: OPA Gatekeeper

### 8.3 Phase 2 Quick Wins (не ждут полного рерайта)

1. **dm-crypt/LUKS**: запустить `configs/k8s/scripts/configure-storage-encryption.sh` на VPS с дополнительным volume; в CI добавлен подготовительный шаг.

---

## 9. Disaster Recovery

### 9.1 Контекст

Нужны сценарии восстановления на случай loss of region/datacenter.

### 9.2 Задачи

1. Документация RTO/RPO по каждому сервису
2. Автоматический DR failover (warm standby на another VPS)
3. Тестирование DR раз в полгода
4. Восстановление данных: RTO < 4 часов, RPO < 15 минут

---

## 10. Canary Deployments

### 10.1 Контекст

Текущий деплой — монолитный rollover на все поды одновременно. Отсутствие gradual rollout повышает риск даунтайма при регрессах.

### 10.2 Задачи

1. Интеграция ArgoCD Rollouts (или Flagger) с существующим GitOps-пайплайном
2. Конфигурация canary-стратегии: 5% → 25% → 50% → 100% traffic
3. Автоматический rollback при превышении порога ошибок (error rate / latency p99)
4. Интеграция с Observability: метрики Prometheus как источник сигналов для rollback
5. Документация runbook: manual promotion, manual rollback, pause

### 10.3 Acceptance Criteria

- Любой deployment в production проходит через canary-фазу автоматически
- Rollback происходит без участия человека при error rate > baseline + 1%
- Время canary-фазы ≤ 10 минут до full rollout

## 11. Bug Bounty / Researcher Program

### 11.1 Контекст

Сейчас проект предоставляет только внутренний reporting через GitHub Security Advisory + email, без бюджета/вознаграждения. В Phase 2 требуется:

- детализированный scope;
- explicit reward tiers;
- вариант self-hosted policy или платформенной интеграции;
- transparent SLA по ответу.

### 11.2 Задачи

1. Подготовить `BUG_BOUNTY_SCOPE.md` со scope, severity tiers, rules, expected response time.
2. Оценить бюджет/возможность денежного вознаграждения.
3. Настроить PGP key fingerprint для шифрования чувствительных отчётов об уязвимостях, чтобы предотвратить перехват информации о zero-day уязвимостях при передаче по email.
4. **Вариант B (self-hosted)**:
    - Создать `BUG_BOUNTY.md` с полной политикой;
    - Добавить в `SECURITY.md` раздел `## Отчеты об уязвимостях` с email, PGP key, SLA.
5. **Вариант C (platform)**:
    - Зарегистрировать программу на HackerOne / Bugcrowd / Intigriti;
    - Определить in-scope: `fittpulse.duckdns.org`, API endpoints;
    - Интегрировать алерты в Slack/Telegram;
    - Добавить ссылку на программу в `SECURITY.md`.

### 11.3 Acceptance Criteria

- PGP key fingerprint опубликован в `SECURITY.md` и `BUG_BOUNTY.md` для безопасной коммуникации.
- Исследователи могут использовать PGP для шифрования отчётов об уязвимостях.
- Документы `BUG_BOUNTY_SCOPE.md` и `BUG_BOUNTY.md` готовы и public.
- Решение по Варианту B или C принято.
- SLA по ответу задокументирован и публичен.

## Сроки (оценочно)

|Этап|Срок|Ответственный|
|---|---|---|
|Infra provisioning (VPS + k8s)|2-3 недели|DevOps|
|Vault + Secrets|1 неделя|DevOps/Backend|
|PostgreSQL HA|2 недели|DBA/DevOps|
|Valkey 9 Cluster|1 неделя|DevOps|
|mTLS / Service Mesh|2 недели|Platform|
|Compliance (152-ФЗ)|4-6 недель|Legal/DevOps|
|Backup DR|1-2 недели|DevOps|
|Observability расширение|1 неделя|Platform|

### Итого Phase 2: 3-4 месяца

В Phase 2 может работать параллельно над новыми фичами из `docs/UI_SPECIFICATION.md` (Achievements, Diet, Devices) — они не зависят от инфраструктурных изменений.
