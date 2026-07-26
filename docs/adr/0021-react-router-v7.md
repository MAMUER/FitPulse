# ADR 0021: Выбор React Router v7 для SPA-навигации

## Статус

Принято

## Контекст

После миграции на React (ADR 0020) требовался роутер для SPA, который бы:
- поддерживал вложенные layout (top bar + tab bar);
- обеспечивал защищённые маршруты (только для авторизованных пользователей);
- давал нативную кнопку назад в браузере;
- интегрировался с auth-контекстом (редирект на логин при отсутствии токена);
- имел стабильный API и активную поддержку.

## Решение

Использовать React Router v7 (`react-router-dom`):

- `<BrowserRouter>` как корневой контейнер;
- `<Routes>` + `<Route>` для декларативной маршрутизации;
- `<Navigate>` для redirect и protected routes;
- `useNavigate()` для программной навигации;
- Layout-компонент (`Layout.jsx`) оборачивает protected routes и содержит top bar + tab bar.

## Последствия

- **Плюсы**: стандартное решение для React SPA, стабильное API, хорошая документация.
- **Нейтрально**: требуется изучение v7 API для команды; конфигурация через JSX, а не declarative config.
- **Риски**: при major-апгрейде React Router могут измениться API; команда должна отслеживать breaking changes.

## Рассмотренные альтернативы

- **Next.js App Router**: перебор для SPA; добавляет server-side сложности.
- **Remix**: избыточен, требует отдельного сервера.
- **SvelteKit**: другой фреймворк, не соответствует стековому решению.
- **HashRouter**: проще для статического хостинга, но URL без history API.

## Реализация

- `web/package.json` — `react-router-dom: ^7.18.1`.
- `web/src/main.jsx` — `<BrowserRouter>` + `<AuthProvider>`.
- `web/src/App.jsx` — `/login`, `/register`, `/verify`, `/confirm`, `/dashboard`, `/profile`, `/training`, `/devices`, `/achievements`, `/diet`, `/health`, `/ml`, `/admin` с Layout и защитой.
