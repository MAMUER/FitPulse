# ADR 0017: UI Specification — Мобильное веб-приложение с 6 вкладками и Chart.js

## Статус

Принято

## Контекст

Frontend начал оформляться как SPA на vanilla JS/ES2026 с мобильным viewport-first дизайном. Требовалось задокументировать экраны, элементы, API-интеграцию и UI-токены, чтобы разработка шла согласованно с бэкенд-контрактами и безопасными практиками.

## Решение

Создать `docs/UI_SPECIFICATION.md`:

1. **Архитектура**: единый `web/index.html` с view-переключением по классу `active`; `tab-bar` с 6 вкладками (Обзор, Профиль, Тренировки, Устройства, Достижения, Диета).
2. **Auth flow**: экран авторизации с тремя состояниями — login, register, verify (dev-token mode + production email confirmation).
3. **Dashboard**: 4 health-summary карточки (пульс, SpO₂, сон, давление), Chart.js график пульса, AI-рекомендации, today’s workout карточка.
4. **Profile**: форма с groups (основное, параметры тела, образ жизни, цели) + модалки смены пароля/email + danger-zone с удалением аккаунта.
5. **Training**: список планов, пустое состояние, FAB для генерации через форму параметров (durationWeeks, maxDuration, preferredTime, days, equipment).
6. **Achievements**: сетка карточек достижений и список соревнований.
7. **Diet**: карточки приёмов пищи (калории, БЖУ).
8. **ML**: classify state (6 классов) + generate plan; читается из `/ml/classify` и `/ml/generate-plan`.
9. **Безопасность**: XSS (`textContent`), CSP nonce-based, HTTPS-only, JWT в `httpOnly` cookie (`Secure`, `SameSite=Strict`), rate-limit UI на 429.
10. **API-слой**: `web/static/js/api.js` централизует все 16 REST-вызовов.

## Последствия

- **Плюсы**: единая точка истины для фронтенда и бэкенда; минимальный стек (vanilla JS) без сборщика.
- **Нейтрально**: требует синхронизации с proto/REST-контрактами; изменение API требует синхронного обновления spec.
- **Риски**: отсутствие type-safe API-клиента может привести к runtime-ошибкам при изменениях API.

## Реализация

- `docs/UI_SPECIFICATION.md`
- `web/index.html`, `web/static/js/api.js`, `web/static/js/app.js`, `web/static/js/modules.js`
- `web/templates/*.html` для отдельных страниц.
