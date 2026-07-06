# Service Level Agreement (SLA)

## Scope

- Применяется к production-окружению (`fitness-platform-production` namespace)
- Охватывает доступность API endpoints и микросервисов
- Не охватывает сторонние сервисы (SMTP, OAuth провайдеры)

## Exclusions

- Плановые технические работы (объявляются заранее)
- Force majeure (стихийные бедствия, DDoS)
- Ошибки внешних провайдеров

| Priority | Response Time | Resolution Time |
| ---------- | --------------- | ----------------- |
| 🔴 Critical (Security) | 15 min | 4 hours |
| 🟠 High (Bug blocking) | 1 day | 3 days |
| 🟡 Medium (Feature) | 3 days | 2 weeks |
| 🟢 Low (Nice to have) | 1 week | Next release |
