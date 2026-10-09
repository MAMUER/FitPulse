# Infrastructure

## Testing Strategy

### Контекст

Staging используется для ручного/автоматизированного тестирования перед production. Отдельное постоянное тестовое окружение не развёрнуто из-за ограниченных ресурсов (один VPS). Интеграционные тесты используют testcontainers-go локально и в CI через ephemeral namespace.

### Acceptance Criteria

- Staging поддерживает deployment feature-веток для тестирования перед production
- CI/CD pipeline включает smoke/integration тесты через testcontainers-go
- Нет дублирования инфраструктуры между staging и production

---

## 3. PostgreSQL High Availability & Disaster Recovery

### 3.1 Контекст

Single PostgreSQL инстанс сейчас работает на том же VPS что и приложение. Phase 2 требует:

- автоматическое переключение при отказе
- read replicas для отдачи аналитической нагрузки

**Важно:** истинная HA с заявленными Acceptance Criteria (RTO < 30s, RPO = 0) возможна только при наличии **минимум 3 отдельных серверов/VPS** в разных географических locations. Patroni + etcd требует quorum из 3 узлов для автоматического failover. Синхронные реплики в пределах одного VPS не защищают от отказа хоста; для true RPO=0 требуется географическое распределение (multi-AZ/multi-region).

**Trade-off**: синхронные реплики в разных AZ увеличивают write latency на 50-200мс из-за ожидания подтверждения от реплик перед commit.

### 3.2 Acceptance Criteria

- RTO < 30 секунд при отказе primary.
- RPO = 0 при использовании синхронных реплик в разных AZ (availability zones).
- Автоматическое восстановление из бэкапа протестировано (ежеквартальные Game Days).
- Документация: `docs/compliance/ШАБЛОН_ОТЧЁТА_RECOVERY_DRILL.md`, `docs/runbooks/RECOVERY_DRILL.md`

---

## 6. Infrastructure as Code (Total rewrite)

### 6.1 Контекст

Phase 1 использует Kustomize + inline-скрипты для k3s. Частичная автоматизация уже есть: Vault SecretStore + ExternalSecret (`configs/k8s/base/vault/`), cert-manager, daily backups (`configs/k8s/base/backup/`). Текущая инфраструктура (2 vCPU / 4 ГБ RAM / 60 ГБ Storage, KVM, Ubuntu 26.04) ограничена для production-нагрузок, поэтому Phase 2 требует:

- аппаратного апгрейда/реновации VPS (обязательно, для поддержки HA-компонентов, Vault, Istio и бóльшего числа подов)
- после увеличения ресурсов VPS необходимо пересчитать параметры Argon2id

Текущие параметры Argon2id (memory 64 MB, iterations 3, parallelism 1) установлены с учётом ограничений текущего 2-vCPU / 4 ГБ сервера.
После переезда на более мощный VPS параметры требуется пересчитать.

**Best Practice**: реализовать автоматический benchmark Argon2id при старте сервиса для калибровки `memory`, `iterations` и `parallelism` (рекомендуется `parallelism` >= 2 для эффективного использования многоядерности CPU) под фактические `resources.limits`, сохраняя время хеширования в пределах 500мс–1с.
Реализовать кастомный benchmark-цикл при старте сервиса (или использовать `github.com/go-crypt/go-crypt`), так как `github.com/alexedwards/argon2id` не имеет встроенного auto-tuning.

- Terraform для управления инфраструктурой
- ArgoCD или Flux для GitOps
- единый репозиторий конфигураций

### 6.2 Acceptance Criteria

- Terraform модули для VPS, K8s, DB, Vault готовы и документированы
- ArgoCD/Flux манифесты готовы к деплою
- Policy as Code: OPA Gatekeeper policies deployed
- Существующие K8s манифесты мигрированы без потери функциональности

---

## 7. Disaster Recovery

### 7.1 Контекст

Нужны сценарии восстановления на случай loss of region/datacenter. Синхронные реплики в пределах одного VPS не защищают от отказа хоста; для true RPO=0 требуется географическое распределение (multi-AZ/multi-region).

Существующие Assets:

- Ежедневные бэкапы PostgreSQL: `configs/k8s/base/backup/`
- Скрипт восстановления: `scripts/restore-to-clone.sh`
- Шаблон отчёта: `docs/compliance/ШАБЛОН_ОТЧЁТА_RECOVERY_DRILL.md`
- Runbook: `docs/runbooks/INCIDENT_RESPONSE.md`

См. также: раздел 3 (PostgreSQL HA) для технической реализации бэкапов и репликации.

### 7.2 Acceptance Criteria

- Документация RTO/RPO по каждому сервису готова
- Восстановление данных: RTO < 4 часов, RPO < 15 минут
- Ежеквартальные recovery drills документированы
- RTO/RPO проверены на production-подобном окружении

---

## 8. Canary Deployments

### 8.1 Контекст

Текущий деплой — монолитный rollover на все поды одновременно. Отсутствие gradual rollout повышает риск даунтайма при регрессах.

**Зависимость**: требует развёрнутого Service Mesh (раздел 2) для sidecar-инъекции и traffic shaping.

### 8.2 Acceptance Criteria

- Любой deployment в production проходит через canary-фазу автоматически
- Rollback происходит без участия человека при error rate > baseline + 1%
- Время canary-фазы ≤ 10 минут до full rollout

---

## 15. Диаграмма зависимостей

### 15.1 Контекст

Phase 2 состоит из нескольких крупных блоков, которые нельзя выполнять хаотично. Ниже — обязательный порядок зависимостей.

### 15.2 Зависимости

```text
[Security email + PGP] ─────────────────────────┐
                                                   ↓
[Vault hardening] ───────► [Secrets Rotation Automation] ───────► [Service Mesh]
                                                           │
                                                           ▼
[Infra provisioning (VPS + k8s)] ───────► [PostgreSQL HA] ───────► [Backup DR]
                                                           │
                                                           ▼
[Observability расширение] ◄──────────────────── [Service Mesh]
           │
           ▼
[CAPTCHA] ───────► [Compliance 152-ФЗ] ───────► [Medical services integration] ───────► [Medical app registration]
                                                                                                   │
                                                                                                   ▼
                                                                                   [Adaptive daily plan retrain]

[Ежеквартальный внешний пентест] ───────► [Bug Bounty]
```

### 15.3 Блокеры

| Блок | Блокирует | Причина |
| --- | --- | --- |
| Infra provisioning | Vault, PostgreSQL HA, Service Mesh, Backup DR | Требует более мощного VPS и стабильного k8s |
| Vault hardening | Secrets Rotation Automation, Service Mesh (cert-manager) | Требует центрального хранилища секретов |
| PostgreSQL HA | Backup DR, Data Processor (production) | Требует стабильного Primary/Replica |
| Service Mesh | Observability (tracing), Canary Deployments | Требует sidecar-инъекции и PeerAuthentication |
| Security email + PGP | Bug Bounty, Pen Test, SECURITY.md обновление | Требует корпоративного ящика до публикации |
| Compliance 152-ФЗ | Medical services integration, Medical app registration | Требует шифрования и audit trail |

### 15.4 Параллельные работы

- **Vault hardening** можно начинать параллельно с **Infra provisioning** (на тестовом стенде)
- **CAPTCHA** не зависит от инфраструктуры, можно делать параллельно с **Compliance**
- **Bug Bounty program setup** не зависит от инфраструктуры, можно делать параллельно
- **Ежеквартальный внешний пентест** можно начинать после **Security email + PGP**, не зависит от инфраструктуры
- **Observability расширение** можно начинать после **Service Mesh**, но до полной миграции

---

## 18. Migration Strategy

### 18.1 Контекст

Каждый major change в Phase 2 требует стратегии миграции без downtime. Ниже — per-component планы.

### 18.2 Vault Migration (Kubernetes Secrets → Vault)

Текущее состояние: Vault уже развёрнут и работает через External Secrets Operator (SecretStore → ExternalSecret). Миграция завершена в Phase 1.

**Документация**: `docs/adr/0023-pgsodium-pii-encryption.md`, `configs/k8s/base/vault/`

**Подход**: Gradual migration с dual-read периодом.

1. **Week 1**: Развёртывание Vault в dev-окружении, настройка Kubernetes auth method
2. **Week 2**: Миграция 1–2 не критичных секретов (например, `SMTP_*`) на Vault, приложения читают из Vault, но fallback на K8s Secret
3. **Week 3**: Миграция всех секретов, dual-read: приложение читает из Vault, при недоступности — из K8s Secret
4. **Week 4**: Отключение K8s Secrets, все приложения читают только из Vault
5. **Rollback**: При проблемах с Vault — переключение обратно на K8s Secrets через environment variable `VAULT_ENABLED=false`

---

### 18.3 PostgreSQL HA Migration (Single → Patroni)

**Подход**: Rolling migration с использованием pg_basebackup.

1. **Week 1**: Развёртывание Patroni + etcd на отдельном VPS/поде
2. **Week 2**: Настройка streaming replication с текущего primary на новый Patroni cluster
3. **Week 3**: Переключение application connection string на Patroni VIP, тестирование failover
4. **Week 4**: Деcommission старого single PostgreSQL
5. **Rollback**: При проблемах — переключение connection string обратно на старый primary

---

### 18.4 Service Mesh Migration (hand-rolled mTLS → Istio/Linkerd)

**Подход**: Canary migration namespace-by-namespace.

1. **Week 1**: Установка Istio control plane в dedicated namespace, без sidecar-инъекции
2. **Week 2**: Включение sidecar-инъекции для 1 не критичного namespace (например, `ml-generator`)
3. **Week 3**: Постепенное включение для остальных namespaces, мониторинг latency/errors
4. **Week 4**: Отключение hand-rolled mTLS, полный переход на mesh
5. **Rollback**: При проблемах — отключение sidecar-инъекции, возврат к hand-rolled mTLS

---

### 18.5 Data Processor Migration (stub → production)

Текущее состояние: data-processor уже работает в production как RabbitMQ consumer с DLQ.

**Подход**: Blue-green deployment consumer'а.

1. **Week 1**: Развёртывание data-processor в production с `PREFETCH=1`, без обработки сообщений (consumer-only, Nack all)
2. **Week 2**: Включение обработки для 10% сообщений (sampling), мониторинг dead-letter queue
3. **Week 3**: Полный rollout, мониторинг lag и error rate
4. **Week 4**: Отключение legacy-публения в `biometric_events` из biometric-service (если publisher migrated)
5. **Rollback**: Отключение data-processor подов, сообщения накапливаются в RabbitMQ
