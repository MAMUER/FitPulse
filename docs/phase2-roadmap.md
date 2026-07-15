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

### 3.2 Задачи

1. Развёртывание Patroni + etcd (или managed Aurora/CloudSQL)
2. Настройка 1 primary + 2 synchronous replicas
3. Настройка pg_basebackup + WAL-архивации в S3
4. Настройка HAProxy/ProxySQL как единой точки входа (connection pooling, health checks, read/write splitting)
5. Интеграция с мониторингом: `pg_stat_replication`, `pg_stat_activity`

### 3.3 Acceptance Criteria

- RTO < 30 секунд при отказе primary.
- RPO = 0 при использовании синхронных реплик в разных AZ (availability zones). **Trade-off**: синхронные реплики в разных AZ увеличивают write latency на 50-200мс из-за ожидания подтверждения от реплик перед commit. Синхронные реплики в пределах одного VPS не защищают от отказа хоста; для true RPO=0 требуется географическое распределение (multi-AZ/multi-region).
- Автоматическое восстановление из бэкапа протестировано (ежеквартальные Game Days).

---


## 4. Compliance: 152-ФЗ

### 4.1 Контекст

152-ФЗ «О персональных данных» требует:

- шифрование at rest и in transit
- хранение данных на территории РФ
- аудит действий с ПДн
- механизмы реализации прав субъекта (доступ, удаление)

### 4.2 Задачи
1. Расширить retention ELK-логов до 3 лет для соответствия 152-ФЗ (текущий retention в `docs/ARCHITECTURE.md`: 90 дней).
2. Подготовка документации (Политика обработки ПДн, Инструкция по работе с инцидентами, DPIA).

### 4.3 Acceptance Criteria

- Все персональные данные (email, biometric) шифруются в БД и в transit
- Audit log доступен для запросов Роскомнадзора
- Утверждена локальная политика безопасности

---

## 5. Backup & Recovery Strategy

### 5.1 Контекст

Phase 1 уже реализовала базовый бэкап, инкрементальные WAL-бэкапы и PITR (см. `docs/ARCHITECTURE.md` и `docs/adr/0009-security-infrastructure-hardening.md`). Phase 2 фокусируется на географической репликации, шифровании и автоматизации тестов восстановления.

### 5.2 Задачи

1. Шифрование бэкапов (AES-256, ключ в Vault)
2. Тестовый стенд восстановления (restore-to-clone)
3. Scheduled Chaos tests: отключение primary БД, Valkey master, Vault

### 5.3 Acceptance Criteria

- Восстановление за < 1 час
- Бэкапы реплицируются в 2 географических зоны
- Recovery drill — раз в квартал с публичным отчётом

---

## 6. Наблюдаемость и SLO

### 6.1 Контекст

- Для GA-релиза необходимы расширенные возможности наблюдаемости.
- Production-окружения содержат: gateway, user-service, biometric-service, training-service, device-connector, device-aggregator, classifier, ml-generator, data-processor.
- Production domain: fittpulse.duckdns.org
- Актуальные сервисы и endpoints:
  - Portal: https://fittpulse.duckdns.org
  - API: https://fittpulse.duckdns.org:8443/api/v1/
  - Health checks: /health, /confirm, /logout
  - ML endpoints: /ml/classify, /ml/generate-plan

### 6.2 Задачи

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

 5. **RECORDING RULES И ALERTMANAGER РОУТИНГ**:
    - Настроить recording rules для pre-aggregated метрик (histogram quantiles, error rate, burn rate) для снижения нагрузки на Prometheus и ускорения запросов Grafana.
    - Заменить stub webhook на полноценные маршруты: Telegram webhook для первичных уведомлений, Slack/PagerDuty для production-каналов.
    - Настроить retention: история алертов хранится 90 дней.

 6. **ON-CALL РОТАЦИЯ**:
    - Внедрить Grafana OnCall или аналогичный инструмент для управления дежурствами.
    - Настроить графики ротации, эскалацию по цепочке (SEV-1 → on-call engineer → Tech Lead → CTO) и уведомления через Slack/Telegram/PagerDuty.
    - Интегрировать с Alertmanager: автоматическое создание инцидентов, acknowledgement, post-incident review tracking.
    - Настроить handoff-процедуры при смене дежурного.

### 6.3 Acceptance Criteria

- Production-сервисы выдерживают 100+ concurrent requests
- Экосистема мониторинга поддерживает 5-секундный интервал сбора для критических метрик
- Все критические условия ошибок вызывают алерты в течение 30 секунд
- Набор дашбордов покрывает все требуемые метрики с визуализациями для алертов
- Метрики ошибочного бюджета SLO приводят к автоматическому применению политик

## 7. Infrastructure as Code (Total rewrite)

### 7.1 Контекст

Phase 1 использует Kustomize + inline-скрипты для k3s. Текущая инфраструктура (1 vCPU / 2 ГБ RAM / 30 ГБ Storage, KVM, РФ) ограничена для production-нагрузок, поэтому Phase 2 требует:

- аппаратного апгрейда/реновации VPS (обязательно, для поддержки HA-компонентов, Vault, Istio и бóльшего числа подов)
- после увеличения ресурсов VPS необходимо пересчитать параметры Argon2id

Текущие параметры Argon2id (memory 64 MB, iterations 3, parallelism 1) установлены с учётом ограничений текущего 1-vCPU сервера. После переезда на более мощный VPS параметры требуется пересчитать. **Best Practice**: реализовать автоматический benchmark Argon2id при старте сервиса для калибровки `memory`, `iterations` и `parallelism` (рекомендуется `parallelism` >= 2 для эффективного использования многоядерности CPU) под фактические `resources.limits`, сохраняя время хеширования в пределах 500мс–1с. Реализовать кастомный benchmark-цикл при старте сервиса (или использовать `github.com/go-crypt/go-crypt`), так как `github.com/alexedwards/argon2id` не имеет встроенного auto-tuning.

- Terraform для управления инфраструктурой
- ArgoCD или Flux для GitOps
- единый репозиторий конфигураций

### 7.2 Задачи

1. Аренда более мощного VPS на территории РФ для покрытия нагрузок Phase 2.
2. Terraform модули: VPS, K8s cluster, DB, Vault
3. ArgoCD для declarative deploy'а
4. Sealed Secrets или External Secrets Operator для секретов
5. Policy as Code: OPA Gatekeeper

---

## 8. Disaster Recovery

### 8.1 Контекст

Нужны сценарии восстановления на случай loss of region/datacenter. Синхронные реплики в пределах одного VPS не защищают от отказа хоста; для true RPO=0 требуется географическое распределение (multi-AZ/multi-region).

### 8.2 Задачи

1. Документация RTO/RPO по каждому сервису
2. Автоматический DR failover (warm standby на another VPS)
3. Тестирование DR раз в полгода
4. Восстановление данных: RTO < 4 часов, RPO < 15 минут

---

## 9. Canary Deployments

### 9.1 Контекст

Текущий деплой — монолитный rollover на все поды одновременно. Отсутствие gradual rollout повышает риск даунтайма при регрессах.

### 9.2 Задачи

1. Интеграция **Flagger** (предпочтительнее Argo Rollouts для Linkerd/Istio/Nginx Ingress) с существующим GitOps-пайплайном.
2. Конфигурация canary-стратегии: 5% → 25% → 50% → 100% traffic с автоматическим анализом метрик (Prometheus) между шагами.
3. Автоматический rollback при превышении порога ошибок (error rate > baseline + 1% или latency p99 > SLO).
4. Интеграция с Observability: использование `MetricTemplate` в Flagger для запросов к Prometheus/Thanos.
5. Документация runbook: manual promotion, manual rollback, pause, анализ логов canary-пода.
6. Настройка `AnalysisTemplate` для кастомных проверок (например, проверка ML-моделей на drift) и интеграция с Slack-уведомлениями при pause для ручного вмешательства.

### 9.3 Acceptance Criteria

- Любой deployment в production проходит через canary-фазу автоматически
- Rollback происходит без участия человека при error rate > baseline + 1%
- Время canary-фазы ≤ 10 минут до full rollout

## 10. Bug Bounty / Researcher Program
### 10.1 Контекст

Базовая self-hosted политика уже реализована: созданы `BUG_BOUNTY_SCOPE.md` и раздел в `SECURITY.md`, определены in-scope/out-of-scope цели и transparent SLA по ответу (best effort). Однако программа работает без бюджета, а отчёты принимаются на личный email, что создаёт operational риски. В Phase 2 требуется усилить криптографическую защиту отчётов и оценить возможность перехода на профессиональные платформы или выделения бюджета.

### 10.2 Задачи

1. **Оценка бюджета**: проанализировать возможность выделения даже символического бюджета на вознаграждения или мерч для исследователей.
2. **Криптографическая защита (PGP)**: сгенерировать и опубликовать PGP key fingerprint (Ed25519/Curve25519) в `SECURITY.md` и `BUG_BOUNTY_SCOPE.md` для шифрования чувствительных отчётов об уязвимостях (защита от перехвата zero-day при передаче по email).
3. **WKD (Web Key Directory)**: опубликовать PGP-ключ через WKD для автоматического обнаружения ключа почтовыми клиентами (GnuPG, Thunderbird).
4. **Оценка платформенной интеграции**: рассмотреть целесообразность миграции с self-hosted (GitHub Advisory + email) на HackerOne / Bugcrowd / Intigriti (включая интеграцию алертов в Slack/Telegram), если появится бюджет.
5. **Миграция на корпоративную почту**: см. раздел замена личного email на `security@fitpulse.app` с hardware 2FA.
### 10.3 Acceptance Criteria

- PGP key fingerprint опубликован в `SECURITY.md` и `BUG_BOUNTY_SCOPE.md`.
- PGP-ключ настроен и доступен через WKD.
- Принято финальное решение по бюджету и/или платформенной интеграции (с документированным обоснованием).
- Личный email заменён на корпоративный alias.

## 11. Корпоративный почтовый ящик для security-отчётов

### 11.1 Контекст

Текущий security reporting использует личный email (`mihnikolaenko12@yandex.ru`), что нарушает best practices:

- нет контроля доступа и audit trail на уровне почты;
- риски компрометации личного аккаунта;
- неформальный домен снижает доверие к программе;
- сложность с делегированием доступа в случае смены ответственного.

### 11.2 Задачи

1. Приобрести корпоративный домен/почтовый аккаунт для security отчётов (например, через Yandex 360 для бизнеса или Google Workspace).
2. Выделить group alias `security@fitpulse.app` с включенным аудитом логов входа и обязательным hardware 2FA (YubiKey) для всех членов security-команды.
3. Обновить `SECURITY.md`, `BUG_BOUNTY_SCOPE.md` с новым корпоративным контактом и удалить упоминания личных ящиков.
4. Сгенерировать и опубликовать PGP-ключ (Ed25519/Curve25519) для корпоративного ящика, загрузить в публичные keyserver'ы.
5. Документировать процесс доступа к ящику, ротации ключей и offboarding сотрудников.
6. Настроить backup-коды и recovery process для hardware 2FA на случай потери YubiKey (хранение backup-кодов в Vault с ограниченным доступом).

### 11.3 Acceptance Criteria

- Все security-отчёты принимаются на корпоративный ящик.
- Личный email больше не указан как primary контакт.
- PGP-ключ опубликован и защищает in-transmit отчёты.
- Есть процедура ротации доступа к ящику.

---

## 12. Ежеквартальный внешний пентест

### 12.1 Контекст

Внутренние сканы (gosec, Trivy, govulncheck, Gitleaks, TruffleHog) покрывают статический анализ кода, зависимости и конфигурации, но не находят уязвимости бизнес-логики, цепочки атак, ошибок авторизации или race conditions, которые проявляются только при hands-on тестировании. Phase 2 требует привлечения независимого подрядчика для этичного хакинга инфраструктуры и приложения раз в квартал.

### 12.2 Задачи

1. Выбрать и заключить договор с независимой компанией по пентестингу (white-box или gray-box).
2. Определить scope: внешний perimeter, внутренняя сеть, веб-приложение, API, мобильные endpoints, социальная инженерия (опционально).
3. Проведение пентеста раз в квартал с детальным отчётом: executive summary, technical findings, risk rating (CVSS), remediation recommendations.
4. Remediation: исправление выявленных critical и high уязвимостей в течение SLA (critical 1–3 рабочих дня, high 3–7 рабочих дней).
5. Интеграция результатов в CI/CD: блокировка деплоя при наличии новых critical/high уязвимостей, найденных в пентесте.
6. Хранение отчётов и remediation history в репозитории (без чувствительных данных) для audit trail.

### 12.3 Acceptance Criteria

- Пентест проводится раз в квартал независимым подрядчиком.
- Отчёт содержит executive summary, technical findings, risk rating (CVSS), remediation recommendations.
- Critical и high уязвимости исправлены в течение SLA (critical 1–3 рабочих дня, high 3–7 рабочих дней).
- Результаты прошлых пентестов хранятся в репозитории (без чувствительных данных) для audit trail.
- Pen test findings интегрированы в CI/CD Security Gate: новые critical/high блокируют мерж.

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
  - Для EU/RF-пользователей: предложить opt-out через email-верификацию или self-hosted hCaptcha с минимальной telemetry.
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

## 17. Ежедневная адаптивная модификация плана

### 17.1 Контекст

Текущий сервер (1 vCPU / 8 ГБ RAM) не позволяет запускать ежедневное ML-переобучение без влияния на отзывчивость приложения. Phase 1 покрывает базовую генерацию плана через `POST /ml/generate-plan` и классификацию состояния через `POST /ml/classify`. Ежедневная автоматическая модификация плана — это задача Phase 2, требующая отдельного планировщика/воркера и более мощной инфраструктуры.

### 17.2 Задачи

1. Развёртывание отдельного scheduled job (Kubernetes CronJob) для ежедневного перерасчёта планов на основе новых биометрических данных.
2. Интеграция с `.dvc` (Data Version Control) для версионирования ML-моделей и данных.
3. Настройка incremental training (дообучение) вместо полного переобучения для снижения нагрузки.
4. Очередь задач (`ml.generate` в RabbitMQ) для асинхронной генерации планов.
5. Мониторинг ресурсов воркера: алерт при CPU > 70% или RAM > 75%.

### 17.3 Acceptance Criteria

- Планы пользователей автоматически пересматриваются раз в 24 часа на основе последних биометрических данных.
- Переобучение не влияет на p95 латентность API (< 2s).
- DVC-tracked модели версионируются и откатываются при деградации качества.
- План переобучения завершается за < 10 минут на выделенном воркере (2+ vCPU, 4+ ГБ RAM).

### 17.4 Infrastructure Requirements

- **Минимум**: 2 vCPU, 4 ГБ RAM выделенный воркер для ML retrain
- **Рекомендуется**: 4 vCPU, 8 ГБ RAM с GPU (CUDA) для ускорения GAN inference
- **Хранилище**: 20 ГБ SSD для DVC cache + моделей

### 17.5 Trade-offs

- На текущем 1-vCPU сервере **не рекомендуется** включать ежедневное переобучение.
- Вместо этого: on-demand регенерация плана при явном запросе пользователя или при значительном изменении биометрических данных (> 20% отклонение от baseline).
- DVC уже инициализирован в проекте (`datasets/`), но pipeline retrain требует отдельного воркера.

---

---

## 18. Диаграмма зависимостей

### 18.1 Контекст

Phase 2 состоит из нескольких крупных блоков, которые нельзя выполнять хаотично. Ниже — обязательный порядок зависимостей.

### 18.2 Зависимости

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

### 18.3 Блокеры

|Блок|Блокирует|Причина|
|---|---|---|
|Infra provisioning|Vault, PostgreSQL HA, Service Mesh, Backup DR|Требует более мощного VPS и стабильного k8s|
|Vault + Secrets|Secrets Rotation Automation, Service Mesh (cert-manager)|Требует центрального хранилища секретов|
|PostgreSQL HA|Backup DR, Data Processor (production)|Требует стабильного Primary/Replica|
|Service Mesh|Observability (tracing), Canary Deployments|Требует sidecar-инъекции и PeerAuthentication|
|Security email + PGP|Bug Bounty, Pen Test, SECURITY.md обновление|Требует корпоративного ящика до публикации|
|Compliance 152-ФЗ|Medical services integration, Medical app registration|Требует шифрования и audit trail|

### 18.4 Параллельные работы

- **Vault + Secrets** можно начинать параллельно с **Infra provisioning** (на тестовом стенде)
- **CAPTCHA** не зависит от инфраструктуры, можно делать параллельно с **Compliance**
- **Bug Bounty program setup** не зависит от инфраструктуры, можно делать параллельно
- **Ежеквартальный внешний пентест** можно начинать после **Security email + PGP**, не зависит от инфраструктуры
- **Observability расширение** можно начинать после **Service Mesh**, но до полной миграции

---

## 19. Оценка стоимости (руб/мес)

### 19.1 Контекст

Все оценки указаны для российского хостинга (Yandex Cloud / Selectel / Timeweb VPS) и approximated. Точные цифры зависят от провайдера и региона.

### 19.2 Инфраструктура

|Компонент|Текущая стоимость|Новая стоимость|Δ|
|---|---|---|---|
|VPS (1 vCPU / 2 ГБ / 30 ГБ)|~1 500 ₽/мес|~3 500 ₽/мес (4 vCPU / 8 ГБ / 80 ГБ SSD)|+2 000 ₽/мес|
|Managed PostgreSQL (Yandex Managed)|—|~2 500–4 000 ₽/мес|+2 500–4 000 ₽/мес|
|Vault (self-hosted на отдельном VPS)|—|~1 500 ₽/мес (2 vCPU / 4 ГБ)|+1 500 ₽/мес|
|Backup storage (S3-compatible, 100 ГБ)|—|~300 ₽/мес|+300 ₽/мес|
|Domain fitpulse.app (первый год)|—|~1 500 ₽/год|+125 ₽/мес|
|SSL-сертификат (Let's Encrypt)|—|0 ₽/мес|0 ₽/мес|
|**Итого инфраструктура**|**~1 500 ₽/мес**|**~8 000–10 500 ₽/мес**|**+6 500–9 000 ₽/мес**|

### 19.3 ML-сервисы

|Компонент|Стоимость|Примечание|
|---|---|---|
|MLflow (self-hosted)|0 ₽/мес|Запускается на существующем VPS|
|DVC remote storage|0 ₽/мес|Локальный диск / S3-compatible |
|GPU-воркер (если нужен inference acceleration)|~5 000–15 000 ₽/мес|Yandex Cloud GPU / Lambda Labs|
|**Итого ML**|**0–15 000 ₽/мес**|Зависит от необходимости GPU|

### 19.4 Security / Compliance

|Компонент|Стоимость|Примечание|
|---|---|---|
|Corp email (Yandex 360 / Google Workspace)|~300–600 ₽/мес за пользователя|1–2 пользователя|
|PGP ключ / WKD|0 ₽/мес|Self-hosted|
|Bug Bounty вознаграждения (опционально)|0–10 000 ₽/мес|Зависит от бюджета|
|**Итого Security**|**300–10 600 ₽/мес**||

### 19.5 Итого Phase 2

|Сценарий|Стоимость/мес|Годовая стоимость|
|---|---|---|
|Минимум (без GPU, без bug bounty)|~8 500 ₽/мес|~102 000 ₽/год|
|Рекомендуемый (с observability, без GPU)|~12 000 ₽/мес|~144 000 ₽/год|
|Максимальный (с GPU, bug bounty)|~25 000–35 000 ₽/мес|~300 000–420 000 ₽/год|

**Trade-off**: На текущем 1-vCPU сервере невозможно запустить Vault + Istio + PostgreSQL HA одновременно. Требуется апгрейд VPS до минимум 4 vCPU / 16 ГБ RAM или разделение на 2 VPS.

---

## 20. Resource Plan: FTE

### 20.1 Контекст

Phase 2 требует специализации, которой нет у единственного разработчика. Ниже — оценка человеко-часов и необходимых ролей.

### 20.2 Роли и ответственность

|Роль|Занятость|Ответственность|
|---|---|---|
|**DevOps/Platform**|0.8 FTE|VPS provisioning, k8s, Vault, PostgreSQL HA, Service Mesh, CI/CD|
|**Backend (Go)**|0.6 FTE|Secrets integration, mTLS migration, admin panel, compliance endpoints|
|**ML/Data Engineer**|0.4 FTE|DVC pipeline, adaptive retrain, model versioning|
|**Frontend**|0.3 FTE|Achievements, Diet, Devices views из UI_SPECIFICATION|
|**Legal/Compliance**|0.2 FTE|152-ФЗ, медицинская регистрация, политики|
|**Security**|0.2 FTE|Bug bounty, PGP, WAF rules, penetration testing|
|**Product/Design**|0.1 FTE|Приоритизация фич, UI/UX approval|

### 20.3 Общие затраты

|Сценарий|FTE|Срок|Человеко-часы|
|---|---|---|---|
|Агрессивный (все параллельно)|1.6 FTE|8 недель|~2 560 ч|
|Рекомендуемый (последовательный)|0.8 FTE|16 недель|~2 560 ч|
|Консервативный (1 человек, 0.5 FTE)|0.5 FTE|32 недели|~2 560 ч|

**Важно**: В текущем состоянии проект поддерживается 1 человеком (`@MAMUER`). Phase 2 **невозможна** без привлечения хотя бы одного дополнительного DevOps/Backend разработчика.

---

## 21. Migration Strategy

### 21.1 Контекст

Каждый major change в Phase 2 требует стратегии миграции без downtime. Ниже — per-component планы.

### 21.2 Vault Migration (Kubernetes Secrets → Vault)

**Подход**: Gradual migration с dual-read периодом.

1. **Week 1**: Развёртывание Vault в dev-окружении, настройка Kubernetes auth method
2. **Week 2**: Миграция 1–2 не критичных секретов (например, `SMTP_*`) на Vault, приложения читают из Vault, но fallback на K8s Secret
3. **Week 3**: Миграция всех секретов, dual-read: приложение читает из Vault, при недоступности — из K8s Secret
4. **Week 4**: Отключение K8s Secrets, все приложения читают только из Vault
5. **Rollback**: При проблемах с Vault — переключение обратно на K8s Secrets через environment variable `VAULT_ENABLED=false`

### 21.3 PostgreSQL HA Migration (Single → Patroni)

**Подход**: Rolling migration с использованием pg_basebackup.

1. **Week 1**: Развёртывание Patroni + etcd на отдельном VPS/поде
2. **Week 2**: Настройка streaming replication с текущего primary на новый Patroni cluster
3. **Week 3**: Переключение application connection string на Patroni VIP, тестирование failover
4. **Week 4**: Деcommission старого single PostgreSQL
5. **Rollback**: При проблемах — переключение connection string обратно на старый primary

### 21.4 Service Mesh Migration (hand-rolled mTLS → Istio/Linkerd)

**Подход**: Canary migration namespace-by-namespace.

1. **Week 1**: Установка Istio control plane в dedicated namespace, без sidecar-инъекции
2. **Week 2**: Включение sidecar-инъекции для 1 не критичного namespace (например, `ml-generator`)
3. **Week 3**: Постепенное включение для остальных namespaces, мониторинг latency/errors
4. **Week 4**: Отключение hand-rolled mTLS, полный переход на mesh
5. **Rollback**: При проблемах — отключение sidecar-инъекции, возврат к hand-rolled mTLS

### 21.5 Data Processor Migration (stub → production)

**Подход**: Blue-green deployment consumer'а.

1. **Week 1**: Развёртывание data-processor в production с `PREFETCH=1`, без обработки сообщений (consumer-only, Nack all)
2. **Week 2**: Включение обработки для 10% сообщений (sampling), мониторинг dead-letter queue
3. **Week 3**: Полный rollout, мониторинг lag и error rate
4. **Week 4**: Отключение legacy-публения в `biometric_events` из biometric-service (если publisher migrated)
5. **Rollback**: Отключение data-processor подов, сообщения накапливаются в RabbitMQ

---

## 22. Risk Register

### 22.1 Контекст

Каждый пункт Phase 2 имеет operational риски. Ниже — реестр рисков с fallback-стратегиями.

### 22.2 Риски

|ID|Риск|Вероятность|Влияние|Митигация|Fallback|
|---|---|---|---|---|---|
|R1|Vault не справляется с нагрузкой при 100+ RPS|Средняя|Высокое|Load testing перед production; Vault cluster из 3 нод|Остаться на Kubernetes Secrets + внешний vault-агент|
|R2|PostgreSQL HA failover работает некорректно|Средняя|Высокое|Ежеквартальные chaos tests; pg_basebackup проверка|Остаться на single PostgreSQL с ежедневными бэкапами|
|R3|Istio потребляет > 1 ГБ RAM на control plane|Высокая|Среднее|Использовать Linkerd вместо Istio (легче)|Остаться на hand-rolled mTLS|
|R4|ML retrain падает по памяти на 2 vCPU|Высокая|Среднее|Ограничить resources.limits; использовать swap|Перенести retrain на GitHub Actions / external GPU|
|R5|152-ФЗ compliance не достигнут|Средняя|Высокое|Юридическая экспертиза на этапе проектирования|Ограничить функционал для РФ-пользователей|
|R6|Корпоративный email не получен (бюджет)|Средняя|Среднее|Использовать бесплатный Yandex 360 для бизнеса|Остаться на личном email с PGP|
|R7|Bug bounty программа привлекает неточные репорты|Высокая|Низкое|Чёткий scope, triage-процесс|Игнорировать некорректные репорты|
|R8|DVC remote не доступен из k8s|Средняя|Среднее|Настроить S3-compatible storage (MinIO)|Локальный DVC cache без remote|
|R9|Service Mesh конфликтует с существующими Network Policies|Средняя|Высокое|Тестирование в dev перед production|Откат на hand-rolled mTLS|
|R10|Adaptive daily retrain перегружает API|Высокая|Высокое|Очередь RabbitMQ + rate limiting на retrain job|On-demand retrain только по запросу пользователя|

### 22.3 Risk Response Plan

|Риск|Ответ|Trigger|Action|
|---|---|---|---|
|R1|Mitigate|Vault latency > 500ms|Масштабировать Vault cluster|
|R2|Mitigate|Failover > 30s|Откат на single PostgreSQL|
|R3|Avoid|Istio memory > 1.5 ГБ|Использовать Linkerd|
|R4|Transfer|ML retrain OOM|Перенести на external CI|
|R5|Accept|Legal costs > 500k ₽|Ограничить функционал|
|R6|Mitigate|Бюджет 0 ₽|Использовать бесплатный email|
|R7|Accept|Трафик < 10 reports/мес|Низкие затраты на triage|
|R8|Mitigate|DVC unavailable|MinIO fallback|
|R9|Mitigate|Mesh errors > 1%|Откат на hand-rolled mTLS|
|R10|Avoid|API latency > 2s|On-demand только|

---

## 23. Exit Criteria для Phase 2

### 23.1 Контекст

Phase 2 завершается, когда выполнены все Must-have критерии. Should-have и Could-have могут быть перенесены на Phase 3.

### 23.2 Must-have (Phase 2 exit criteria)

|Критерий|Метрика|Приоритет|
|---|---|---|
|Vault развёрнут и все секреты мигрированы|0 секретов в Kubernetes Secrets|P0|
|PostgreSQL HA с failover < 30s|RTO < 30s, RPO = 0|P0|
|Service Mesh активен между всеми сервисами|mTLS 100%, zero manual cert rotation|P0|
|152-ФЗ compliance документация готова|Политика утверждена, audit log 3 года|P0|
|Security email заменён на корпоративный|Личный email удалён из SECURITY.md|P0|
|CI/CD pipeline обновлён (govulncheck, Gitleaks, TruffleHog)|Все scans проходят, Security Gate PASS|P1|
|Backup DR протестирован|Recovery drill раз в квартал, RTO < 1ч|P1|
|Observability: Grafana + Alertmanager|Дашборды покрывают все critical endpoints|P1|

### 23.3 Should-have (Phase 2 exit criteria — желательно)

|Критерий|Меторика|Приоритет|
|---|---|---|
|CAPTCHA интегрирован|Error rate 429 ↓ на 50%|P2|
|Bug bounty программа запущена|≥ 5 reports/мес|P2|
|Ежеквартальный внешний пентест проведён|Отчёт с remediation plan|P2|
|Medical services integration API готов|Sync с Flo/OKOK работает|P2|
|Adaptive daily plan retrain (on-demand)|Retrain завершается < 10 мин|P3|

### 23.4 Could-have / Won't-have (переносится на Phase 3)

|Критерий|Причина переноса|
|---|---|
|Canary Deployments (Flagger)|Требует Istio + extensive testing, низкий приоритет для 1-сервисной архитектуры|
|Medical app registration в Минздраве|Юридический процесс 3-4 месяца, не зависит от технической реализации|
|GPU-ускорение для ML|Дорого, текущий объём данных не требует GPU|
|Full disaster recovery (warm standby на another VPS)|Требует второго VPS, дорого для учебного проекта|

---

## 24. Timeline с Milestones

### 24.1 Контекст

Phase 2 планируется на **3–4 месяца** (12–16 недель) при нагрузке 0.5–0.8 FTE. Ниже — детальный timeline с вехами.

### 24.2 Milestones

|Milestone|Срок|Deliverables|Exit criteria|
|---|---|---|---|
|**M1: Foundation**|Недели 1–2|VPS upgrade, Vault deployed, corporate email|Vault отвечает < 10ms, email работает| 
|**M2: Security Hardening**|Недели 3–4|Secrets rotation automation, mTLS migration started, PGP key published|0 секретов в K8s Secrets, PGP fingerprint в SECURITY.md|
|**M3: Database HA**|Недели 5–6|PostgreSQL HA deployed, failover tested, backup DR|RTO < 30s, recovery drill пройден|
|**M4: Observability**|Недели 7–8|Service mesh deployed, Grafana dashboards, Alertmanager|Дашборды покрывают 100% critical endpoints, алерты работают|
|**M5: Compliance**|Недели 9–10|152-ФЗ documentation, medical API, security email migrated, pen test vendor contracted|Политика утверждена, medical sync работает, pen test запланирован|
|**M6: Polish**|Недели 11–12|CAPTCHA, bug bounty launch, adaptive retrain on-demand|Все Must-have критерии выполнены|

### 24.3 Gantt Chart (text-based)

```
Неделя:    1    2    3    4    5    6    7    8    9    10   11   12
VPS:       [████████████████████████████████████████████████████████████]
Vault:          [████████████████████████████████████████████████████████]
Secrets:              [████████████████████████████████████████████████████]
PostgreSQL HA:               [████████████████████████████████████████████]
Service Mesh:                     [████████████████████████████████████████]
Observability:                           [████████████████████████████████████]
Compliance:                                      [████████████████████████████]
CAPTCHA:                                                  [████████████████████]
Bug Bounty:                                                      [████████████████]
Pen Test:                                                              [████████████████]
Medical:                                                                      [████████████████]
Adaptive Retrain:                                                                 [████████████████]
```

### 24.4 Дependencies Timeline

```text
M1 (Foundation)
  ├── VPS upgrade ──────┐
  ├── Vault ────────────┤
  └── Corporate email ──┘
          │
          ▼
M2 (Security Hardening)
  ├── Secrets rotation ────────► M3
  ├── mTLS migration ──────────┤
  └── PGP key ────────────────┘
              │
              ▼
M3 (Database HA) ───────► M4
              │
              ▼
M4 (Observability) ───────► M5
              │
              ▼
M5 (Compliance) ───────► M6
              │
              ▼
M6 (Polish)
  ├── CAPTCHA
  ├── Bug Bounty
  ├── Pen Test
  └── Adaptive Retrain
```

### 24.5 Critical Path

```
VPS upgrade → Vault → PostgreSQL HA → Service Mesh → Observability → Compliance → Medical
```

**Любая задержка в Critical Path задержит всю Phase 2 на 1–2 недели.**

### 24.6 Buffer

- **10% buffer** на непредвиденные проблемы (итого 13–14 недель вместо 12)
- **Еженедельный sync** для корректировки timeline
- **Go/No-go checkpoint** на Milestone 3: если PostgreSQL HA не справляется — откат на managed CloudSQL

---

## 25. Критерии приёмки Phase 2

### 25.1 Контекст

Phase 2 считается завершённой, когда выполнены все Must-have критерии из раздела 22.2.

### 25.2 Checklist

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

### 25.3 Go/No-go Criteria

|Критерий|Go|No-go|
|---|---|---|
|Vault latency|P99 < 50ms|P99 > 200ms → откат|
|PostgreSQL failover|RTO < 30s|RTO > 60s → откат на single|
|Service Mesh overhead|Memory < 500MB/pod|Memory > 1GB/pod → Linkerd вместо Istio|
|Budget|≤ 12 000 ₽/мес|≥ 20 000 ₽/мес → сокращение scope|

---

## 26. Phase 3 Preview

### 26.1 Что точно не входит в Phase 2

- Canary Deployments (Flagger + Argo Rollouts)
- Full medical app registration в Минздраве
- GPU-ускорение для ML inference
- Multi-region DR (требует второго датацентра)
- Advanced ML: reinforcement learning для адаптации планов

### 26.2 Что потенциально перейдёт в Phase 3

- Service Mesh → полноценный Istio с traffic shaping
- Vault → HSM-backed key management
- Observability → OpenTelemetry Collector + Thanos
- ML → online learning сFeedback loop

### 26.3 Предварительный объём Phase 3

|Этап|Срок|Ответственный|
|---|---|---|
|Canary Deployments|2–3 недели|DevOps/Backend|
|Medical registration|3–4 недели|Legal|
|Advanced ML (RL)|3–4 недели|ML Engineer|
|Multi-region DR|4–6 недель|DevOps|
|Full Service Mesh (Istio)|2–3 недели|Platform|

**Итого Phase 3: 3–4 месяца**

---

## 27. Расширение поддерживаемых устройств

### 27.1 Контекст

Текущие интегрированные устройства: Fitbit (OAuth 2.0) и Garmin (OAuth 1.0a). В UI присутствуют карточки для Fitbit, Garmin и Withings. Samsung Galaxy Watch и Huawei Watch D2 поддерживаются только в roadmap Phase 2.

### 27.2 План

|Этап|Устройство|Срок|Приоритет|Задачи|
|---|---|---|---|---|
|1|Withings|1 неделя|P0|OAuth 2.0, пул пульса/SpO₂/давления/температуры/сна/шагов|
|2|Samsung Galaxy Watch|3–4 недели|P2|Samsung Health Connect API: требует регистрации в Samsung Developer Program, создания приложения в Samsung Galaxy Store, обязательной установки Samsung Health на телефон пользователя, OAuth 2.0 через Samsung account. Данные передаются с телефона, а не напрямую с часов.|
|3|Huawei Watch D2|3–4 недели|P2|Huawei Health Kit: требует регистрации в Huawei Developer Alliance, официального приложения в AppGallery, OAuth 2.0 через Huawei ID. Данные идут через телефон. Аналогично Samsung — средняя сложность.|

### 27.3 Acceptance Criteria

- Каждое устройство имеет working OAuth flow (auth → callback → token storage)
- Минимум 3 метрики (heart_rate, spo2, sleep) синхронизируются автоматически
- Данные поступают через общий `POST /devices/{id}/ingest` в device-connector
- UI отображает статус подключения и последнюю синхронизацию

### 27.4 Архитектурные ограничения

- Device-connector остаётся универсальным: валидирует записи, дедуплицирует, переправляет в biometric-service
- Device-aggregator получает нового провайдера на каждое устройство
- OAuth-токены хранятся в `device_provider_accounts` с шифрованием
- Для устройств без публичного OAuth API (Samsung Galaxy Watch, Huawei Watch D2) требуется регистрация в соответствующих developer programs и создание мобильного приложения для передачи данных с телефона на backend

