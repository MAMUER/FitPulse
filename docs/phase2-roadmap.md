# Phase 2 Backlog

> Детализационный бэклог инфраструктуры, масштабирования и hardening production-ready wellness-платформы.
> Phase 1 уже работает в production.

---

Детализированные разделы вынесены в подкаталог [docs/phase2/](phase2/):

- [Security](phase2/security.md) — Service Mesh, Security Reporting, CAPTCHA, Secrets Rotation
- [Infrastructure](phase2/infrastructure.md) — PostgreSQL HA, IaC, DR, Canary, Dependencies, Migration Strategy
- [Compliance](phase2/compliance.md) — Compliance 152-ФЗ, GDPR, medical boundary
- [Observability](phase2/observability.md) — Observability and SLO
- [Medical](phase2/medical.md) — Wellness boundary enforcement, medical registration (не требуется)
- [ML](phase2/ml.md) — Daily adaptive plan retrain, Cost estimate, Resource Plan FTE
- [Devices](phase2/devices.md) — Extended devices
- [Yandex ID](phase2/yandex-id.md) — Yandex ID production verification

---

## Goals & Objectives

1. **Надёжность:** RTO < 30s, RPO = 0 для критичных сервисов
2. **Безопасность:** Service Mesh mTLS, secrets rotation, security reporting
3. **Compliance:** Централизованный audit trail, retention 3 года, quarterly access review
4. **Наблюдаемость:** RED/SLO дашборды, Jaeger tracing, burn-rate алерты
5. **Масштабируемость:** Terraform + ArgoCD, canary deployments
6. **Medical boundary:** FitPulse остаётся wellness-платформой

## Success Criteria

| Критерий | Целевое значение |
| --- | --- |
| Availability | 99.9% для gateway, biometric-service, training-service |
| RTO | < 30 секунд при отказе primary PostgreSQL |
| RPO | = 0 при синхронных репликах в разных AZ |
| Audit log retention | 3 года в Elasticsearch |
| Secrets rotation | Автоматическая, без перезапуска сервисов |
| Time to detect SEV-1 | < 5 минут |
| Time to resolve SEV-1 | < 1 час |
| DPO response time | < 30 дней на запросы субъектов ПДн |
| Medical registration | Не требуется (wellness-платформа) |

## Timeline

| Квартал | Фокус |
| --- | --- |
| Q1 2027 | PostgreSQL HA, Vault hardening, Observability базовый уровень, quarterly access review |
| Q2 2027 | Service Mesh, Secrets Rotation, ELK/Fluent Bit, CAPTCHA production |
| Q3 2027 | Canary Deployments, GDPR/152-ФЗ compliance завершение, Medical boundary docs |
| Q4 2027 | Pen Test, Bug Bounty, Adaptive ML retrain, DPA/contracts финализация |

---

## Why Yandex ID instead of Google OAuth?

The project is a student project with practically no budget. Yandex ID was chosen because:

1. It's free for students and small projects
2. No credit card required
3. Simple integration
4. Targets Russian-speaking users

Other social authentications (VK, Mail.ru, Odnoklassniki, etc.) are planned for Phase 2 when the project has more resources.

