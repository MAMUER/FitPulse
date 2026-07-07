# FitPulse — Критерии приёмки

## Definition of Done

- [ ] Все unit-тесты проходят (`go test ./...`)
- [ ] Линтер без ошибок (`golangci-lint run` → 0 issues)
- [ ] Security-scan без критических уязвимостей (gosec, govulncheck, Trivy)
- [ ] Приложение разворачивается через `kubectl apply -k configs/k8s/base/`
- [ ] Регистрация → верификация email → логин → получение профиля работают последовательно
- [ ] Документация обновлена (README, API.md, ARCHITECTURE.md)

## Критерии приемки архитектуры

### Инфраструктура

- [ ] Матрица окружений применена ко всем компонентам
- [ ] PostgreSQL 18 с pgcrypto для at-rest columns, key management (envelope encryption, не двойное шифрование)
- [ ] RabbitMQ 4 с persistent queues и DLQ
- [ ] Valkey 9
- [ ] ELK Stack: 90 дней хранения, JSON-логи, RBAC в Kibana
- [ ] Prometheus: service discovery, recording rules, Alertmanager

### Наблюдаемость

- [ ] Все сервисы логируют в структурированном JSON (timestamp, level, correlationId, userId, action)
- [ ] Реализованы 6 обязательных Prometheus-метрик
- [ ] Настроены алерты с эскалацией по уровням SEV

### Безопасность

- [ ] Network Policies разделяют зоны dmz/app/data/monitoring
- [ ] RBAC: минимальные права, отдельные ServiceAccount
- [x] Шифрование: pgsodium (libsodium) для PII, AES-256-GCM для TOTP, LUKS volumes, secrets
- [x] mTLS для внутренних gRPC-коммуникаций (TLS 1.3, сертификаты в Kubernetes Secret)
- [x] WAF настроен с базовым набором правил (host Nginx + ModSecurity CRS v4; ingress-nginx `enable-modsecurity: "true"`)

### Релизный процесс

- [ ] Пайплайн включает все этапы
- [ ] Автоматический rollback при error rate > baseline + 1%
- [ ] Cosign подпись образов, SBOM (syft) → OCI artifact рядом с образом, проверка через cosign verify + admission webhook (Kyverno/OPA)

### Приемка

- [ ] Availability: 99.9%
- [ ] Latency p95: < 2s (SLO), canary gate < 3s, rollback при p95 > 5s
- [ ] MTTR: < 5 мин
- [ ] RTO: < 1 час. RPO = 0 только при multi-AZ; single-VPS: RPO < 1 мин (WAL shipping)
- [ ] Пентест запланирован и пройден

### Документация

- [ ] ADR для всех архитектурных решений
- [ ] Runbook для эксплуатации и отката
- [ ] OpenAPI-спецификация актуальна и покрыта тестами
