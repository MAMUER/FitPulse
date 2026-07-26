# ADR 0020: Миграция фронтенда на React 19 + Vite

## Статус

Принято

## Контекст

Frontend был реализован как набор vanilla JS-файлов (`web/static/js/api.js`, `app.js`, `modules.js`) с ручной маршрутизацией view и HTML-шаблонами в `web/templates/`. Это давало быстрый старт, но при росте экранов и логики возникли проблемы:
- отсутствие компонентной переиспользуемости;
- ручное управление DOM и state;
- сложность поддержки при добавлении новых экранов (Auth, Dashboard, Profile, Training, Devices, Achievements, Diet, Health, ML, Admin);
- слабая типизация и нет type-safe рендеринга;
- сложность тестирования UI-логики.

## Решение

Перевести фронтенд на React 19 + Vite + React Router v7:

- **Vite** как сборщик: быстрая разработка, HMR, единый конфиг.
- **React Router v7** для SPA-навигации: защищённые маршруты, вложенные layout, кнопка назад работает нативно.
- **Компонентная архитектура**: каждый экран — компонент в `web/src/components/`.
- **AuthContext**: единый контекст для токена, пользователя и флага `isAdmin`.
- **Chart.js (react-chartjs-2)**: графики пульса и других метрик.
- **CSS**: сохраняем как plain CSS файлы для прямой портировки, без CSS-modules/styled-components.
- **Vite proxy**: `/api` → `localhost:8080` для разработки.
- **Тестирование**: Vitest + React Testing Library; Biome для линтинга и форматирования.

## Последствия

- **Плюсы**: компонентная архитектура, переиспользуемость, type-safe JSX, современный стек, автocomplete в IDE.
- **Нейтрально**: требуется Node.js 24+ для сборки; чуть больше зависимостей.
- **Риски**: миграция требовала переписывания всего фронтенда; старые файлы удалены.

## Рассмотренные альтернативы

- **Оставить vanilla JS**: проще для миника, но не масштабируется.
- **Next.js**: полный фреймворк, но избыточен для SPA на одном Gateway.
- **Svelte**: меньше кода, но меньше экосистемы и командного опыта.
- **CSS Modules / styled-components**: более scalable стили, но сложнее портировать существующий CSS.

## Реализация

- `web/package.json` — React 19, React Router v7, Chart.js, Vite, Biome, Vitest.
- `web/vite.config.js` — proxy `/api` → backend, alias `@` → `src/`.
- `web/src/main.jsx` — точка входа: BrowserRouter + AuthProvider.
- `web/src/App.jsx` — роутер с protected routes.
- `web/src/contexts/AuthContext.jsx` — auth state management.
- `web/src/components/` — все экраны: Auth, Confirm, Dashboard, Profile, Training, Devices, Achievements, Diet, Health, ML, Admin.
- `web/src/utils/api.js` — все REST-вызовы.
- `web/biome.json`, `web/vitest.config.js` — линтинг/формат и тесты.
- Удалены `web/templates/`, `web/static/js/`, `web/static/css/`.
- `cmd/gateway/main.go` — статика теперь раздаётся из `web/dist/`.
- `cmd/gateway/handlers_auth.go` — `/confirm` теперь отдает `web/dist/index.html` вместо `web/templates/confirm.html`.
