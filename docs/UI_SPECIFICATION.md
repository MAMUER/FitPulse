# FitPulse — UI Specification

> **Scope:** Мобильное веб-приложение (SPA) для пользователя системы FitPulse.
> **Target device:** Мобильный браузер (Viewport `390–430 px`, touch-first).
> **Base URL:** `https://fitpulse.duckdns.org:8443/`

---

## 1. Архитектура интерфейса

### 1.1. Тип приложения

Single Page Application (SPA), состоящая из одного HTML-файла `web/index.html` с переключением `view` по классу `active`.

### 1.2. Навигация

Нижний tab-bar с 6 кнопками. Переключение происходит мгновенно без перезагрузки страницы.

|Tab|Icon (SVG)|View ID|Заголовок|
|---|---|---|---|
|Обзор|4-quadrant grid|`dashboardView`|Обзор|
|Профиль|User circle|`profileView`|Профиль|
|Тренировки|Lightning bolt|`trainingView`|Тренировки|
|Устройства|Smartphone|`devicesView`|Устройства|
|Достижения|Trophy|`achievementsView`|Достижения|
|Диета|Container/bottle|`dietView`|Диета|

Дополнительные view без tab-bar:

- `mlView` (AI-анализ) — открывается как модалка/оверлей или отдельный экран;
- `mlClassify` / `mlGenerate` — отдельные templates.

### 1.3. Технологии

- **HTML5**, vanilla JS (ES2026), CSS-переменные.
- **Chart.js** (`chart.umd.min.js`) — график пульса на Dashboard.
- **Fetch API** через `web/static/js/api.js` для всех запросов к Gateway.
- JWT хранится в `httpOnly` cookie с флагами `Secure`, `SameSite=Strict` (подробности в разделе 11).

---

## 2. Экраны и состояние

### 2.1. Экран авторизации (`authScreen`)

Состояния (отображаются по классу `active` / `hidden`):

|Состояние|Форма / элемент|Описание|
|---|---|---|
|Логин|`loginForm`|Email + пароль|
|Регистрация|`registerForm`|Имя, email, пароль|
|Верификация email|`verifyForm`|Подтверждение токена|
|Ошибка|`authError` / `loginError` / `registerError` / `confirmError`|Общий блок ошибок|

**Поля формы логина:**

- `loginEmail` — `type=email`, `inputmode=email`, `maxlength=254`, `autocomplete=email`.
- `loginPassword` — `type=password`, `autocomplete=current-password`.

**Поля формы регистрации:**

- `regName` — `type=text`, `maxlength=100`, `minlength=2`, `pattern=[A-Za-zА-Яа-яЁё\s\-]+`.
- `regEmail` — email, `maxlength=254`.
- `regPassword` — `type=password`, `minlength=8`, с подсказкой `passwordHint` (длина, заглавная, строчная, цифра).

**Валидация:**

- `regPassword` в реальном времени обновляет `passwordHint`.
- Кнопка `registerBtn` заблокирована (`disabled`), до выполнения всех требований к паролю.
- При сабмите формы запрос уходит в `api.register()` → `POST /api/v1/register`.

**Верификация email:**

- После успешной регистрации показывается `verifyForm` с email-адресом.
- Если нет SMTP / в режиме разработки — показывается секция `devTokenSection` с токеном.
- Для production: пользователь переходит по ссылке из письма → токен передаётся в `POST /api/v1/auth/confirm`.

**Переходы:**

- `toRegister` — из логина в регистрацию.
- `toLogin` — из регистрации в логин.
- `backToLogin` — из верификации в логин.

### 2.2. Главное приложение (`mainScreen`)

Структура:

```text
header.top-bar
  h2#pageTitle
  button#logoutBtn
main.content
  section.view (один активный)
nav.tab-bar
  button.tab × 6
```

**Логика переключения view:**

- По клику на tab: удалить `active` у текущего view, добавить текущему, обновить `pageTitle`.
- При открытии view вызывается соответствующий `init*()` из `app.js`.

---

## 3. Обзор (Dashboard)

**View ID:** `dashboardView`
**Заголовок:** `Обзор`

### 3.1. Сводка здоровья (`health-summary`)

4 карточки в сетке 2×2:

|Карточка|ID|Единица|Диапазон|
|---|---|---|---|
|Пульс|`hrValue`|уд/мин|30–220|
|SpO₂|`spo2Value`|%|70–100|
|Сон|`sleepValue`|часов|0–24|
|Давление|`bpValue`|мм рт.ст.|—|

Данные подгружаются через `GET /profile` и `GET /biometrics`.

### 3.2. Динамика пульса (`chart-section`)

- `<canvas id="heartChart">` — последние 24ч / 7 дней.
- Библиотека: Chart.js.
- Данные: массив `{timestamp, value}` из `GET /biometrics?type=heart_rate`.

### 3.3. AI-анализ (`ai-section`)

- Карточка с классом `ai-card`.
- `aiRecommendation` — краткая рекомендация (например, «Восстановление»).
- `aiDescription` — пояснение.
- Source: результат `POST /ml/classify`.

### 3.4. Тренировка на сегодня (`today-section`)

- Блок `todayWorkout` — карточка с текущим / ближайшим занятием из активного плана.
- Если плана нет: `workout-placeholder` с текстом «Сгенерируйте программу тренировок в разделе "Тренировки"».

---

## 4. Профиль (`profileView`)

**View ID:** `profileView`
**Заголовок:** `Профиль`

### 4.1. Основная форма `profileForm`

Группы полей:

**Основное:**

- Никнейм (`profNickname`) — `maxlength=30`, обязательный.
- Возраст (`profAge`) — `type=number`, `min=18`, `max=100`.
- Пол (`profGender`) — select: `male`, `female`.

**Параметры тела:**

- Рост (`profHeight`) — см, `min=50`, `max=300`.
- Вес (`profWeight`) — кг, `min=20`, `max=500`, шаг `0.1`.

**Образ жизни:**

- Уровень подготовки (`profFitness`) — select: `beginner`, `intermediate`, `advanced`.
- Тип питания (`profNutrition`) — select: `balanced`, `high_protein`, `vegetarian`, `vegan`, `keto`, `paleo`.

**Цели** — radio-chips (`goal`):

- `weight_loss` — Похудение
- `muscle_gain` — Набор мышц
- `endurance` — Выносливость
- `flexibility` — Гибкость

### 4.2. Безопасность

Кнопки:

- `changePasswordBtn` — открывает модалку смены пароля.
- `changeEmailBtn` — открывает модалку смены email.

**Модалка смены пароля (`changePasswordModal`):**

- `currentPassword`
- `newPassword` + `passwordHint`
- `confirmPassword`
- Кнопки: `cancelChangePassword`, submit → `PUT /profile/security/password`.

**Модалка смены email (`changeEmailModal`):**

- `newEmail`
- `emailConfirmPassword`
- Кнопки: `cancelChangeEmail`, submit.

**Важно:** обе модалки имеют оверлей `.modal-overlay`. При клике на оверлей — закрытие.

### 4.3. Опасная зона (`danger-zone`)

- Заголовок красным: «Опасная зона».
- `deleteProfileBtn` → модалка подтверждения через пароль → `DELETE /profile`.

### 4.4. Эндпоинты

|Действие|Метод|Путь|
|---|---|---|
|Загрузить профиль|GET|`/profile`|
|Обновить профиль|PUT|`/profile`|
|Сменить пароль|PUT|`/profile/security/password`|
|Сменить email|PUT|`/profile/security/email`|
|Удалить профиль|DELETE|`/profile`|

---

## 5. Тренировки (`trainingView`)

**View ID:** `trainingView`
**Заголовок:** `Тренировки`

### 5.1. Список планов (`plansList`)

- Карточки планов: название, дата создания, статус (`active` / `completed` / `archived`).
- Пустое состояние: `empty-state` с иконкой `🏃` и текстом «AI создаст персональный план...».
- Эндпоинт: `GET /training/plans`.

### 5.2. Кнопка генерации

- `generatePlanBtn` — плавающая кнопка (FAB) внизу экрана.
- При нажатии открывается **модалка / отдельный экран** `ml-generate.html` с формой параметров:
  - Тип тренировки (select)
  - Длительность (недели), `min=1`, `max=12`
  - Макс. длительность сессии (мин)
  - Предпочтительное время суток
  - Дни тренировок (чекбоксы Пн–Вс)
  - Оборудование (чекбоксы)
- Submit → `POST /training/generate` или `POST /ml/generate-plan`.

### 5.3. Детали плана

При открытии плана:

- Список дней → упражнения → подходы/повторения.
- Кнопка «Завершить тренировку» → `POST /training/complete`.

### 5.4. Прогресс

- График прогресса (Chart.js) — `GET /training/progress`.

---

## 6. Устройства (`devicesView`)

**View ID:** `devicesView`
**Заголовок:** `Устройства`

### 6.1. Подключённые устройства (`connectedDevicesList`)

Список карточек:

- Иконка устройства (emoji или SVG)
- Название: `Apple Watch`, `Samsung Galaxy Watch`, `Huawei Watch D2`, `Amazfit T-Rex 3`
- Статус: «Подключено» / «Отключено»
- Кнопка «Отключить»

Эндпоинты:

- `POST /devices/register` — регистрация устройства.
- `POST /devices/{id}/ingest` — приём данных.
- Управление состоянием через локальное состояние приложения.

### 6.2. Выбор устройства (`deviceSelector`)

Сетка доступных устройств для подключения. При выборе происходит регистрация и переключение статуса.

---

## 7. Достижения (`achievementsView`)

**View ID:** `achievementsView`
**Заголовок:** `Достижения`

### 7.1. Достижения (`achievementsList`)

Сетка карточек (`achievements-grid`):

- Первая тренировка
- 7 дней подряд
- 100 тренировок
- Нормализация пульса
- и т.д.

### 7.2. Соревнования (`competitionsList`)

- Список активных челленджей.
- Позиция в рейтинге.
- Призы.

Источник данных: `GET /training/progress` + бизнес-логика на клиенте.

---

## 8. Диета (`dietView`)

**View ID:** `dietView`
**Заголовок:** `Диета`

### 8.1. План питания (`dietPlanContainer`)

Карточки приёмов пищи:

- Завтрак / Обед / Ужин / Перекусы
- Калории, белки/жиры/углеводы.
- Ингредиенты/примеры блюд.

Источник: `POST /ml/generate-plan` → диетная часть ответа.

---

## 9. ML-анализ (`mlView`)

**View ID:** `mlView`

### 9.1. Классификация состояния

- Показывает текущие параметры: пульс, HRV, SpO₂, температура, давление.
- Кнопка «Классифицировать» → `POST /ml/classify`.
- Результат: класс состояния, уверенность, рекомендация.

### 9.2. Генерация плана

См. раздел 5.2 (модалка с формой параметров).

Результат:

- План тренировок (JSON) с днями, упражнениями, подходами.
- Диетический план (калории, БЖУ, приёмы пищи).

---

## 10. Обработка ошибок

### 10.1. Кастомные страницы ошибок

Все ошибки отображаются через `web/static/errors/`:

- `403.html` — «Доступ запрещён»
- `404.html` — «Страница не найдена»
- `500.html` — «Внутренняя ошибка»
- `error.html` / `error-500.html` — шаблоны для серверных ошибок.

### 10.2. Сетеые ошибки в SPA

- Таймаут `fetch` — 10 секунд.
- При 401 — переход на экран авторизации.
- При 403 — показ `403.html` (сервер возвращает 403 как 404 для предотвращения перечисления ресурсов; в SPA показ 403.html используется для внутренних проверок прав).
- При 5xx — показ `500.html` + retry-кнопка.

---

## 11. Безопасность интерфейса

|#|Мера|Реализация|
|---|---|---|
|1|XSS-защита|Везде `textContent` вместо `innerHTML` для пользовательских данных.|
|2|CSP|`Content-Security-Policy` через NGINX + мета-тег в `index.html`.|
|3|HTTPS-only|Все запросы идут на `https://` (HSTS).|
|4|JWT в `httpOnly` cookie|Хранится в cookie с флагами `HttpOnly`, `Secure`, `SameSite=Strict`. При logout — удаление cookie сервером.|
|5|Валидация на клиенте|Все поля имеют `type`, `min`, `max`, `pattern`, `required`.|
|6|Подпись ответов|`api.js` проверяет HMAC-подпись критических ответов (опционально).|
|7|Rate limit UI|При 429 — показ сообщения «Слишком много запросов, попробуйте через минуту».|

---

## 12. API-интеграция

Все запросы централизованы в `web/static/js/api.js`:

|Функция|Метод|Путь|
|---|---|---|
|`register(name, email, password)`|POST|`/api/v1/register`|
|`registerWithInvite(code, name, email, password)`|POST|`/api/v1/register/invite`|
|`login(email, password)`|POST|`/api/v1/login`|
|`confirmEmail(token)`|POST|`/api/v1/auth/confirm`|
|`logout()`|POST|`/logout`|
|`getProfile()`|GET|`/profile`|
|`updateProfile(data)`|PUT|`/profile`|
|`addBiometric(type, value)`|POST|`/biometrics`|
|`getBiometrics(type)`|GET|`/biometrics`|
|`generatePlan(params)`|POST|`/training/generate`|
|`getPlans()`|GET|`/training/plans`|
|`completeTraining(planId)`|POST|`/training/complete`|
|`getProgress()`|GET|`/training/progress`|
|`classifyState()`|POST|`/ml/classify`|
|`registerDevice(type)`|POST|`/devices/register`|
|`ingestDevice(id, data)`|POST|`/devices/{id}/ingest`|

Все ответы — JSON. Ошибки имеют формат `{error: string}`.

---

## 13. Список страниц и views

|View / Template|URL-путь (JH)|Назначение|
|---|---|---|
|`index.html` (auth)|`/`|Логин / регистрация / верификация|
|`dashboard.html`|—|Обзор (SPA view)|
|`profile.html`|—|Профиль (SPA view)|
|`training.html`|—|Тренировки (SPA view)|
|`achievements.html`|—|Достижения (SPA view)|
|`ml-classify.html`|—|ML классификация|
|`ml-generate.html`|—|Генерация плана|
|`404.html`|—|Сustom 404|
|`403.html`|—|Custom 403|
|`500.html`|—|Custom 500|

---

## 14. Дизайн-токены (CSS-переменные)

Основные переменные из `main.css`:

```css
:root {
  --bg-primary: #0a0a0a;
  --bg-surface: #1a1a1a;
  --bg-elevated: #2a2a2a;
  --text-primary: #ffffff;
  --text-secondary: #a0a0a0;
  --accent: #4f46e5;
  --accent-hover: #6366f1;
  --red: #ef4444;
  --green: #10b981;
  --yellow: #f59e0b;
  --radius-sm: 8px;
  --radius-md: 12px;
  --radius-lg: 20px;
  --font-mono: 'SF Mono', 'Fira Code', monospace;
}
```

Тёмная тема (dark-mode-only). Контраст WCAG AA.

---

## 15. Приоритеты реализации

1. **P0** — Auth (login/register/confirm), Dashboard (биометрия + график), Profile (форма + смена пароля).
2. **P1** — Training: список планов + генерация + завершение.
3. **P2** — Devices: подключение.
4. **P3** — Achievements, Diet, ML-классификация.
5. **UX/Polish** — скелетон-экраны, pull-to-refresh, offline-индикатор, skeleton loaders.
