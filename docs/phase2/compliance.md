# Compliance

## 4. Compliance: 152-ФЗ, GDPR и медицинские данные

### 4.1 Текущий статус Phase 1

| Требование | Статус Phase 1 | Что требуется в Phase 2 |
| --- | --- | --- |
| Шифрование at rest | ✅ pgsodium (aegis256), pgcrypto | Расширить на все категории ПДн |
| Шифрование in transit | ✅ TLS 1.3, mTLS | Service Mesh для автоматической ротации |
| Аудит действий | ⚠️ ConsoleAuditLogger (stdout) | Централизованный ELK/Fluent Bit с retention 3 года |
| Хранение на территории РФ | ✅ VPS в РФ | Убедиться, что backup storage также в РФ |
| Срок хранения audit logs | ⚠️ retention не настроен | 3 года для audit logs, 90 дней для метрик |
| Cookie consent | ✅ UI готов | Привязать к GDPR Art. 7 |
| DPA с Yandex ID | ✅ Документ готов | Убедиться, что согласие собирается явно |
| Quarterly access review | ⚠️ Runbook готов | Автоматизировать через cronjob / GitHub Action |
| Medical boundary | ✅ Документация готова | Убедиться, что код не пересекает границу |
| DPIA | ✅ Готов | Пересматривать при изменении архитектуры |

### 4.2 Acceptance Criteria Phase 2

| Критерий | Целевое состояние | Proof |
| --- | --- | --- |
| Централизованный audit trail | Все audit-события в Elasticsearch с retention 3 года | `internal/audit/audit.go` → Fluent Bit → ES |
| Cookie consent banner | Везде, где собираются ПДн, есть явное согласие | `web/src/components/CookieConsent.jsx` |
| DPA compliance | Все third-party процессоры имеют DPA | `docs/compliance/DPA_YANDEX_ID.md`, реестр в `РЕЕСТР_ОБРАБОТКИ_ПДН.md` |
| Quarterly access review | Автоматический чеклист + отчёт | `docs/runbooks/QUARTERLY_ACCESS_REVIEW.md` + cronjob |
| Medical boundary enforcement | Технические флаги wellness/medical в коде | Флаг `is_medical_data` в repository layer |
| GDPR Art. 20 portability | JSON export стандартизирован | `GET /api/v1/profile/export` |
| Breach notification SLA | Уведомление Роскомнадзора за 72 часа | Runbook `INCIDENT_RESPONSE.md` |

### 4.3 Зависимости

- **ELK/Fluent Bit** — требуется для централизованного audit trail (зависит от Infrastructure)
- **Cookie Consent** — уже готов в UI, требуется только валидация
- **DPA** — документ готов, требуется только явный consent в UI
- **Quarterly Access Review** — runbook готов, требуется автоматизация

### 4.4 Ссылки

- [Политика обработки ПДн](../compliance/ПОЛИТИКА_ОБРАБОТКИ_ПДН.md)
- [DPA с Yandex ID](../compliance/DPA_YANDEX_ID.md)
- [Реестр обработки ПДн](../compliance/РЕЕСТР_ОБРАБОТКИ_ПДН.md)
- [DPIA](../compliance/DPIA.md)
- [Runbook: Инциденты](../runbooks/INCIDENT_RESPONSE.md)
- [Runbook: Quarterly Access Review](../runbooks/QUARTERLY_ACCESS_REVIEW.md)
