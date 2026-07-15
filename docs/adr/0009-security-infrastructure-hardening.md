# ADR 0009: Усиление безопасности и инфраструктуры

## Контекст

Согласно требованиям комплексного аудита, система требует hardening по безопасности, инфраструктуре, наблюдаемости, CI/CD и ML-пайплайну для соответствия production-ready стандартам.

## Решение

Реализовать следующие меры усиления:

1. **Зоны безопасности**: Полные Network Policies для dmz, app-zone, data-zone, monitoring-zone.
2. **RBAC**: Принцип минимальных привилегий с выделенными ServiceAccounts и минимальными ролями.
3. **Шифрование**: pgsodium для PostgreSQL (детерминированное AEAD для PII, envelope encryption для токенов).
4. **Бэкапы**: ежедневный `pg_dump` (custom format) с 7-дневным retention. WAL-based инкрементальные бэкапы с поддержкой PITR запланированы на Phase 2.
5. **Метрики**: дополнительные Prometheus-метрики для трейкинга ошибок, ML-уверенности, пулов БД, очередей, sync lag.
6. **CI/CD**: 9-этапный пайплайн с security gates (SAST, dependency scans, container scanning, smoke-тесты). Canary-деплой и автоматические триггеры отката запланированы на Phase 2.
7. **OpenAPI**: обновление до 3.0.3 с исчерпывающей API-документацией.

## Последствия

- усиленная security posture с zero-trust networking;
- улучшенная наблюдаемость с детальными метриками и алертингом;
- надёжный deployment pipeline с ручным откатом (`kubectl rollout undo`);
- production-ready инфраструктурная конфигурация (частично; canary/PITR — Phase 2).

## Реализация

- обновлены Network Policies в `configs/k8s/base/security-zones.yaml`;
- усиленный RBAC в `configs/k8s/base/rbac/rbac.yaml`;
- ежедневный бэкап PostgreSQL через `pg_dump` (7-day retention) в CI/CD; WAL-архивация и PITR запланированы на Phase 2;
- расширены метрики в `internal/metrics/metrics.go` и `internal/metrics/extended.go`;
- расширен CI/CD пайплайн в `.github/workflows/ci.yml` (9 этапов, security gates);
- обновлён OpenAPI до 3.0.3 в `api/rest/swagger.yaml`.

## Рассмотренные альтернативы

- Коммерческие решения для бэкапов: более высокая стоимость, вендор-лок-ин.
- Ручные security reviews: медленнее, более error-prone.
- Базовый CI/CD: недостаточный для production надёжности.
