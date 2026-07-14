# Service Level Agreement (SLA)

## Scope

- Применяется к production-окружению (`fitness-platform-production` namespace)
- Охватывает доступность API endpoints и микросервисов
- Не охватывает сторонние сервисы (SMTP, OAuth провайдеры)

## Uptime Target

| Phase | Topology | Target Uptime | Notes |
|-------|----------|---------------|-------|
| Phase 1 | single-node k3s | **99.0%** | Без HA, single point of failure; planned maintenance excluded |
| Phase 2+ | multi-node k3s / k8s | **99.5%** | При наличии реплик и автоматического восстановления |

Расчёт: downtime ≤ 3.65 дня/год для 99.0%, ≤ 1.83 дня/год для 99.5%.
Planned maintenance windows исключаются из расчёта.

## Exclusions

- Плановые технические работы (объявляются заранее)
- Force majeure (стихийные бедствия, DDoS)
- Ошибки внешних провайдеров

## Severity & Response Times

Оценка серьезности соответствует секции "Типы уязвимостей" в [`SECURITY.md`](SECURITY.md).

Кратко (best effort, без юридических гарантий):

| Priority | Response Time | Resolution Time |
|----------|---------------|-----------------|
| 🔴 Critical (Security) | 48 часов — подтверждение; 7 рабочих дней — assessment | 30 рабочих дней — план исправления |
| 🟠 High (Bug blocking) | 3–7 рабочих дней | 2–4 недели |
| 🟡 Medium (Feature) | 1–2 недели | Следующий релиз |
| 🟢 Low (Nice to have) | Следующий релиз | Best effort |

## Важно

- **Данный SLA не является юридическим обязательством**. Проект распространяется "как есть" без гарантий. Время реакции и исправления указаны в качестве рекомендации (best effort) и могут отличаться в зависимости от доступности мейнтейнеров.
- Проект поддерживается добровольцами без команды 24/7. Реакция осуществляется в свободное от основной работы время.
- Для security-уязвимостей используется отдельный процесс responsible disclosure — см. `BUG_BOUNTY_SCOPE.md` и `SECURITY.md`.
