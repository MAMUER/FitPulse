# ADR 0007: Релизный пайплайн

## Контекст

Проект требует надёжного релизного процесса для обеспечения качества, безопасности и быстрого отката в production.

## Решение

Спроектировать релизный пайплайн:

1. **Development**: feature-ветки с pre-commit хуками.
2. **Code Review**: 2+ approve, SAST, dependency scans.
3. **CI Build**: unit/integration/contract тесты, сканирование контейнеров, multi-arch сборка.
4. **Deploy Test**: автоматические smoke-тесты.
5. **Deploy Staging**: UAT, performance/security тесты, chaos engineering.
6. **Release Candidate**: Git tags, changelogs, migration plans.
7. **Post-Deploy Monitoring**: 24ч наблюдение с определёнными метриками.

## Последствия

- обеспечивает высокое качество релизов с комплексным тестированием;

## Реализация

- настроен CI/CD пайплайн (GitHub Actions `.github/workflows/ci.yml`) с 9 этапами;
- настроены мониторинговые дашборды для пост-деплойного наблюдения;

## Рассмотренные альтернативы

- Меньше этапов: снижение качества assurance.
