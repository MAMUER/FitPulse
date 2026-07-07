# ADR 0016: Операционная инфраструктура — SLA, Issue Automation и Incident Runbook

## Статус

Принято

## Контекст

После первичного релиза проект получил операционные артефакты, обеспечивающие управление инцидентами, автоматизацию issue-трекинга и соглашения об уровне сервиса. Эти артефакты не были описаны в исходных ADR-0001–0011.

## Решение

1. **SLA (`.github/SLA.md`)**:
   - отдельные временные рамки реакции и исправления по приоритету (best effort, без юридических гарантий):
     - Critical (Security): 1–3 рабочих дня / 1–2 недели
     - High (Bug blocking): 3–7 рабочих дней / 2–4 недели
     - Medium: 1–2 недели / следующий релиз
     - Low: следующий релиз / best effort

2. **GitHub Issue Automation (не реализовано)**:
   - автоматическое triage-лейблирование новых issues: не реализовано;
   - назначение ответственных по ключевым словам (`gateway`, `biometric`, `ml`, `device`, `frontend`, `k8s`, `docker`, `ci/cd`): не реализовано;
   - приветствие новых контрибьюторов: не реализовано;
   - секретный канал для security-labeled issues: не реализовано;
   - Telegram-уведомление о `priority:critical`: не реализовано.

3. **Incident Response Runbook (`docs/runbooks/INCIDENT_RESPONSE.md`)**:
   - классификация по SEV-1..SEV-4;
   - фазы ответа: triage → stabilization → resolution → recovery;
   - акроним IC (Incident Commander), playbook роли.

## Последствия

- **Плюсы**: ускорение TTM на поддержку пользователей, понятные ожидания по приоритетам, автоматический routing ответственности.
- **Нейтрально**: требует поддержания актуальности при росте команды/провайдеров.
- **Риски**: Telegram-токен/chat-ID как секреты в GitHub — нужны rotation и audit.

## Реализация

- `.github/SLA.md`
- `docs/runbooks/INCIDENT_RESPONSE.md`
- `docs/runbooks/OPERATIONS_RUNBOOK.md` (отмечен как зависимость в playbook).
