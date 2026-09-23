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
| `https://fittpulse.ru` | текущий production-домен. При переходе на дополнительный домен он автоматически добавляется в scope. |
| Веб-интерфейс (`web/src/`, `web/static/fonts/`, `web/static/errors/`) | React SPA, шрифты, страницы ошибок |
| Все API endpoints (`/api/v1/...`) | auth, biometrics, training, profile, devices, admin (`/api/v1/admin/*`), ML classification/generation |
| Исходный код сервисов (`cmd/*`, `api/*`, `internal/*`) | backend, protobuf, адаптеры |
| Инфраструктура: K8s deployment manifests, Ingress NGINX ModSecurity configs (`configs/k8s/base/ingress-nginx/`), scripts (`scripts/`, `configs/k8s/scripts/`) | без доступа к живому кластеру |
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

## PGP Encryption

Для защиты чувствительных отчётов об уязвимостях при передаче по email используйте PGP-шифрование.

**PGP fingerprint**: `-----BEGIN PGP PUBLIC KEY BLOCK-----

mDMEarNy8RYJKwYBBAHaRw8BAQdALnbfkcW/gpHJIdFDIhl1RsZzRbPmohRMllrT
+82BRm60LUZpdFB1bHNlIFNlY3VyaXR5IDxtaWhuaWtvbGFlbmtvMTJAeWFuZGV4
LnJ1PoiWBBMWCgA+FiEE9M/2lOiJ2v5TztiQds+nIny3PfAFAmqzcvECGyMFCQHh
M4AFCwkIBwIGFQoJCAsCBBYCAwECHgECF4AACgkQds+nIny3PfAcSQEAq8DVtDUL
S5hBXHrX4CyqAei6Oemb68zuSR+xdPj1tE8BAPkdZ+eUBswjX+3pYimGXwutYeSj
KB2kZNQaaRr4/jwD
=vO9G
-----END PGP PUBLIC KEY BLOCK-----`  
**Public key server**: `hkps://keys.openpgp.org`  
**WKD endpoint**: `<https://fittpulse.ru/.well-known/openpgpkey/hu/`>

### Как отправить encrypted report

1. Скачайте наш публичный PGP-ключ:

   ```bash
   gpg --keyserver hkps://keys.openpgp.org --recv-keys 76CFA7227CB73DF0
   ```

2. Зашифруйте отчёт:

   ```bash
   gpg --encrypt --armor --recipient 76CFA7227CB73DF0 report.txt
   ```

3. Отправьте зашифрованный файл на `mihnikolaenko12@yandex.ru`.

## Reporting

Используйте **GitHub Security Advisory** (репозиторий → Security → "Report a vulnerability") или email: `mihnikolaenko12@yandex.ru`

**Рекомендуется**: шифруйте чувствительные отчёты PGP (см. раздел выше).

Ожидаемый ответ (best effort, без юридических гарантий):

- 48 часов — подтверждение получения
- 7 рабочих дней — assessment и triage
- 30 рабочих дней — план исправления для критических уязвимостей

## Severity & Response

Оценка серьезности соответствует секции "Типы уязвимостей" в [`SECURITY.md`](SECURITY.md). Ниже — ориентировочные временные рамки (best effort, без юридических гарантий):

| Severity | Примеры | Время реакции (ориентир) |
| -------- | ------- | ------------------------ |
| Critical | RCE, SQLi с доступом к данным, auth bypass, утечка PII/tokens, подделка JWT/2FA | 48 часов — подтверждение; 7 рабочих дней — assessment; 30 рабочих дней — план исправления |
| High | XSS, CSRF, недостатки контроля доступа, небезопасная десериализация, обход rate limit/auth middleware | 3–7 рабочих дней |
| Medium | Missing security headers, weak crypto, info disclosure, session management issues | 1–2 недели |
| Low | Missing rate limiting, verbose errors, missing CSP directives | Следующий релиз / best effort |

> **Важно**: SLA не является юридическим обязательством. Проект распространяется "как есть" без гарантий. Реакция осуществляется добровольцами в свободное время (best effort).

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

### Последнее обновление: 2026-09-23
