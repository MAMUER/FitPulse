# ADR 0017: UI Specification — React SPA с Vite и React Router v7

## Статус

Принято

## Контекст

Frontend был переведён с vanilla JS/ES2026 на React 19 с Vite для улучшения поддерживаемости, type-safe рендеринга и современной архитектуры компонентов. Требовалось задокументировать экраны, элементы, API-интеграцию и UI-токены.

## Решение

Создать `docs/UI_SPECIFICATION.md` и мигрировать фронтенд на React:

```text
web/
├── index.html
├── package.json
├── vite.config.js
├── eslint.config.js
├── vitest.config.js
├── src/
│   ├── main.jsx
│   ├── App.jsx
│   ├── index.css
│   ├── contexts/
│   │   └── AuthContext.jsx
│   ├── utils/
│   │   ├── api.js
│   │   ├── validators.js
│   │   └── exerciseNames.js
│   ├── components/
│   │   ├── Layout/
│   │   ├── Auth/
│   │   ├── Dashboard/
│   │   ├── Profile/
│   │   ├── Training/
│   │   ├── Devices/
│   │   ├── Achievements/
│   │   ├── Diet/
│   │   ├── Health/
│   │   ├── ML/
│   │   └── Admin/
│   └── test/
│       └── setup.js
├── static/
│   ├── fonts/
│   └── errors/
└── dist/
    ├── index.html
    ├── static/
    └── errors/
```

1. **Архитектура**: React 19 + Vite + React Router v7; компоненты в `web/src/components/`; контексты в `web/src/contexts/`; утилиты в `web/src/utils/`.
2. **Auth flow**: экран авторизации с состояниями login, register, verify и 2FA — dev-token mode + production email confirmation.
3. **Dashboard**: 4 health-summary карточки (пульс, SpO₂, сон, давление), Chart.js график пульса, AI-рекомендации, today’s workout карточка.
4. **Profile**: форма с groups (основное, параметры тела, образ жизни, цели) + модалки смены пароля/email + danger-zone с удалением аккаунта.
5. **Training**: список планов, пустое состояние, FAB для генерации через форму параметров.
6. **Achievements**: сетка карточек достижений и список соревнований.
7. **Diet**: карточки приёмов пищи (калории, БЖУ) + калькулятор калорий по Mifflin-St Jeor.
8. **Admin**: панель администратора с управлением пользователями/приглашениями (только для роли admin).
9. **ML**: classify state (6 классов) + generate plan; читается из `/ml/classify` и `/ml/generate-plan`.
10. **Безопасность**: XSS (`textContent`), CSP nonce-based, HTTPS-only, JWT в `httpOnly` cookie (`Secure`, `SameSite=Strict`), rate-limit UI на 429.
11. **API-слой**: `web/src/utils/api.js` централизует все REST-вызовы.
12. **Тестирование**: Vitest + React Testing Library; ESLint 9 с React плагинами.

## Последствия

- **Плюсы**: компонентная архитектура, переиспользуемость, type-safe JSX, современный стек (Vite), автocomplete в IDE.
- **Нейтрально**: требуется Node.js 24+ для сборки; чуть больше зависимостей.
- **Риски**: миграция требовала переписывания всего фронтенда; старые файлы удалены.

## Реализация

- `docs/UI_SPECIFICATION.md`
- `web/src/` — все React компоненты, контексты, утилиты
- `web/vite.config.js`, `web/package.json`
- `web/eslint.config.js`, `web/vitest.config.js`
- `web/static/` — шрифты и HTML-страницы ошибок
- `web/dist/` — production сборка Vite
- Старые файлы `web/templates/`, `web/static/js/`, `web/static/css/` удалены
