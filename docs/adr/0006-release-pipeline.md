# ADR 0006: Релизный пайплайн

## Статус

Принято

## Контекст

Проект требует надёжного релизного процесса для обеспечения качества, безопасности и быстрого отката в production.

## Решение

Спроектировать релизный пайплайн:

1. **Development**: feature-ветки с pre-commit хуком `scripts/pre-commit` (TruffleHog secret scan).
2. **Code Review**: SAST (Semgrep, gosec, CodeQL), dependency scans (Trivy, govulncheck, Dependabot). Branch protection с 2+ approve не требуется: проект поддерживается одним человеком.
3. **CI Build**: unit/integration/contract тесты, сканирование контейнеров (Trivy), multi-arch сборка через Docker Buildx.
4. **Deploy Test**: автоматические smoke-тесты через testcontainers-go.
5. **Deploy Staging**: UAT-тесты, security scans. Performance тесты не реализованы.
6. **Release Candidate**: Git tags с SLSA provenance, миграции БД через Flyway. Changelog generation не реализован.
7. **Post-Deploy Monitoring**: immediate post-deploy проверки (логи, health checks). 24ч наблюдение задокументировано как организационный процесс в `docs/runbooks/INCIDENT_RESPONSE.md` (Post-deploy monitoring checklist): ответственный следит за Telegram-алертами и Grafana в течение 24ч после деплоя.

## Последствия

- обеспечивает высокое качество релизов с комплексным тестированием;

## Реализация

- настроен CI/CD пайплайн (GitHub Actions `.github/workflows/ci.yml`) с security scanning (Semgrep, gosec, CodeQL, Trivy, Gitleaks, TruffleHog, Checkov, Kubescape, kube-hunter), unit/integration/contract тесты, multi-arch сборка через Buildx, Trivy image scan, cosign signing;
- pre-commit хук `scripts/pre-commit` (TruffleHog secret scan);
- деплой staging: namespace creation, image tags update, rollout wait, health verify, UAT tests;
- деплой production: `kubectl apply` для manifests, Flyway migrations, seed-admin, post-deploy checks (logs, pod status, deployment verify);
- GitHub releases с SLSA provenance;
- Canary-деплой и автоматические триггеры отката по метрикам запланированы на Phase 2.
- Changelog generation не реализован.
- Branch protection не требуется.
- 24ч пост-деплойного наблюдения задокументировано как организационный процесс в `docs/runbooks/INCIDENT_RESPONSE.md`.

## Рассмотренные альтернативы

- Меньше этапов: снижение качества assurance.
