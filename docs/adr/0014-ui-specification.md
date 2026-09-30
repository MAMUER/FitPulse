# ADR 0014: UI Specification — React SPA с Vite и React Router v7

## Статус

Принято

## Контекст

Frontend мигрировал на React 19 + Vite для компонентной архитектуры, type-safe JSX и удобства поддержки. Требовалось задокументировать экраны, API-интеграцию, токены и UI-токены в едином месте.

## Решение

Создать `docs/UI_SPECIFICATION.md` и мигрировать фронтенд на React:

```text
web/
├── index.html
├── package.json
├── vite.config.js
├── biome.json
├── vitest.config.js
├── src/
│   ├── main.jsx
│   ├── App.jsx
│   ├── index.css
│   ├── contexts/
│   │   └── AppContext.jsx
│   ├── hooks/
│   │   ├── useAuth.js
│   │   ├── useTraining.js
│   │   ├── useNutrition.js
│   │   ├── useProfile.js
│   │   ├── useHealth.js
│   │   ├── useAI.js
│   │   └── ...
│   ├── screens/
│   │   ├── Login.jsx
│   │   ├── Register.jsx
│   │   ├── Home.jsx
│   │   ├── Nutrition.jsx
│   │   ├── Training.jsx
│   │   ├── Profile.jsx
│   │   ├── AI.jsx / Chat.jsx
│   │   ├── Achievements.jsx
│   │   ├── Body.jsx
│   │   ├── Calendar.jsx
│   │   ├── Integrations.jsx
│   │   ├── TwoFASetup.jsx / TwoFAVerify.jsx
│   │   └── ...
│   ├── components/
│   │   ├── MetricCard.jsx
│   │   ├── Panel.jsx
│   │   └── ...
│   ├── services/
│   │   ├── api.js
│   │   └── storage.js
│   ├── utils/
│   │   ├── backendRequest.js
│   │   ├── i18n.js
│   │   └── helpers.js
│   ├── reducers/
│   │   ├── appState.js
│   │   └── appReducer.js
│   └── test/
│       └── setup.js
└── dist/
    ├── index.html
    └── static/
```

1. **Архитектура**: React 19 + Vite + React Router v7; экраны в `web/src/screens/`; хуки в `web/src/hooks/`; сервисы в `web/src/services/`.
2. **Auth flow**: экраны Login/Register/Reset + 2FA setup/verify. Детали реализации аутентификации описаны в ADR 0024.
3. **Dashboard/Home**: health-summary карточки, today's workout, AI-рекомендации.
4. **Profile**: форма с группами (основное, параметры тела, образ жизни, цели) + danger-zone.
5. **Training**: список планов, FAB для генерации через форму параметров.
6. **Achievements**: сетка карточек достижений, загрузка с бэкенда `/api/v1/achievements`.
7. **Diet/Nutrition**: карточки приёмов пищи (калории, БЖУ), период просмотра (день/неделя/месяц).
8. **AI/Chat**: классификация состояния, генерация плана, генерация диеты через `/api/v1/ml/chat`.
9. **Integrations**: управление источниками данных через Open Wearables.
10. **Body**:body composition экран.
11. **Calendar**: календарь тренировок.
12. **Videos**: видеоплощадки.
13. **Безопасность**: XSS mitigation через textContent; HTTPS-only в production. Детали аутентификации и cookie security описаны в ADR 0024.
14. **API-слой**: `web/src/services/api.js` и `web/src/utils/backendRequest.js` централизуют REST-вызовы.
15. **Тестирование**: Vitest + React Testing Library + Playwright E2E; Biome линтинг и форматирование.

## Последствия

- **Плюсы**: компонентная архитектура, переиспользуемость, современный стек (Vite + React 19), автocomplete в IDE.
- **Нейтрально**: требуется Node.js >=24 для сборки; чуть больше зависимостей.

## Реализация

- `docs/UI_SPECIFICATION.md`
- `web/src/screens/` — все экраны (Login, Register, Home, Nutrition, Training, Profile, AI, Chat, Achievements, Body, Calendar, Integrations, TwoFA, Legal и др.)
- `web/src/components/` — переиспользуемые компоненты (MetricCard, Panel, AuthLayout, EmptyState и др.)
- `web/src/hooks/` — кастомные хуки (useAuth, useTraining, useNutrition, useProfile, useHealth, useAI и др.)
- `web/src/contexts/AppContext.jsx` — глобальный state
- `web/src/services/api.js`, `web/src/utils/backendRequest.js` — API-слой
- `web/src/reducers/` — state management
- `web/vite.config.js`, `web/package.json`
- `web/biome.json`, `web/vitest.config.js`
- `web/static/` — шрифты и HTML-страницы ошибок
- `web/dist/` — production сборка Vite
