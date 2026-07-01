# FitPulse — Критерии приёмки

## Definition of Done

- [ ] Все unit-тесты проходят (`go test ./...`)
- [ ] Линтер без ошибок (`golangci-lint run` → 0 issues)
- [ ] Security-scan без критических уязвимостей (gosec, govulncheck, Trivy)
- [ ] Приложение разворачивается через `docker compose up -d`
- [ ] Регистрация → верификация email → логин → получение профиля работают последовательно
- [ ] Документация обновлена (README, API.md, ARCHITECTURE.md)

## Критерии приемки архитектуры

### Инфраструктура

- [ ] Матрица окружений применена ко всем компонентам
- [ ] PostgreSQL 18 с pgcrypto + LUKS (dm-crypt) + Vault Transit для sensitive fields
- [ ] RabbitMQ 4 с persistent queues и DLQ
- [ ] Redis 7 с persistence и кластеризацией
- [ ] ELK Stack: 90 дней хранения, JSON-логи, RBAC в Kibana
- [ ] Prometheus: service discovery, recording rules, Alertmanager

### Наблюдаемость

- [ ] Все сервисы логируют в структурированном JSON (timestamp, level, correlationId, userId, action)
- [ ] Реализованы 6 обязательных Prometheus-метрик
- [ ] Настроены алерты с эскалацией по уровням SEV

### Безопасность

- [ ] Network Policies разделяют зоны dmz/app/data/monitoring
- [ ] RBAC: минимальные права, отдельные ServiceAccount
- [ ] Шифрование: pgcrypto, KMS/volumes, Vault/secrets
- [ ] mTLS для внутренних gRPC-вызовов (SPIRE / cert-manager + linkerd)
- [ ] WAF настроен с базовым набором правил
- [ ] Secrets rotation: 90 дней, автоматизировано через Vault

### Релизный процесс

- [ ] Пайплайн включает все 9 этапов
- [ ] Canary-деплой с критериями успеха/отката
- [ ] Автоматический rollback при error rate > baseline + 5%
- [ ] Cosign подпись образов, SBOM generation (syft)

### Приемка

- [ ] Availability: 99.9%
- [ ] Latency p95: < 5s
- [ ] MTTR: < 5 мин
- [ ] RTO: < 1 час, RPO = 0 (синхронные реплики)
- [ ] Пентест запланирован и пройден
- [ ] Реализованы механизмы соответствия 152-ФЗ

### Документация

- [ ] ADR для всех архитектурных решений
- [ ] Runbook для эксплуатации и отката
- [ ] OpenAPI-спецификация актуальна и покрыта тестами
