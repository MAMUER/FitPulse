# FitPulse — Критерии приёмки

## Definition of Done

- [x] Все unit-тесты проходят (`go test ./...`)
- [x] Линтер без ошибок (`golangci-lint run` → 0 issues)
- [x] Security-scan без критических уязвимостей (gosec, govulncheck, Trivy)
- [x] Приложение разворачивается через `kubectl apply -k configs/k8s/base/`
- [x] Регистрация → верификация email → логин → получение профиля работают последовательно
- [x] Документация обновлена (README, API.md, ARCHITECTURE.md)

## Критерии приемки архитектуры

### Инфраструктура

- [x] Матрица окружений применена ко всем компонентам
- [x] PostgreSQL 18 с pgsodium for at-rest columns, key management (envelope encryption, не двойное шифрование)
- [x] RabbitMQ 4 с persistent queues и DLQ
- [x] Valkey 9
- [x] MinIO для хранения бэкапов (S3-compatible, в рамках free-stack)
- [x] Vault + External Secrets Operator для управления секретами (PostgreSQL backend)

### Наблюдаемость

- [x] Все сервисы логируют в структурированном JSON (timestamp, level, service, correlationId, userId, action)
- [x] Реализованы 6 обязательных Prometheus-метрик
- [x] Prometheus + Grafana + Alertmanager развёрнуты
- [x] Jaeger + OpenTelemetry для distributed tracing
- [ ] Loki/Fluent Bit для централизованного логирования (Phase 2)

### Безопасность

- [x] Network Policies разделяют зоны dmz/app/data/monitoring
- [x] RBAC: минимальные права, отдельные ServiceAccount
- [x] Шифрование: pgsodium (libsodium) для PII, AES-256-GCM для TOTP, LUKS volumes, secrets
- [x] mTLS для внутренних gRPC-коммуникаций (TLS 1.3, hand-rolled certs)
- [x] WAF настроен с базовым набором правил (Ingress NGINX + ModSecurity CRS v4; cert-manager для TLS; automated CRS updates через CronJob)
- [x] Argon2id хеширование паролей (memory 64 MB, iterations 3, parallelism 1)
- [x] TOTP 2FA с backup codes
- [x] Refresh token rotation + reuse detection
- [x] Rate limiting (per-IP 10r/s, burst 50; per-user 100r/s, burst 200)
- [x] CAPTCHA (Cloudflare Turnstile) при превышении rate limit
- [x] Security reporting: corporate email + PGP (Phase 2)

### Релизный процесс

- [x] Пайплайн включает все этапы
- [x] Cosign подпись образов, SBOM (syft) → OCI artifact рядом с образом, проверка через cosign verify
- [x] gosec, govulncheck, Trivy, TruffleHog, Gitleaks в CI/CD

### Compliance

- [x] 152-ФЗ: шифрование at rest (pgsodium) и in transit (TLS 1.3)
- [x] GDPR: explicit consent для специальных категорий ПДн (здоровье, менструальный цикл)
- [x] Cookie consent banner (GDPR Art. 7)
- [x] Medical disclaimer в UI при вводе health-данных
- [x] DPA с Google OAuth (трансграничная передача ПДн в США)
- [x] DPIA (Data Protection Impact Assessment) — `docs/compliance/DPIA.md`
- [x] Реестр обработки ПДн — `docs/compliance/РЕЕСТР_ОБРАБОТКИ_ПДН.md`
- [ ] Centralized audit trail в Elasticsearch (Phase 2; текущий audit-logger пишет в stdout)
- [ ] Quarterly access review (runbook готов, требуется автоматизация)

### Приемка

- [ ] Availability: 99.9% (gateway, biometric-service, training-service)
- [ ] Latency p95: < 2s (SLO), canary gate < 3s, rollback при p95 > 5s
- [ ] MTTR: < 5 мин
- [ ] RTO: < 30 секунд (PostgreSQL HA, Phase 2). Single-VPS: RTO < 1 час.
- [ ] RPO: = 0 при multi-AZ (Phase 2). Single-VPS: RPO < 1 мин (WAL shipping).
- [ ] Security events: MTTD < 5 мин, MTTR SEV-1 < 1 час
- [ ] Audit log delivery lag: < 5 сек (Phase 2, centralized ELK)

### ML / AI

- [x] Classifier (rule-based) — Phase 1 production
- [x] ML Generator (Conditional Diffusion Model, DDPM) — Phase 1 production
- [ ] Ежедневное переобучение (Phase 2)

### Устройства

- [x] Open Wearables webhook (`/api/v1/integrations/open-wearables/webhook`)
- [x] Device Aggregator — production
- [ ] Samsung Galaxy Watch, Huawei Watch D2 (Phase 2)

### Admin

- [x] Admin CLI для управления пользователями
- [x] Invite-коды для регистрации
- [x] Просмотр пользователей и устройств

### UI / Accessibility

- [x] WCAG 2.1 AA compliance
- [x] High contrast mode
- [x] Reduced motion support
- [x] Cookie consent banner
- [x] Medical disclaimer на экранах с health-данными
- [x] Skip navigation link
- [x] Touch targets ≥ 44×44px
- [x] Color-blind considerations (не только цвет для статусов)
- [x] ARIA live regions для динамических обновлений

### Документация

- [x] ADR для всех архитектурных решений
- [x] Runbook для эксплуатации и отката
- [x] Protobuf/gRPC спецификация актуальна и покрыта тестами
- [x] API reference (REST + gRPC)
- [x] SLO/SLI definition
- [x] Recovery drill template
- [x] Incident response playbook
- [x] Quarterly access review runbook
- [x] DPIA, Privacy Policy, DPA, Registry
