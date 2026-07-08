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

## 2. Service Mesh

### 2.1 Контекст

Phase 1 покрывает базовый mTLS между микросервисами на уровне gRPC (TLS 1.3, hand-rolled certs из Kubernetes Secret). Phase 2 переводит внутренние коммуникации на полноценный service mesh (Istio/Linkerd) с автоматической ротацией сертификатов, SPIFFE ID и распределённым трейсингом.

### 2.2 Задачи

1. Переход с hand-rolled mTLS на Istio (полноценный mesh) или lightweight вариант (SPIRE для SPIFFE ID / cert-manager + linkerd)
2. Настройка `cert-manager` + внутреннего CA для автоматической ротации
3. Включение strict mTLS для всех namespace (через `linkerd` или `istio`) и отказ от статических сертификатов в Secret
4. Добавление `PeerAuthentication` и `AuthorizationPolicy`
5. Включение tracing (Jaeger/Zipkin) через mesh sidecar

### 2.3 Acceptance Criteria

- mTLS активен между всеми сервисами через service mesh (Istio/Linkerd) с автоматической ротацией сертификатов через cert-manager
- Статические сертификаты в Kubernetes Secret удалены, все сертификаты генерируются динамически и монтируются через sidecar
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
4. Настройка HAProxy/ProxySQL как единой точки входа (connection pooling, health checks, read/write splitting)
5. Интеграция с мониторингом: `pg_stat_replication`, `pg_stat_activity`

### 3.3 Acceptance Criteria

- RTO < 30 секунд при отказе primary.
- RPO = 0 при использовании синхронных реплик в разных AZ (availability zones). **Trade-off**: синхронные реплики в разных AZ увеличивают write latency на 50-200мс из-за ожидания подтверждения от реплик перед commit. **Примечание**: синхронные реплики в пределах одного VPS не защищают от отказа хоста; для true RPO=0 требуется географическое распределение (multi-AZ/multi-region).
- Автоматическое восстановление из бэкапа протестировано (ежеквартальные Game Days).

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

## 7. Наблюдаемость и SLO

### 7.1 Контекст

- Для GA-релиза необходимы расширенные возможности наблюдаемости.
- Production-окружения содержат: gateway, user-service, biometric-service, training-service, device-connector, device-aggregator, classifier, ml-generator, data-processor.
- Production domain: fittpulse.duckdns.org
- Актуальные сервисы и endpoints:
  - Portal: https://portal.fittpulse.duckdns.org
  - API: https://api.fittpulse.duckdns.org/api/v1/
  - Health checks: /health, /confirm, /logout
  - ML endpoints: /ml/classify, /ml/generate-plan

### 7.2 Задачи

1. **ОПРЕДЕЛИТЬ SLI/SLO**:
   - Целевая доступность: 99.9% в месяц (исключая плановые работы)
   - Целевая латентность: p95 < 2s для всех критических endpoints:
     - /api/v1/auth/login
     - /api/v1/biometrics
     - /ml/generate-plan
   - Ошибочный бюджет: 0.1% в месяц
     - Отслеживается через метрики ошибок 5xx
     - Burn rate ошибочного бюджета отслеживается в Grafana

2. **ИНСТРУМЕНТАЦИЯ АЛЕРТИНГА**:
   - Настроить Alertmanager для критических алертов
   - Интегрировать Telegram webhook для первичных уведомлений
   - Реализовать политики эскалации для критических инцидентов
   - Настроить retention: история алертов хранится 90 дней

3. **РАСПРЕДЕЛЁННЫЙ ТРЕЙСИНГ**:
   - Добавить Jaeger/Zipkin sidecar-инструментацию для всех FastAPI сервисов
   - Инструментировать gRPC вызовы в Go сервисах через OpenTelemetry
   - Обеспечить propagation trace context во всех межсервисных вызовах
   - Настроить sampling на уровне >=1% для захвата репрезентативного трафика

4. **ДАШБОРДЫ**:
   - Grafana дашборды:
     - **Доступность** (p99 за 7-дневное окно)
     - **Латентность** (p50/p95/p99 процентили + трендовый анализ)
     - **Ошибки** (HTTP 5xx rate + обнаружение всплесков)
     - **Burn rate ошибочного бюджета** (отслеживает пополнение/восстановление)
     - **Матрица здоровья сервисов** со статус-индикаторами
   - RED метрики:
     - **R**ate (запросов/сек), **E**rror rate, **D**uration (процентили латентности)

### 7.3 Acceptance Criteria

- Production-сервисы выдерживают 100+ concurrent requests
- Экосистема мониторинга поддерживает 5-секундный интервал сбора для критических метрик
- Все критические условия ошибок вызывают алерты в течение 30 секунд
- Набор дашбордов покрывает все требуемые метрики с визуализациями для алертов
- Метрики ошибочного бюджета SLO приводят к автоматическому применению политик

## 8. Infrastructure as Code (Total rewrite)

### 8.1 Контекст

Phase 1 использует Kustomize + inline-скрипты для k3s. Текущая инфраструктура (1 vCPU / 2 ГБ RAM / 30 ГБ Storage, KVM, РФ) ограничена для production-нагрузок, поэтому Phase 2 требует:

- аппаратного апгрейда/реновации VPS (обязательно, для поддержки HA-компонентов, Vault, Istio и бóльшего числа подов)
- после увеличения ресурсов VPS необходимо пересчитать параметры Argon2id

Примечание: текущие параметры Argon2id (memory 64 MB, iterations 3, parallelism 1) установлены с учётом ограничений текущего 1-vCPU сервера. После переезда на более мощный VPS параметры требуется пересчитать. **Best Practice**: реализовать автоматический benchmark Argon2id при старте сервиса (или в CI/CD) для калибровки `memory` и `iterations` под фактические `resources.limits.memory` и CPU-квоты пода, сохраняя время хеширования в пределах 500мс-1с. Использовать Go-библиотеку `github.com/alexedwards/argon2id` с auto-tuning или Rust crate `argon2` с feature `auto-tune`.

- Terraform для управления инфраструктурой
- ArgoCD или Flux для GitOps
- единый репозиторий конфигураций

### 8.2 Задачи

1. Аренда более мощного VPS на территории РФ для покрытия нагрузок Phase 2.
2. Terraform модули: VPS, K8s cluster, DB, Vault
3. ArgoCD для declarative deploy'а
4. Sealed Secrets или External Secrets Operator для секретов
5. Policy as Code: OPA Gatekeeper

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

1. Интеграция **Flagger** (предпочтительнее Argo Rollouts для Linkerd/Istio/Nginx Ingress) с существующим GitOps-пайплайном.
2. Конфигурация canary-стратегии: 5% → 25% → 50% → 100% traffic с автоматическим анализом метрик (Prometheus) между шагами.
3. Автоматический rollback при превышении порога ошибок (error rate > baseline + 1% или latency p99 > SLO).
4. Интеграция с Observability: использование `MetricTemplate` в Flagger для запросов к Prometheus/Thanos.
5. Документация runbook: manual promotion, manual rollback, pause, анализ логов canary-пода.
6. Настройка `AnalysisTemplate` для кастомных проверок (например, проверка ML-моделей на drift) и интеграция с Slack-уведомлениями при pause для ручного вмешательства.

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

2. Оценить бюджет/возможность денежного вознаграждения.
3. Настроить PGP key fingerprint для шифрования чувствительных отчётов об уязвимостях, чтобы предотвратить перехват информации о zero-day уязвимостях при передаче по email.
4. **Вариант B (self-hosted)**:
    - Создать `BUG_BOUNTY.md` с полной политикой;
    - Добавить в `SECURITY.md` раздел `## Отчеты об уязвимостях` с email, PGP key, SLA.
    - Опубликовать PGP-ключ через WKD (Web Key Directory) для автоматического обнаружения ключа почтовыми клиентами (GnuPG, Thunderbird).
5. **Вариант C (platform)**:
    - Зарегистрировать программу на HackerOne / Bugcrowd / Intigriti;
    - Определить in-scope: `fittpulse.duckdns.org`, API endpoints;
    - Интегрировать алерты в Slack/Telegram;
    - Добавить ссылку на программу в `SECURITY.md`.

### 11.3 Acceptance Criteria

- PGP key fingerprint опубликован в `SECURITY.md` и `BUG_BOUNTY.md` для безопасной коммуникации.
- Исследователи могут использовать PGP для шифрования отчётов об уязвимостях.
- Документ `BUG_BOUNTY.md` готов и public.
- Решение по Варианту B или C принято.
- SLA по ответу задокументирован и публичен.

## 12. Корпоративный почтовый ящик для security-отчётов

### 12.1 Контекст

Текущий security reporting использует личный email (`mihnikolaenko12@yandex.ru`), что нарушает best practices:

- нет контроля доступа и audit trail на уровне почты;
- риски компрометации личного аккаунта;
- неформальный домен снижает доверие к программе;
- сложность с делегированием доступа в случае смены ответственного.

### 12.2 Задачи

1. Приобрести корпоративный домен/почтовый аккаунт для security отчётов (например, через Yandex 360 для бизнеса или Google Workspace).
2. Выделить group alias `security@fitpulse.app` с включенным аудитом логов входа и обязательным hardware 2FA (YubiKey) для всех членов security-команды.
3. Обновить `SECURITY.md`, `BUG_BOUNTY_SCOPE.md`, `BUG_BOUNTY.md` с новым корпоративным контактом и удалить упоминания личных ящиков.
4. Сгенерировать и опубликовать PGP-ключ (Ed25519/Curve25519) для корпоративного ящика, загрузить в публичные keyserver'ы.
5. Документировать процесс доступа к ящику, ротации ключей и offboarding сотрудников.
6. Настроить backup-коды и recovery process для hardware 2FA на случай потери YubiKey (хранение backup-кодов в Vault с ограниченным доступом).

### 12.3 Acceptance Criteria

- Все security-отчёты принимаются на корпоративный ящик.
- Личный email больше не указан как primary контакт.
- PGP-ключ опубликован и защищает in-transmit отчёты.
- Есть процедура ротации доступа к ящику.

---

## 13. CAPTCHA (Cloudflare Turnstile)

### 13.1 Контекст

Phase 1 использует жёсткий rate limiting (при превышении порога — блокировка). Это создаёт риск отзыва legitimate-пользователей при ложных срабатываниях, cross-user NAT и burst-трафике. CAPTCHA при превышении порога ошибок позволяет подтвердить человечность, не блокируя полностью.

### 13.2 Задачи

1. Интеграция Cloudflare Turnstile (fallback: hCaptcha) на уровне Gateway.
2. Определение триггеров для показа CAPTCHA (например, ошибки 429 повторяются с одного IP/клиента).
3. Хранение state токена с коротким TTL.
4. Логирование капчи и её успешного решения, корреляция с alertmanager.
5. Обеспечение accessibility (WCAG 2.1 AA): CAPTCHA должна поддерживать screen readers и keyboard navigation.
6. Fallback для пользователей без JS: альтернативный механизм подтверждения (например, email-верификация или honeypot-поля).

### 13.3 Acceptance Criteria

- При превышении порога rate limit пользователь видит CAPTCHA, а не жёсткий блок.
- Успешное решение CAPTCHA снимает ограничение на фиксированный период (например, 5 минут).
- CAPTCHA логируется с correlationId и участвует в RED metrics.

### 13.4 Privacy Considerations

**Cloudflare Turnstile**:

- **Data collection**: Turnstile собирает telemetry (IP, user agent, browser fingerprint) для проверки человечности. Это может конфликтовать с 152-ФЗ и GDPR.
- **Mitigation**:
  - Использовать Turnstile в "managed" режиме (минимальная telemetry) вместо "interactive".
  - Добавить уведомление в Privacy Policy о использовании CAPTCHA-сервиса third-party.
  - Для EU/RF-пользователей: предложить opt-out через email-верификацию (fallback из раздела 13.2 задача 6) или self-hosted hCaptcha с минимальной telemetry.
  - **Best Practice**: реализовать feature flag `CAPTCHA_PROVIDER` с значениями `mcaptcha`, `hcaptcha`, `cloudflare`. Для пользователей из РФ/ES по умолчанию использовать self-hosted `mCaptcha` для полного compliance с 152-ФЗ и GDPR.
  - Альтернатива: математическая CAPTCHA (менее безопасна, но privacy-friendly).
- **Compliance check**: Проконсультироваться с Legal о допустимости передачи IP в Cloudflare для пользователей из РФ/ES.

---

## 14. Secrets Rotation Automation

### 14.1 Контекст

Phase 2 внедряет HashiCorp Vault для хранения секретов, но требуется автоматизация ротации и инъекции секретов в поды без перезапуска сервисов.

### 14.2 Задачи

1. Интеграция Vault Agent Injector для автоматической инъекции секретов в поды через sidecar-контейнер.
2. Настройка `vault.hashicorp.com/agent-inject` аннотаций для всех сервисов.
3. Автоматическая ротация секретов при истечении TTL (настройка `lease_duration` и `renewal`).
4. Интеграция с Kubernetes ServiceAccount для аутентификации в Vault (Kubernetes auth method).
5. **Hot reload**: использовать `Stakater Reloader` (аннотация `reloader.stakater.com/auto: "true"`) для автоматического rolling update при ротации секретов во Vault/K8s Secrets, что гарантирует атомарное применение без downtime и race conditions.
6. Мониторинг: алерты при неудачной ротации или истечении TTL.

### 14.3 Acceptance Criteria

- Все секреты инжектируются в поды автоматически без перезапуска сервисов.
- Ротация происходит прозрачно для приложений.
- Алерты срабатывают при проблемах с ротацией.

---

## 15. Интеграция с медицинскими сервисами

### 15.1 Контекст

FitPulse обрабатывает биометрические и медицинские данные пользователей. Для повышения точности классификации и персонализации планов требуется интеграция с внешними сервисами здоровья и синхронизация с медицинской картой.

### 15.2 Задачи

1. API для синхронизации с медицинской картой пользователя (с согласия субъекта).
2. Шифрование всех медицинских данных at rest и in transit.

### 15.3 Acceptance Criteria

- Синхронизация с медицинской картой работает при наличии согласия пользователя.
- Все медицинские данные защищены по 152-ФЗ.

---

## 16. Регистрация приложения как медицинского

### 16.1 Контекст

FitPulse выходит за рамки wellness-приложения: plans генерируются на основе физиологического состояния, используются биометрические данные, есть интеграция с медицинскими сервисами. Это требует официальной регистрации как медицинского ПО/сервиса.

### 16.2 Задачи

1. Юридическая экспертиза: соответствие FitPulse определению медицинского сервиса/продукта в РФ.
2. Подготовка технической документации для регистрации.
3. Подача заявки в Минздрав/Росздравнадзор.
4. Получение сертификата/разрешения на медицинское использование.
5. Обновление политик конфиденциальности и обработки данных с учётом медицинского статуса.
6. Информирование пользователей о медицинском статусе сервиса через интерфейс.

### 16.3 Acceptance Criteria

- FitPulse зарегистрирован как медицинское ПО/сервис.
- Размещён сертификат/разрешение в разделе About/Legal.
- Политики обновлены и доступны до регистрации.
- Пользователи видят статус медицинского сервиса в интерфейсе.

---

## Сроки (оценочно)

|Этап|Срок|Ответственный|
|---|---|---|
|Infra provisioning (VPS + k8s)|2-3 недели|DevOps|
|Vault + Secrets|1 неделя|DevOps/Backend|
|PostgreSQL HA|2 недели|DBA/DevOps|
|mTLS / Service Mesh|2 недели|Platform|
|Compliance (152-ФЗ)|4-6 недель|Legal/DevOps|
|Backup DR|1-2 недели|DevOps|
|Observability расширение|1 неделя|Platform|
|CAPTCHA (Cloudflare Turnstile)|3-5 дней|Backend/Platform|
|Security email + PGP|3 дня|DevOps/Security|
|Bug Bounty program setup|1 неделя|Security/Legal|
|Medical services integration|2 недели|Backend/Product|
|Medical app registration|3-4 недели|Legal/DevOps|

### Итого Phase 2: 3-4 месяца

В Phase 2 может работать параллельно над новыми фичами из `docs/UI_SPECIFICATION.md` (Achievements, Diet, Devices) — они не зависят от инфраструктурных изменений.

**Важно**: Раздел 12 (корпоративная почта) и раздел 11.2 задача 3 (PGP-ключ) должны быть выполнены **до** публикации `SECURITY.md` с новыми контактами. Текущий `SECURITY.md` указывает личный email `mihnikolaenko12@yandex.ru`, что нарушает best practices и создаёт single point of failure.
