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

### Наблюдаемость

- [x] Все сервисы логируют в структурированном JSON (timestamp, level, service, correlationId, userId, action)
- [x] Реализованы 6 обязательных Prometheus-метрик

### Безопасность

- [x] Network Policies разделяют зоны dmz/app/data/monitoring
- [x] RBAC: минимальные права, отдельные ServiceAccount
- [x] Шифрование: pgsodium (libsodium) для PII, AES-256-GCM для TOTP, LUKS volumes, secrets
- [x] mTLS для внутренних gRPC-коммуникаций (TLS 1.3, сертификаты в Kubernetes Secret)
- [x] WAF настроен с базовым набором правил (Ingress NGINX + ModSecurity CRS v4; cert-manager для TLS; automated CRS updates через CronJob)

### Релизный процесс

- [x] Пайплайн включает все этапы
- [x] Cosign подпись образов, SBOM (syft) → OCI artifact рядом с образом, проверка через cosign verify

### Приемка

- [ ] Availability: 99.9%
- [ ] Latency p95: < 2s (SLO), canary gate < 3s, rollback при p95 > 5s
- [ ] MTTR: < 5 мин
- [ ] RTO: < 1 час. RPO = 0 только при multi-AZ; single-VPS: RPO < 1 мин (WAL shipping)

### Документация

- [x] ADR для всех архитектурных решений
- [x] Runbook для эксплуатации и отката
- [x] Protobuf/gRPC спецификация актуальна и покрыта тестами
