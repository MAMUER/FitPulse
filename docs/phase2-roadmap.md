# Phase 2 Backlog

> Детализационный бэклог инфраструктуры и масштабирования.

---

## Table of Contents

- [2. Service Mesh](#2-service-mesh)
- [3. PostgreSQL High Availability & Disaster Recovery](#3-postgresql-high-availability--disaster-recovery)
- [4. Compliance: 152-ФЗ](#4-compliance-152-фз)
- [5. Наблюдаемость и SLO](#5-наблюдаемость-и-slo)
- [6. Infrastructure as Code (Total rewrite)](#6-infrastructure-as-code-total-rewrite)
- [7. Disaster Recovery](#7-disaster-recovery)
- [8. Canary Deployments](#8-canary-deployments)
- [9. Security Reporting: Corporate Email + PGP](#9-security-reporting-corporate-email--pgp)
- [10. CAPTCHA (Cloudflare Turnstile)](#10-captcha-cloudflare-turnstile)
- [11. Secrets Rotation Automation](#11-secrets-rotation-automation)
- [12. Интеграция с медицинскими сервисами](#12-интеграция-с-медицинскими-сервисами)
- [13. Регистрация приложения как медицинского](#13-регистрация-приложения-как-медицинского)
- [14. Ежедневная адаптивная модификация плана](#14-ежедневная-адаптивная-модификация-плана)
- [15. Диаграмма зависимостей](#15-диаграмма-зависимостей)
- [16. Оценка стоимости (руб/мес)](#16-оценка-стоимости-рубмес)
- [17. Resource Plan: FTE](#17-resource-plan-fte)
- [18. Migration Strategy](#18-migration-strategy)
- [19. Risk Register](#19-risk-register)
- [20. Exit Criteria для Phase 2](#20-exit-criteria-для-phase-2)
- [21. Timeline с Milestones](#21-timeline-с-milestones)
- [22. Критерии приёмки Phase 2](#22-критерии-приёмки-phase-2)
- [23. Phase 3 Preview](#23-phase-3-preview)
- [24. Расширение поддерживаемых устройств](#24-расширение-поддерживаемых-устройств)
- [25. Полноценная двухфакторная верификация через Google](#25-полноценная-двухфакторная-верификация-через-google)

---

## 2. Service Mesh

### 2.1 Контекст

Phase 1 покрывает базовый mTLS между микросервисами на уровне gRPC (TLS 1.3, hand-rolled certs из Kubernetes Secret). Phase 2 переводит внутренние коммуникации на полноценный service mesh (Istio/Linkerd) с автоматической ротацией сертификатов, SPIFFE ID и распределённым трейсингом.

### 2.2 Acceptance Criteria

- mTLS активен между всеми сервисами через service mesh (Istio/Linkerd) с автоматической ротацией сертификатов через cert-manager
- Статические сертификаты в Kubernetes Secret удалены, все сертификаты генерируются динамически и монтируются через sidecar
- egress/ingress traffic control через AuthorizationPolicy
- Внешний доступ к сервисам возможен только через Gateway

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

---

## 4. Compliance: 152-ФЗ

### 4.1 Контекст

152-ФЗ «О персональных данных» требует:

- шифрование at rest и in transit
- хранение данных на территории РФ
- аудит действий с ПДн
- механизмы реализации прав субъекта (доступ, удаление)

Документы подготовлены в `docs/compliance/`.

### 4.2 Acceptance Criteria

| Критерий | Статус | Примечание |
| --- | --- | --- |
| Хранение данных на территории РФ | ⚠️ Требует реализации | Нужно разместить сервисы и БД на хосте в РФ |
| Утверждена локальная политика безопасности | ⚠️ Требует утверждения | Документы подготовлены в `docs/compliance/`, требуется формальное утверждение |

---

## 5. Наблюдаемость и SLO

### 5.1 Контекст

Production-окружения содержат: gateway, user-service, biometric-service, training-service, classifier, ml-generator, data-processor.
Production domain: fittpulse.ru
Актуальные сервисы и endpoints:

- Portal: <https://fittpulse.ru>
- API: <https://fittpulse.ru/api/v1/>
- Health checks: /health, /confirm, /logout
- ML endpoints: /api/v1/ml/chat
Centralized logging (ELK/Fluent Bit): см. пункт 4 (Compliance: 152-ФЗ, retention 90 дней).

### 5.2 Acceptance Criteria

- Все critical endpoints покрыты RED/SLO дашбордами в Grafana
- Burn rate алерты срабатывают при превышении порога в течение 30 секунд
- Jaeger UI доступен и принимает трейсы от всех сервисов (trace ID коррелируется с логами)
- On-call бот доставляет алерты SEV-1/SEV-2 в Telegram в течение 1 минуты
- Метрики ошибочного бюджета SLO приводят к автоматическому применению политик
- Retention Prometheus: 15 дней (hot); retention Alertmanager: 90 дней (см. раздел 4 для централизованного логирования)

---

## 6. Infrastructure as Code (Total rewrite)

### 6.1 Контекст

Phase 1 использует Kustomize + inline-скрипты для k3s. Текущая инфраструктура (2 vCPU / 4 ГБ RAM / 60 ГБ Storage, KVM, Ubuntu 26.04) ограничена для production-нагрузок, поэтому Phase 2 требует:

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

---

## 7. Disaster Recovery

### 7.1 Контекст

Нужны сценарии восстановления на случай loss of region/datacenter. Синхронные реплики в пределах одного VPS не защищают от отказа хоста; для true RPO=0 требуется географическое распределение (multi-AZ/multi-region).

См. также: раздел 3 (PostgreSQL HA) для технической реализации бэкапов и репликации.

### 7.2 Acceptance Criteria

- Документация RTO/RPO по каждому сервису готова
- Восстановление данных: RTO < 4 часов, RPO < 15 минут

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

## 9. Security Reporting: Corporate Email + PGP

### 9.1 Контекст

Текущий security reporting использует личный email (`mihnikolaenko12@yandex.ru`), что нарушает best practices. Базовая self-hosted политика уже реализована: созданы `BUG_BOUNTY_SCOPE.md` и раздел в `SECURITY.md`, определены in-scope/out-of-scope цели и transparent SLA по ответу (best effort).

В Phase 2 требуется:

- заменить личный email на корпоративный alias
- усилить криптографическую защиту отчётов через PGP
- оценить возможность перехода на профессиональные платформы или выделения бюджета

**Связанные разделы**:

- Раздел 10 (ежеквартальный внешний пентест) зависит от наличия корпоративного email
- Раздел 11 (Bug Bounty) интегрируется с этим контуром

### 9.2 Acceptance Criteria

- PGP key fingerprint опубликован в `SECURITY.md` и `BUG_BOUNTY_SCOPE.md`.
- PGP-ключ настроен и доступен через WKD.
- Все security-отчёты принимаются на корпоративный ящик.
- Личный email больше не указан как primary контакт.
- PGP-ключ защищает in-transmit отчёты.
- Есть процедура ротации доступа к ящику.
- Принято финальное решение по бюджету и/или платформенной интеграции (с документированным обоснованием).

---

## 10. CAPTCHA (Cloudflare Turnstile)

### 10.1 Контекст

Phase 1 использует жёсткий rate limiting (при превышении порога — блокировка). Это создаёт риск отзыва legitimate-пользователей при ложных срабатываниях, cross-user NAT и burst-трафике. CAPTCHA при превышении порога ошибок позволяет подтвердить человечность, не блокируя полностью.

Интеграция реализована в `cmd/gateway/captcha.go`, манифесты в `configs/k8s/base/captcha/`.

### 10.2 Acceptance Criteria

- При превышении порога rate limit пользователь видит CAPTCHA, а не жёсткий блок.
- Успешное решение CAPTCHA снимает ограничение на фиксированный период (например, 5 минут).
- CAPTCHA логируется с correlationId и участвует в RED metrics.

---

## 11. Secrets Rotation Automation

### 11.1 Контекст

Phase 2 внедряет HashiCorp Vault для хранения секретов (см. раздел 6), но требуется автоматизация ротации и инъекции секретов в поды без перезапуска сервисов.

**Связанные разделы**:

- Раздел 6 (Infrastructure as Code) — развёртывание Vault
- Раздел 2 (Service Mesh) — cert-manager для сертификатов mesh

### 11.2 Acceptance Criteria

- Все секреты инжектируются в поды автоматически без перезапуска сервисов.
- Ротация происходит прозрачно для приложений.
- Алерты срабатывают при проблемах с ротацией.

---

## 12. Интеграция с медицинскими сервисами

### 12.1 Контекст

FitPulse обрабатывает биометрические и медицинские данные пользователей. Для повышения точности классификации и персонализации планов требуется интеграция с внешними сервисами здоровья и синхронизация с медицинской картой.

Документация: `docs/medical-registration/`.

### 12.2 Acceptance Criteria

- Синхронизация с медицинской картой работает при наличии согласия пользователя.
- Все медицинские данные защищены по 152-ФЗ.

---

## 13. Регистрация приложения как медицинского

### 13.1 Контекст

FitPulse выходит за рамки wellness-приложения: plans генерируются на основе физиологического состояния, используются биометрические данные, есть интеграция с медицинскими сервисами. Это требует официальной регистрации как медицинского ПО/сервиса.

Документация: `docs/medical-registration/`.

**Связанные разделы**:

- Раздел 4 (Compliance: 152-ФЗ) — базовое compliance покрытие
- Раздел 12 (Интеграция с медицинскими сервисами) — техническая интеграция

### 13.2 Acceptance Criteria

- FitPulse зарегистрирован как медицинское ПО/сервис.
- Размещён сертификат/разрешение в разделе About/Legal.
- Политики обновлены и доступны до регистрации.
- Пользователи видят статус медицинского сервиса в интерфейсе.

---

## 14. Ежедневная адаптивная модификация плана

### 14.1 Контекст

Текущий сервер (2 vCPU / 4 ГБ RAM / 60 ГБ Storage, KVM, Ubuntu 26.04) не позволяет запускать ежедневное ML-переобучение без влияния на отзывчивость приложения.
Phase 1 покрывает базовую генерацию плана и классификацию состояния через `POST /api/v1/ml/chat`.
Ежедневная автоматическая модификация плана — это задача Phase 2, требующая отдельного планировщика/воркера и более мощной инфраструктуры.

DVC pipeline инициализирован (`dvc.yaml`, `params.yaml`), манифесты готовы.

### 14.2 Acceptance Criteria

- Планы пользователей автоматически пересматриваются раз в 24 часа на основе последних биометрических данных.
- Переобучение не влияет на p95 латентность API (< 2s).
- DVC-tracked модели версионируются и откатываются при деградации качества.
- План переобучения завершается за < 10 минут на выделенном воркере (2+ vCPU, 4+ ГБ RAM).

---

## 15. Диаграмма зависимостей

### 15.1 Контекст

Phase 2 состоит из нескольких крупных блоков, которые нельзя выполнять хаотично. Ниже — обязательный порядок зависимостей.

### 15.2 Зависимости

```text
[Security email + PGP] ─────────────────────────┐
                                                 ↓
[Vault + Secrets] ───────► [Secrets Rotation Automation] ───────► [Service Mesh]
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
| Vault + Secrets | Secrets Rotation Automation, Service Mesh (cert-manager) | Требует центрального хранилища секретов |
| PostgreSQL HA | Backup DR, Data Processor (production) | Требует стабильного Primary/Replica |
| Service Mesh | Observability (tracing), Canary Deployments | Требует sidecar-инъекции и PeerAuthentication |
| Security email + PGP | Bug Bounty, Pen Test, SECURITY.md обновление | Требует корпоративного ящика до публикации |
| Compliance 152-ФЗ | Medical services integration, Medical app registration | Требует шифрования и audit trail |

### 15.4 Параллельные работы

- **Vault + Secrets** можно начинать параллельно с **Infra provisioning** (на тестовом стенде)
- **CAPTCHA** не зависит от инфраструктуры, можно делать параллельно с **Compliance**
- **Bug Bounty program setup** не зависит от инфраструктуры, можно делать параллельно
- **Ежеквартальный внешний пентест** можно начинать после **Security email + PGP**, не зависит от инфраструктуры
- **Observability расширение** можно начинать после **Service Mesh**, но до полной миграции

---

## 16. Оценка стоимости (руб/мес)

### 16.1 Контекст

Все оценки указаны для российского хостинга (Yandex Cloud / Selectel / Timeweb VPS) и approximated. Точные цифры зависят от провайдера и региона.

### 16.2 Инфраструктура

| Компонент | Текущая стоимость | Новая стоимость | Δ |
| --- | --- | --- | --- |
| VPS (2 vCPU / 4 ГБ / 60 ГБ) | ~2 500 ₽/мес | ~3 500 ₽/мес (4 vCPU / 8 ГБ / 80 ГБ SSD) | +1 000 ₽/мес |
| Managed PostgreSQL (Yandex Managed) | — | ~2 500–4 000 ₽/мес | +2 500–4 000 ₽/мес |
| Vault (self-hosted на отдельном VPS) | — | ~1 500 ₽/мес (2 vCPU / 4 ГБ) | +1 500 ₽/мес |
| Backup storage (S3-compatible, 100 ГБ) | — | ~300 ₽/мес | +300 ₽/мес |
| Domain fittpulse.ru (первый год) | — | ~1 500 ₽/год | +200 ₽/мес |
| SSL-сертификат (Let's Encrypt) | — | 0 ₽/мес | 0 ₽/мес |
| **Итого инфраструктура** | **~1 500 ₽/мес** | **~8 000–10 500 ₽/мес** | **+6 500–9 000 ₽/мес** |

### 16.3 ML-сервисы

| Компонент | Стоимость | Примечание |
| --- | --- | --- |
| MLflow (self-hosted) | 0 ₽/мес | Запускается на существующем VPS |
| DVC remote storage | 0 ₽/мес | Локальный диск / S3-compatible |
| GPU-воркер (если нужен inference acceleration) | ~5 000–15 000 ₽/мес | Yandex Cloud GPU / Lambda Labs |
| **Итого ML** | **0–15 000 ₽/мес** | Зависит от необходимости GPU |

### 16.4 Security / Compliance

| Компонент | Стоимость | Примечание |
| --- | --- | --- |
| Corp email (Yandex 360 / Google Workspace) | ~300–600 ₽/мес за пользователя | 1–2 пользователя |
| PGP ключ / WKD | 0 ₽/мес | Self-hosted |
| Bug Bounty вознаграждения (опционально) | 0–10 000 ₽/мес | Зависит от бюджета |
| **Итого Security** | **300–10 600 ₽/мес** | — |

### 16.5 Итого Phase 2

| Сценарий | Стоимость/мес | Годовая стоимость |
| --- | --- | --- |
| Минимум (без GPU, без bug bounty) | ~8 500 ₽/мес | ~102 000 ₽/год |
| Рекомендуемый (с observability, без GPU) | ~12 000 ₽/мес | ~144 000 ₽/год |
| Максимальный (с GPU, bug bounty) | ~25 000–35 000 ₽/мес | ~300 000–420 000 ₽/год |

**Trade-off**: На текущем 2-vCPU / 4 ГБ сервере невозможно запустить Vault + Istio + PostgreSQL HA одновременно. Требуется апгрейд VPS до минимум 4 vCPU / 16 ГБ RAM или разделение на 2 VPS.

---

## 17. Resource Plan: FTE

### 17.1 Контекст

Phase 2 требует специализации, которой нет у единственного разработчика. Ниже — оценка человеко-часов и необходимых ролей.

### 17.2 Роли и ответственность

| Роль | Занятость | Ответственность |
| --- | --- | --- |
| **DevOps/Platform** | 0.8 FTE | VPS provisioning, k8s, Vault, PostgreSQL HA, Service Mesh, CI/CD |
| **Backend (Go)** | 0.6 FTE | Secrets integration, mTLS migration, admin panel, compliance endpoints |
| **ML/Data Engineer** | 0.4 FTE | DVC pipeline, adaptive retrain, model versioning |
| **Frontend** | 0.3 FTE | Achievements, Diet, Devices views из UI_SPECIFICATION |
| **Legal/Compliance** | 0.2 FTE | 152-ФЗ, медицинская регистрация, политики |
| **Security** | 0.2 FTE | Bug bounty, PGP, WAF rules, penetration testing |
| **Product/Design** | 0.1 FTE | Приоритизация фич, UI/UX approval |

### 17.3 Общие затраты

| Сценарий | FTE | Срок | Человеко-часы |
| --- | --- | --- | --- |
| Агрессивный (все параллельно) | 1.6 FTE | 8 недель | ~2 560 ч |
| Рекомендуемый (последовательный) | 0.8 FTE | 16 недель | ~2 560 ч |
| Консервативный (1 человек, 0.5 FTE) | 0.5 FTE | 32 недели | ~2 560 ч |

**Важно**: В текущем состоянии проект поддерживается 1 человеком (`@MAMUER`). Phase 2 **невозможна** без привлечения хотя бы одного дополнительного DevOps/Backend разработчика.

---

## 18. Migration Strategy

### 18.1 Контекст

Каждый major change в Phase 2 требует стратегии миграции без downtime. Ниже — per-component планы.

### 18.2 Vault Migration (Kubernetes Secrets → Vault)

**Подход**: Gradual migration с dual-read периодом.

1. **Week 1**: Развёртывание Vault в dev-окружении, настройка Kubernetes auth method
2. **Week 2**: Миграция 1–2 не критичных секретов (например, `SMTP_*`) на Vault, приложения читают из Vault, но fallback на K8s Secret
3. **Week 3**: Миграция всех секретов, dual-read: приложение читает из Vault, при недоступности — из K8s Secret
4. **Week 4**: Отключение K8s Secrets, все приложения читают только из Vault
5. **Rollback**: При проблемах с Vault — переключение обратно на K8s Secrets через environment variable `VAULT_ENABLED=false`

### 18.3 PostgreSQL HA Migration (Single → Patroni)

**Подход**: Rolling migration с использованием pg_basebackup.

1. **Week 1**: Развёртывание Patroni + etcd на отдельном VPS/поде
2. **Week 2**: Настройка streaming replication с текущего primary на новый Patroni cluster
3. **Week 3**: Переключение application connection string на Patroni VIP, тестирование failover
4. **Week 4**: Деcommission старого single PostgreSQL
5. **Rollback**: При проблемах — переключение connection string обратно на старый primary

### 18.4 Service Mesh Migration (hand-rolled mTLS → Istio/Linkerd)

**Подход**: Canary migration namespace-by-namespace.

1. **Week 1**: Установка Istio control plane в dedicated namespace, без sidecar-инъекции
2. **Week 2**: Включение sidecar-инъекции для 1 не критичного namespace (например, `ml-generator`)
3. **Week 3**: Постепенное включение для остальных namespaces, мониторинг latency/errors
4. **Week 4**: Отключение hand-rolled mTLS, полный переход на mesh
5. **Rollback**: При проблемах — отключение sidecar-инъекции, возврат к hand-rolled mTLS

### 18.5 Data Processor Migration (stub → production)

**Подход**: Blue-green deployment consumer'а.

1. **Week 1**: Развёртывание data-processor в production с `PREFETCH=1`, без обработки сообщений (consumer-only, Nack all)
2. **Week 2**: Включение обработки для 10% сообщений (sampling), мониторинг dead-letter queue
3. **Week 3**: Полный rollout, мониторинг lag и error rate
4. **Week 4**: Отключение legacy-публения в `biometric_events` из biometric-service (если publisher migrated)
5. **Rollback**: Отключение data-processor подов, сообщения накапливаются в RabbitMQ

---

## 19. Risk Register

### 19.1 Контекст

Каждый пункт Phase 2 имеет operational риски. Ниже — реестр рисков с fallback-стратегиями.

### 19.2 Риски

| ID | Риск | Вероятность | Влияние | Митигация | Fallback |
| --- | --- | --- | --- | --- | --- |
| R1 | Vault не справляется с нагрузкой при 100+ RPS | Средняя | Высокое | Load testing перед production; Vault cluster из 3 нод | Остаться на Kubernetes Secrets + внешний vault-агент |
| R2 | PostgreSQL HA failover работает некорректно | Средняя | Высокое | Ежеквартальные chaos tests; pg_basebackup проверка | Остаться на single PostgreSQL с ежедневными бэкапами |
| R3 | Istio потребляет > 1 ГБ RAM на control plane | Высокая | Среднее | Использовать Linkerd вместо Istio (легче) | Остаться на hand-rolled mTLS |
| R4 | ML retrain падает по памяти на 2 vCPU | Высокая | Среднее | Ограничить resources.limits; использовать swap | Перенести retrain на GitHub Actions / external GPU |
| R5 | 152-ФЗ compliance не достигнут | Средняя | Высокое | Юридическая экспертиза на этапе проектирования | Ограничить функционал для РФ-пользователей |
| R6 | Корпоративный email не получен (бюджет) | Средняя | Среднее | Использовать бесплатный Yandex 360 для бизнеса | Остаться на личном email с PGP |
| R7 | Bug bounty программа привлекает неточные репорты | Высокая | Низкое | Чёткий scope, triage-процесс | Игнорировать некорректные репорты |
| R8 | DVC remote не доступен из k8s | Средняя | Среднее | Настроить S3-compatible storage (MinIO) | Локальный DVC cache без remote |
| R9 | Service Mesh конфликтует с существующими Network Policies | Средняя | Высокое | Тестирование в dev перед production | Откат на hand-rolled mTLS |
| R10 | Adaptive daily retrain перегружает API | Высокая | Высокое | Очередь RabbitMQ + rate limiting на retrain job | On-demand retrain только по запросу пользователя |

### 19.3 Risk Response Plan

| Риск | Ответ | Trigger | Action |
| --- | --- | --- | --- |
| R1 | Mitigate | Vault latency P99 > 500ms | Масштабировать Vault cluster |
| R2 | Mitigate | Failover > 30s | Откат на single PostgreSQL |
| R3 | Avoid | Istio memory > 1.5 ГБ | Использовать Linkerd |
| R4 | Transfer | ML retrain OOM | Перенести на external CI |
| R5 | Accept | Legal costs > 500k ₽ | Ограничить функционал |
| R6 | Mitigate | Бюджет 0 ₽ | Использовать бесплатный email |
| R7 | Accept | Трафик < 10 reports/мес | Низкие затраты на triage |
| R8 | Mitigate | DVC unavailable | MinIO fallback |
| R9 | Mitigate | Mesh errors > 1% | Откат на hand-rolled mTLS |
| R10 | Avoid | API latency > 2s | On-demand только |

---

## 20. Exit Criteria для Phase 2

### 20.1 Контекст

Phase 2 завершается, когда выполнены все Must-have критерии. Should-have и Could-have могут быть перенесены на Phase 3.

### 20.2 Must-have (Phase 2 exit criteria)

| Критерий | Метрика | Приоритет |
| --- | --- | --- |
| Vault развёрнут и все секреты мигрированы | 0 секретов в Kubernetes Secrets | P0 |
| PostgreSQL HA с failover < 30s | RTO < 30s, RPO = 0 | P0 |
| Service Mesh активен между всеми сервисами | mTLS 100%, zero manual cert rotation | P0 |
| 152-ФЗ compliance документация готова | Политика утверждена, audit log 3 года | P0 |
| Security email заменён на корпоративный | Личный email удалён из SECURITY.md | P0 |
| CI/CD pipeline обновлён (govulncheck, Gitleaks, TruffleHog) | Все scans проходят, Security Gate PASS | P1 |
| Backup DR протестирован | Recovery drill раз в квартал, RTO < 1ч | P1 |
| Observability: Grafana + Alertmanager + Jaeger | Дашборды покрывают все critical endpoints, SLO алерты настроены, Jaeger развёрнут | P1 |
| Python сервисы инструментированы OTel | ml-generator отправляет трейсы в Jaeger | P1 |
| Correlation ID: trace ID ↔ логи | Все сервисы коррелируют трейсы с логами | P2 |
| On-call бот доставляет алерты в Telegram | SEV-1/SEV-2 доставлены в течение 1 минуты | P2 |

### 20.3 Should-have (Phase 2 exit criteria — желательно)

| Критерий | Метрика | Приоритет |
| --- | --- | --- |
| CAPTCHA интегрирован | Error rate 429 ↓ на 50% | P2 |
| Bug bounty программа запущена | ≥ 5 reports/мес | P2 |
| Ежеквартальный внешний пентест проведён | Отчёт с remediation plan | P2 |
| Adaptive daily plan retrain (on-demand) | Retrain завершается < 10 мин | P3 |

### 20.4 Could-have / Won't-have (переносится на Phase 3)

| Критерий | Причина переноса |
| --- | --- |
| Canary Deployments (Flagger) | Требует Istio + extensive testing, низкий приоритет для 1-сервисной архитектуры |
| Medical app registration в Минздраве | Юридический процесс 3-4 месяца, не зависит от технической реализации |
| GPU-ускорение для ML | Дорого, текущий объём данных не требует GPU |
| Full disaster recovery (warm standby на another VPS) | Требует второго VPS, дорого для учебного проекта |

---

## 21. Timeline с Milestones

### 21.1 Контекст

Phase 2 планируется на **12 недель** при нагрузке 0.5–0.8 FTE, с **10% buffer** (итого 13–14 недель). Ниже — детальный timeline с вехами.

### 21.2 Milestones

| Milestone | Срок | Deliverables | Exit criteria |
| --- | --- | --- | --- |
| **M1: Foundation** | Недели 1–2 | VPS upgrade, Vault deployed, corporate email | Vault отвечает < 10ms, email работает |
| **M2: Security Hardening** | Недели 3–4 | Secrets rotation automation, mTLS migration started, PGP key published | 0 секретов в K8s Secrets, PGP fingerprint в SECURITY.md |
| **M3: Database HA** | Недели 5–6 | PostgreSQL HA deployed, failover tested, backup DR | RTO < 30s, recovery drill пройден |
| **M4: Observability** | Недели 7–8 | Service mesh deployed, Grafana dashboards, Alertmanager | Дашборды покрывают 100% critical endpoints, алерты работают |
| **M5: Compliance** | Недели 9–10 | 152-ФЗ documentation, medical API, security email migrated | Политика утверждена, medical sync работает |
| **M6: Polish** | Недели 11–12 | CAPTCHA, bug bounty launch, adaptive retrain on-demand | Все Must-have критерии выполнены |

### 21.3 Gantt Chart

```text
Неделя:    1    2    3    4    5    6    7    8    9    10   11   12   13   14
VPS:       [████████████████████████████████████████████████████████████████████████████████████████████████]
Vault:          [████████████████████████████████████████████████████████████████████████████████████████████████]
Secrets:              [████████████████████████████████████████████████████████████████████████████████████████████████]
PostgreSQL HA:               [████████████████████████████████████████████████████████████████████████████████████████████████]
Service Mesh:                     [████████████████████████████████████████████████████████████████████████████████████████████████]
Observability:                           [████████████████████████████████████████████████████████████████████████████████████████████████]
Compliance:                                      [████████████████████████████████████████████████████████████████████████████████████████████████]
CAPTCHA:                                                  [████████████████████████████████████████████████████████████████████████████████████████████████]
Bug Bounty:                                                      [████████████████████████████████████████████████████████████████████████████████████████████████]
Pen Test:                                                              [████████████████████████████████████████████████████████████████████████████████████████████████]
Medical:                                                                      [████████████████████████████████████████████████████████████████████████████████████████████████]
Adaptive Retrain:                                                                 [████████████████████████████████████████████████████████████████████████████████████████████████]
```

### 21.4 Critical Path

```text
VPS upgrade → Vault → PostgreSQL HA → Service Mesh → Observability → Compliance → Medical
```

**Любая задержка в Critical Path задержит всю Phase 2 на 1–2 недели.**

### 21.5 Buffer

- **10% buffer** на непредвиденные проблемы (итого 13–14 недель вместо 12)
- **Еженедельный sync** для корректировки timeline
- **Go/No-go checkpoint** на Milestone 3: если PostgreSQL HA не справляется — откат на managed CloudSQL

---

## 22. Критерии приёмки Phase 2

### 22.1 Контекст

Phase 2 считается завершённой, когда выполнены все Must-have критерии из раздела 20.2.

### 22.2 Checklist

- [ ] Vault развёрнут, все секреты мигрированы, ротация работает
- [ ] PostgreSQL HA с Patroni, failover < 30s протестирован
- [ ] Service Mesh (Istio/Linkerd) активен, strict mTLS включён
- [ ] Observability: Grafana + Alertmanager + дашборды готовы
- [ ] 152-ФЗ compliance документация утверждена
- [ ] Security email migrated на корпоративный ящик
- [ ] CI/CD pipeline обновлён: govulncheck, Gitleaks, TruffleHog, Security Gate PASS
- [ ] Backup DR протестирован, recovery drill пройден
- [ ] CAPTCHA интегрирован, error rate 429 ↓ на 50%
- [ ] Bug bounty программа запущена, PGP ключ опубликован
- [ ] Ежеквартальный внешний пентест проведён, critical/high уязвимости исправлены

### 22.3 Go/No-go Criteria

| Критерий | Go | No-go |
| --- | --- | --- |
| Vault latency | P99 < 50ms | P99 > 200ms → откат |
| PostgreSQL failover | RTO < 30s | RTO > 60s → откат на single |
| Service Mesh overhead | Memory < 500MB/pod | Memory > 1GB/pod → Linkerd вместо Istio |
| Budget | ≤ 12 000 ₽/мес | ≥ 20 000 ₽/мес → сокращение scope |

---

## 23. Phase 3 Preview

### 23.1 Что точно не входит в Phase 2

- Canary Deployments (Flagger + Argo Rollouts)
- Full medical app registration в Минздраве
- GPU-ускорение для ML inference
- Multi-region DR (требует второго датацентра)
- Advanced ML: reinforcement learning для адаптации планов

### 23.2 Что потенциально перейдёт в Phase 3

- Service Mesh → полноценный Istio с traffic shaping
- Vault → HSM-backed key management
- Observability → OpenTelemetry Collector + Thanos
- ML → online learning с feedback loop

### 23.3 Предварительный объём Phase 3

| Этап | Срок | Ответственный |
| --- | --- | --- |
| Canary Deployments | 2–3 недели | DevOps/Backend |
| Medical registration | 3–4 недели | Legal |
| Advanced ML (RL) | 3–4 недели | ML Engineer |
| Multi-region DR | 4–6 недель | DevOps |
| Full Service Mesh (Istio) | 2–3 недели | Platform |

**Итого Phase 3: 3–4 месяца**

---

## 24. Расширение поддерживаемых устройств

### 24.1 Контекст

Текущие интегрированные источники здоровья: Open Wearables (агрегатор Apple Health, Garmin, Health Connect и др.). Samsung Galaxy Watch и Huawei Watch D2 поддерживаются через Open Wearables в roadmap Phase 2.

Open Wearables интеграция уже завершена (P0).

### 24.2 План

| Этап | Источник/устройство | Срок | Приоритет | Задачи |
| --- | --- | --- | --- | --- |
| 1 | Open Wearables | Завершён | P0 | Агрегация данных здоровья через единый webhook |
| 2 | Samsung Galaxy Watch | 3–4 недели | P2 | Через Open Wearables / Samsung Health Connect |
| 3 | Huawei Watch D2 | 3–4 недели | P2 | Через Open Wearables / Huawei Health Kit |

### 24.3 Acceptance Criteria

- Каждое устройство имеет working Open Wearables integration (aggregator → webhook → biometric-service)
- Минимум 3 метрики (heart_rate, spo2, sleep) синхронизируются автоматически
- Данные поступают через `POST /api/v1/integrations/open-wearables/webhook` в biometric-service
- UI отображает статус подключения и последнюю синхронизацию

### 24.4 Архитектурные ограничения

- Biometric-service остаётся универсальным: принимает webhook от Open Wearables, валидирует, сохраняет в `biometric_data`
- Device-aggregator используется как легковесный webhook-forwarder для Open Wearables
- Прямые OAuth-интеграции (Fitbit, Withings, Flo, OKOK) удалены из кодовой базы

---

## 25. Полноценная двухфакторная верификация через Google

### 25.1 Контекст

Чтобы разблокировать полноценный вход для всех пользователей через Google OAuth 2.0 с production-статусом consent screen, требуется:

- подтверждённый домен, принадлежащий проекту;
- живой homepage на этом домене;
- privacy policy и terms of service на том же домене;
- branding verification пройдена.

**Связанные разделы**:

- Раздел 9 (Security Reporting) — OAuth integration требует готовой инфраструктуры
- Раздел 6 (Infrastructure as Code) — TLS через cert-manager, Ingress

### 25.2 Предпосылки

- DNS-записи домена указывают на VPS / внешний load balancer, где поднят кластер k3s.
- В Google Cloud Console:
  - добавлен `fittpulse.ru` в **Authorized domains** для проекта `fitpulse-1780824080979`;
  - настроен OAuth 2.0 Client ID (Web application) с authorised redirect URIs:
    - `https://fittpulse.ru/api/v1/auth/google/callback`
  - в consent screen указаны:
    - Homepage: `https://fittpulse.ru`
    - Privacy Policy: `https://fittpulse.ru/privacy`
    - Terms of Service: `https://fittpulse.ru/terms`
- На VPS / в k8s:
  - cert-manager выписывает TLS-сертификат для `fittpulse.ru`;
  - ingress/routes проксируют `/`, `/privacy`, `/terms` на gateway;
  - SPA на React отдаёт главную страницу и юридические страницы.

### 25.3 Задачи

| Этап | Задача | Срок | Приоритет |
| --- | --- | --- | --- |
| 2 | Подготовить инфраструктуру: TLS через cert-manager, Ingress/Route для `fittpulse.ru` | 2–3 дня | P0 |
| 3 | Обновить `SECURITY.md`, CI конфиги, deployment manifests под новый домен | 1 день | P1 |
| 4 | В Google Cloud Console обновить authorized domains, consent screen URLs, redirect URIs | 1 день | P0 |
| 5 | Пройти branding verification (логотип 120×120, скриншоты, описание) | 1–2 дня | P0 |
| 6 | Перевести consent screen из testing в production | 1 день | P0 |

### 25.4 Acceptance Criteria

- `https://fittpulse.ru`, `/privacy`, `/terms` доступны из внешней сети по HTTPS без авторизации.
- Google OAuth consent screen находится в статусе **production**.
- Вход через Google работает для любых пользователей без ограничения в 100 аккаунтов и без 7-дневного истечения токена.

### 25.5 Риски и mitigation

| Риск | Вероятность | Воздействие | Mitigation |
| --- | --- | --- | --- |
| Домен не прошёл верификацию | Средняя | Высокое | Предварительно submit в Google, подготовить все материалы (логотип, скриншоты, описание) |
| Проблемы с DNS propagation | Средняя | Среднее | Использовать TTL 300s на время переезда, мониторить `dig`/`nslookup` |
| Просрочение сертификата cert-manager | Низкая | Низкое | cert-manager автоматически продлевает; настроить алерты за 7 дней до истечения |
