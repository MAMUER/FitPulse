# Phase 2 Backlog

> Phase 2 — инфраструктура, масштабирование и hardening production-ready wellness-платформы.
> Phase 1 уже работает в production: gateway, user-service, biometric-service, training-service, classifier, ml-generator, data-processor, device-aggregator, Vault (External Secrets), PostgreSQL, Valkey, RabbitMQ, WAF, daily backups, Yandex ID, TOTP 2FA.

---

## Goals & Objectives

1. **Надёжность:** RTO < 30s, RPO = 0 для критичных сервисов за счёт PostgreSQL HA и DR.
2. **Безопасность:** Service Mesh mTLS, secrets rotation, security reporting, CAPTCHA, bug bounty.
3. **Compliance:** Централизованный audit trail, retention 3 года, quarterly access review, cookie consent, DPA.
4. **Наблюдаемость:** RED/SLO дашборды, Jaeger tracing, burn-rate алерты, централизованное логирование.
5. **Масштабируемость:** Terraform + ArgoCD, canary deployments, GPU-опциональный ML inference.
6. **Medical boundary:** FitPulse остаётся wellness-платформой, не становится медицинским ПО.

## Success Criteria (Phase 2 в целом)

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

## Phase 1 Reality Check

| Компонент | Статус Phase 1 | Примечание |
| --- | --- | --- |
| Gateway (Go) | ✅ Production | gRPC clients, handlers auth/health/training |
| User Service | ✅ Production | PostgreSQL, pgsodium, RBAC |
| Biometric Service | ✅ Production | body composition, conditions, menstrual cycles |
| Training Service | ✅ Production | plans, achievements, classifier integration |
| Classifier | ✅ Production | Rule-based illness/overtraining/stress detection |
| ML Generator | ✅ Production | DDPM inference, 3-tier fallback |
| Data Processor | ✅ Production | RabbitMQ consumer, DLQ |
| Device Aggregator | ✅ Production | Open Wearables webhook forwarder |
| Vault + External Secrets | ✅ Production | SecretStore → Vault, automatic injection |
| PostgreSQL | ✅ Production | Single instance, daily backups, pgsodium |
| Valkey | ✅ Production | Sessions, rate limiting, TOTP cache |
| RabbitMQ | ✅ Production | DLQ, retry logic |
| WAF | ✅ Production | ModSecurity + OWASP CRS v4 |
| TLS | ✅ Production | cert-manager + Let's Encrypt |
| Yandex ID | ✅ Production | Consent screen testing, DPA готов |
| TOTP 2FA | ✅ Production | Argon2id, pgsodium encryption |
| Audit Logging | ⚠️ Partial | ConsoleAuditLogger (stdout), ELK не развёрнут |
| Cookie Consent | ✅ UI ready | localStorage, banner в App.jsx |
| Medical disclaimers | ✅ UI ready | Register.jsx, Body.jsx, Legal.jsx |

---

## Table of Contents

- [Security](security.md)
  - [2. Service Mesh](security.md#2-service-mesh)
  - [9. Security Reporting: Corporate Email + PGP](security.md#9-security-reporting-corporate-email--pgp)
  - [10. CAPTCHA (Cloudflare Turnstile)](security.md#10-captcha-cloudflare-turnstile)
  - [11. Secrets Rotation Automation](security.md#11-secrets-rotation-automation)
- [Infrastructure](infrastructure.md)
  - [Testing Strategy](infrastructure.md#testing-strategy)
  - [3. PostgreSQL High Availability & Disaster Recovery](infrastructure.md#3-postgresql-high-availability--disaster-recovery)
  - [6. Infrastructure as Code (Total rewrite)](infrastructure.md#6-infrastructure-as-code-total-rewrite)
  - [7. Disaster Recovery](infrastructure.md#7-disaster-recovery)
  - [8. Canary Deployments](infrastructure.md#8-canary-deployments)
  - [15. Диаграмма зависимостей](infrastructure.md#15-диаграмма-зависимостей)
  - [18. Migration Strategy](infrastructure.md#18-migration-strategy)
- [Compliance](compliance.md)
  - [4. Compliance: 152-ФЗ](compliance.md#4-compliance-152-фз)
- [Observability](observability.md)
  - [5. Наблюдаемость и SLO](observability.md#5-наблюдаемость-и-slo)
- [Medical](medical.md)
  - [12. Wellness boundary](medical.md#12-wellness-boundary)
  - [13. Юридический статус](medical.md#13-юридический-статус-wellness-не-медицинское-по)
- [ML](ml.md)
  - [14. Ежедневная адаптивная модификация плана](ml.md#14-ежедневная-адаптивная-модификация-плана)
  - [16. Оценка стоимости (руб/мес)](ml.md#16-оценка-стоимости-рубмес)
  - [17. Resource Plan: FTE](ml.md#17-resource-plan-fte)
- [Devices](devices.md)
  - [24. Расширение поддерживаемых устройств](devices.md#24-расширение-поддерживаемых-устройств)
- [Yandex ID](yandex-id.md)
  - [25. Production verification Yandex ID](yandex-id.md#25-production-verification-yandex-id)

---

## Timeline

| Квартал | Фокус |
| --- | --- |
| Q1 2027 | PostgreSQL HA, Vault hardening, Observability базовый уровень, quarterly access review |
| Q2 2027 | Service Mesh, Secrets Rotation, ELK/Fluent Bit, CAPTCHA production |
| Q3 2027 | Canary Deployments, GDPR/152-ФЗ compliance завершение, Medical boundary docs |
| Q4 2027 | Pen Test, Bug Bounty, Adaptive ML retrain, DPA/contracts финализация |

## Gantt Chart

```text
Неделя:    1    2    3    4    5    6    7    8    9    10   11   12   13   14
PostgreSQL HA:  [████████████████████████████████████████████████████████████████████████████████████████████████]
Vault hardening:       [████████████████████████████████████████████████████████████████████████████████████████████████]
Observability base:            [████████████████████████████████████████████████████████████████████████████████████████████████]
Service Mesh:                      [████████████████████████████████████████████████████████████████████████████████████████████████]
Secrets Rotation:                           [████████████████████████████████████████████████████████████████████████████████████████████████]
ELK/Fluent Bit:                                   [████████████████████████████████████████████████████████████████████████████████████████████████]
Canary Deploy:                                          [████████████████████████████████████████████████████████████████████████████████████████████████]
Compliance:                                                 [████████████████████████████████████████████████████████████████████████████████████████████████]
Pen Test + Bug Bounty:                                                [████████████████████████████████████████████████████████████████████████████████████████████████]
Adaptive Retrain:                                                          [████████████████████████████████████████████████████████████████████████████████████████████████]
```

## Critical Path

```text
PostgreSQL HA → Vault hardening → Observability base → Service Mesh → Secrets Rotation → ELK → Canary Deploy → Compliance → Pen Test → Adaptive Retrain
```

**Любая задержка в Critical Path задержит всю Phase 2 на 1–2 недели.**

## Dependencies

- Phase 1 production ✅ — все сервисы работают, документировать текущее состояние перед изменениями
- PostgreSQL HA требует VPS upgrade (4 vCPU / 16 ГБ RAM минимум) или 2+ отдельных серверов
- Service Mesh требует Istio/Linkerd, несовместим с текущим hand-rolled mTLS без миграции
- ELK/Fluent Bit требует отдельного кластера или managed-сервиса

## Risk Register

| Риск | Вероятность | Влияние | Митигация |
| --- | --- | --- | --- |
| VPS не поддерживает HA-компоненты одновременно | Высокая | Высокое | Апгрейд до 4 vCPU / 16 ГБ или разделение на 2 VPS |
| Service Mesh увеличивает latency | Средняя | Среднее | Canary migration namespace-by-namespace, мониторинг p95 |
| ELK не влезает в текущий VPS | Высокая | Среднее | Managed Elasticsearch / S3 archive для старых логов |
| Реклассификация как медицинское ПО | Низкая | Высокое | Держать wellness-статус, disclaimer, юридическая экспертиза |
| Переобучение ML влияет на API latency | Средняя | Среднее | Выделенный воркер 2+ vCPU, DVC, canary rollout моделей |
