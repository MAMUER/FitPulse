# Security

## 2. Service Mesh

### 2.1 Текущий статус Phase 1

Phase 1 покрывает базовый mTLS между микросервисами на уровне gRPC (TLS 1.3, hand-rolled certs). Phase 2 переводит внутренние коммуникации на полноценный service mesh (Istio/Linkerd) с автоматической ротацией сертификатов, SPIFFE ID и распределённым трейсингом.

### 2.2 Acceptance Criteria

- mTLS активен между всеми сервисами через service mesh (Istio/Linkerd) с автоматической ротацией сертификатов через cert-manager
- Статические сертификаты в Kubernetes Secret удалены, все сертификаты генерируются динамически и монтируются через sidecar
- egress/ingress traffic control через AuthorizationPolicy
- Внешний доступ к сервисам возможен только через Gateway

---

## 9. Security Reporting: Corporate Email + PGP

### 9.1 Текущий статус Phase 1

Текущий security reporting использует личный email (`mihnikolaenko12@yandex.ru`), что нарушает best practices. Базовая self-hosted политика уже реализована: созданы `BUG_BOUNTY_SCOPE.md` и раздел в `SECURITY.md`, определены in-scope/out-of-scope цели и transparent SLA по ответу (best effort).

В Phase 2 требуется:

- заменить личный email на корпоративный alias
- усилить криптографическую защиту отчётов через PGP
- оценить возможность перехода на профессиональные платформы или выделения бюджета

**Связанные разделы**:

- Раздел 10 (ежеквартальный внешний пентест) зависит от наличия корпоративного email
- Раздел 11 (Bug Bounty) интегрируется с этим контуром

### 9.2 Acceptance Criteria

- PGP key fingerprint опубликован в `SECURITY.md` и `BUG_BOUNTY_SCOPE.md`.
- PGP-ключ настроен и доступен через WKD.
- Все security-отчёты принимаются на корпоративный ящик `mihnikolaenko12@yandex.ru`.
- Личный email `mihnikolaenko12@yandex.ru` больше не указан как primary контакт.
- PGP-ключ защищает in-transmit отчёты.
- Есть процедура ротации доступа к ящику.
- Принято финальное решение по бюджету и/или платформенной интеграции (с документированным обоснованием).

---

## 10. CAPTCHA (Cloudflare Turnstile)

### 10.1 Текущий статус Phase 1

Phase 1 использует жёсткий rate limiting (при превышении порога — блокировка). Это создаёт риск отзыва legitimate-пользователей при ложных срабатываниях, cross-user NAT и burst.

CAPTCHA частично реализован: `cmd/gateway/captcha.go`, манифесты в `configs/k8s/base/captcha/`.

### 10.2 Acceptance Criteria

- При превышении порога rate limit пользователь видит CAPTCHA, а не жёсткий блок.
- Успешное решение CAPTCHA снимает ограничение на фиксированный период (например, 5 минут).
- CAPTCHA логируется с correlationId и участвует в RED metrics.

---

## 11. Secrets Rotation Automation

### 11.1 Текущий статус Phase 1

Phase 1 использует Vault + External Secrets Operator (SecretStore → ExternalSecret) для автоматической инъекции секретов. Ротация секретов работает через Vault, но требуется автоматизация ротации без перезапуска сервисов.

**Связанные разделы**:

- Раздел 6 (Infrastructure as Code) — развёртывание Vault
- Раздел 2 (Service Mesh) — cert-manager для сертификатов mesh

### 11.2 Acceptance Criteria

- Все секреты инжектируются в поды автоматически без перезапуска сервисов.
- Ротация происходит прозрачно для приложений.
- Алерты срабатывают при проблемах с ротацией.
