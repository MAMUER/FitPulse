# FitPulse — Bug Bounty Scope

FitPulse — open-source fitness platform.  
Программа не является платной: на текущем этапе **бюджета/финансового вознаграждения нет**.  
Мы принимаем добровольные сообщения об уязвимостях и публично атрибутируем исследователей.

---

## Статус

- **Текущий статус**: неактивна в денежном выражении
- **Why**: проект без бюджета, free open-source
- **Форма признания**: honourable mention в Security Advisories + публичное спасибо в `SECURITY.md`
- **Вознаграждение**: нет денежного вознаграждения на текущем этапе

---

## In Scope

| Target | Notes |
| -------- | ------- |
| `https://fittpulse.duckdns.org` | production domain |
| Все API endpoints (`/api/v1/...`) | auth, biometrics, training, profile, devices, ML classification |
| `cmd/*`, `api/*`, `internal/*` | исходный код сервисов |
| Инфраструктура: K8s deployment manifests, NGINX configs, scripts | без доступа к реальному кластеру |
| CI/CD workflows и secrets handling | без доступа к GitHub Secrets |

### Исключения из in-scope

- `docs/`, `*.md` без sensitive data
- CI configs без disclosure secrets
- Инфраструктурные ресурсы, доступ к которым у вас нет (локальный dev)

---

## Out of Scope

- Документация без sensitive data
- Dev-окружение без production данных
- Внутренние IP и сервисы без публичного доступа
- DoS-атаки, фuzzing без явного разрешения

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
| ---------- | --------- | -------------- |
| Critical | RCE, SQLi с доступом к данным, auth bypass, утечка PII/tokens | 24–48 часов |
| High | XSS, CSRF, недостатки контроля доступа, небезопасная десериализация | 3–7 дней |
| Medium | Missing security headers, weak crypto, info disclosure | 7–14 дней |
| Low | Missing rate limiting, verbose errors, missing CSP | 14–30 дней |

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

## Последнее обновление: 2026-07-01
