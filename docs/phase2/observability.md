# Observability

## 5. Наблюдаемость и SLO

### 5.1 Текущий статус Phase 1

Production-сервисы: gateway, user-service, biometric-service, training-service, classifier, ml-generator, data-processor, device-aggregator.
Production domain: fittpulse.duckdns.org

| Компонент | Статус Phase 1 | Что требуется в Phase 2 |
| --- | --- | --- |
| Prometheus | ✅ Развёрнут | Retention 15 дней (hot), добавить long-term storage |
| Alertmanager | ✅ Развёрнут | Retention 90 дней, добавить настройки для SEV-1/SEV-2 |
| Grafana | ✅ Развёрнут | RED/SLO дашборды для всех critical endpoints |
| Jaeger | ⚠️ Частично | Принимает трейсы от всех сервисов, требуется валидация |
| Loki / Централизованное логирование | ❌ Не развёрнено | ELK/Fluent Bit с retention 3 года для audit logs |
| Audit log | ⚠️ stdout (ConsoleAuditLogger) | Централизованный сбор в Elasticsearch |

### 5.2 Acceptance Criteria Phase 2

- Все critical endpoints покрыты RED/SLO дашбордами в Grafana
- Burn rate алерты срабатывают при превышении порога в течение 30 секунд
- Jaeger UI доступен и принимает трейсы от всех сервисов (trace ID коррелируется с логами)
- On-call бот доставляет алерты SEV-1/SEV-2 в Telegram в течение 1 минуты
- Метрики ошибочного бюджета SLO приводят к автоматическому применению политик
- Retention Prometheus: 15 дней (hot); retention Alertmanager: 90 дней; audit logs: 3 года в Elasticsearch

### 5.3 SLO Targets

| Service | Availability | P95 Latency | Error Budget |
| --- | --- | --- | --- |
| gateway | 99.9% | < 2s | < 0.1% |
| user-service | 99.9% | < 2s | < 0.1% |
| biometric-service | 99.9% | < 2s | < 0.1% |
| training-service | 99.9% | < 2s | < 0.1% |
| classifier | 99.5% | < 500ms | < 0.5% |
| ml-generator | 99.5% | < 1s | < 0.5% |
| data-processor | 99.9% | < 2s | < 0.1% |
| device-aggregator | 99.9% | < 2s | < 0.1% |
| security-events (audit log ingestion) | 99.9% | < 5s delivery lag | < 0.1% |
| vault (secrets availability) | 99.9% | < 200ms | < 0.1% |

### 5.4 Security & Reliability SLO

| Category | SLO | Target | Status |
| --- | --- | --- | --- |
| Security events | Время обнаружения инцидента (MTTD) | < 5 мин | ✅/❌ |
| Security events | Время реагирования (MTTR) для SEV-1 | < 1 час | ✅/❌ |
| Security events | Audit log delivery lag (event → ELK) | < 5 сек | ✅/❌ |
| ML inference | Classifier P99 latency | < 500ms | ✅/❌ |
| ML inference | ML generator P99 latency | < 1s | ✅/❌ |
| Vault | Secret read availability | 99.9% | ✅/❌ |
| Vault | Secret read P99 latency | < 200ms | ✅/❌ |
| Vault | Vault unseal time (after restart) | < 30 сек | ✅/❌ |

### 5.5 Зависимости

- **ELK/Fluent Bit** — требуется для централизованного аудит-логирования
- **Service Mesh** — требуется для distributed tracing (Jaeger)
- **Alertmanager** — уже работает, требуется донастройка для SEV-1/SEV-2

### 5.6 Ссылки

- [Runbook: Инциденты](../runbooks/INCIDENT_RESPONSE.md)
- [SLO Monthly Report Template](../compliance/SLO_MONTHLY_REPORT_TEMPLATE.md)
- [Recovery Drill Template](../compliance/ШАБЛОН_ОТЧЁТА_RECOVERY_DRILL.md)
