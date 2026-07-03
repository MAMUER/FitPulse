# FitPulse — Bug Bounty Scope

FitPulse — open-source fitness platform.
На текущем этапе программа **не предполагает денежного вознаграждения**: проект запускается без бюджета на выплаты.
Мы принимаем добровольные сообщения об уязвимостях и публично атрибутируем исследователей в Security Advisories и `SECURITY.md`.

---

## Статус

- **Текущий статус**: не подразумевает денежное вознаграждение
- **Причина**: проект без бюджета, free open-source
- **Форма признания**: honourable mention в Security Advisories + публичное спасибо в `SECURITY.md`
- **Вознаграждение**: нет денежного вознаграждения на текущем этапе

---

## In Scope

| Target | Notes |
| -------- | ------- |
| `https://fittpulse.duckdns.org` | production domain + все поддомены при их появлении |
| Веб-интерфейс (`web/`, `web/static/`, `web/templates/`) | frontend, статика, шаблоны |
| Все API endpoints (`/api/v1/...`) | auth, biometrics, training, profile, devices, admin (`/api/v1/admin/*`), ML classification/generation |
| Исходный код сервисов (`cmd/*`, `api/*`, `internal/*`) | backend, protobuf, адаптеры |
| Инфраструктура: K8s deployment manifests, NGINX configs (`deploy/lb/`), scripts (`scripts/`, `configs/k8s/scripts/`) | без доступа к живому кластеру |
| CI/CD workflows и secrets handling | без доступа к GitHub Secrets |

### Исключения из in-scope

- GitHub Secrets и другие реальные секреты инфраструктуры недоступны для тестирования
- Живой кластер, PostgreSQL, Valkey, RabbitMQ — не предоставляются для тестирования; уязвимости в них принимаются только как доказательства через публичный интерфейс

---

## Out of Scope

- Документация (`docs/`, `*.md`) без sensitive data
- Dev-окружение без production данных
- Внутренние IP и сервисы без публичного доступа
- Внутренние gRPC-сервисы и базы данных, не exposed через интернет
- DoS-атаки, фuzzing без явного разрешения
- Социальная инженерия за пределами взаимодействия с публичным интерфейсом проекта

---

## Reporting

Используйте **GitHub Security Advisory** (репозиторий → Security → "Report a vulnerability") или email: `mihnikolaenko12@yandex.ru`

Ожидаемый ответ:

- 48 часов — подтверждение получения
- 7 дней — assessment и triage
- 30 дней — план исправления для критических уязвимостей

---

## Severity & Response

| Severity | Примеры | Время ответа |
| ---------- | ---------- | -------------- |
| Critical | RCE, SQLi с доступом к данным, auth bypass, утечка PII/tokens, подделка JWT/2FA | 24–48 часов |
| High | XSS, CSRF, недостатки контроля доступа, небезопасная десериализация, обход rate limit/auth middleware | 3–7 дней |
| Medium | Missing security headers, weak crypto, info disclosure, небезопасная конфигурация NGINX/K8s | 7–14 дней |
| Low | Missing rate limiting, verbose errors, missing CSP directives | 14–30 дней |

---

## Responsible Disclosure

- **Disclosure deadline**: 90 дней с момента первого контакта
- **No exploitation beyond PoC**
- **No data exfiltration beyond what is necessary for proof**
- **No DoS / disruption of service**
- **No social engineering beyond testing scope**

---

## Контакты

- **GitHub Security Advisory**: [Create a security advisory](https://github.com/MAMUER/fitpulse/security/advisories)
- **Email**: [mihnikolaenko12@yandex.ru](mailto:mihnikolaenko12@yandex.ru)

---

### Последнее обновление: 2026-07-03
