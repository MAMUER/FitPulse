# ADR 0016: Операционная инфраструктура — SLA, Issue Automation и Incident Runbook

## Статус

Принято

## Контекст

После первичного релиза проект получил операционные артефакты, обеспечивающие управление инцидентами, автоматизацию issue-трекинга и соглашения об уровне сервиса. Эти артефакты не были описаны в исходных ADR-0001–0011.

## Решение

1. **SLA (`.github/SLA.md`)**:
   - отдельные временные рамки реакции и исправления по приоритету:
     - Critical (Security): 4ч / 24ч
     - High (Bug blocking): 1 день / 3 дня
     - Medium: 3 дня / 2 недели
     - Low: 1 неделя / следующий релиз

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
