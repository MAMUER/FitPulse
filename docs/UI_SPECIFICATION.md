# FitPulse — UI Specification

> **Scope:** Мобильное веб-приложение (SPA) для пользователя системы FitPulse.
> **Target device:** Мобильный браузер (Viewport `390–430 px`, touch-first).
> **Base URL:** `https://fittpulse.duckdns.org:8443/` (development).

---

## 1. Архитектура интерфейса

### 1.1. Тип приложения

Single Page Application (SPA), состоящая из одного HTML-файла `web/index.html` с переключением `view` по классу `active`. Дополнительно используется серверный шаблон `web/templates/confirm.html` для страницы подтверждения email по маршруту `GET /confirm`.

### 1.2. Навигация

Нижний tab-bar с 7 кнопками (6 для всех пользователей + 1 скрытая для админов). Переключение происходит мгновенно без перезагрузки страницы.

|Tab|Icon (SVG)|data-view|View ID|Заголовок|Примечание|
|---|---|---|---|---|---|
|Обзор|4-quadrant grid|`dashboard`|`dashboardView`|Обзор|Активен по умолчанию|
|Профиль|User circle|`profile`|`profileView`|Профиль|—|
|Тренировки|Lightning bolt|`training`|`trainingView`|Тренировки|—|
|Устройства|Smartphone|`devices`|`devicesView`|Устройства|—|
|Достижения|Trophy|`achievements`|`achievementsView`|Достижения|—|
|Диета|Container/bottle|`diet`|`dietView`|Диета|—|
|Админка|Shield|`admin`|`adminView`|Админка|Скрыт по умолчанию (`display: none`)|

Дополнительные view без tab-bar:

- `mlView` (AI-анализ) — открывается как модалка/оверлей или отдельный экран;

### 1.3. Технологии

- **HTML5**, vanilla JS (ES2026), CSS-переменные.
- **Chart.js** (`chart.umd.min.js`) — график пульса на Dashboard.
- **Fetch API** через `web/static/js/api.js` для всех запросов к Gateway.
- JWT хранится в `httpOnly` cookie (подробности в разделе 12).

---

## 2. Экраны и состояние

### 2.1. Экран авторизации (`authScreen`)

Состояния (отображаются по классу `active` / `hidden`):

|Состояние|Форма / элемент|ID|Описание|
|---|---|---|---|
|Логин|`loginForm`|`loginForm`|Email + пароль|
|Логин с 2FA|`login2FAForm`|`login2FAForm`|Ввод TOTP-кода или резервного кода|
|Регистрация|`registerForm`|`registerForm`|Имя, email, пароль|
|Верификация email|`verifyForm`|`verifyForm`|Подтверждение токена|
|Ошибка|`authError` / `loginError` / `login2FAError` / `registerError` / `confirmError`|`authError`, `loginError`, `login2FAError`, `registerError`, `confirmError`|Общий блок ошибок|

**Поля формы логина:**

- `loginEmail` — `type=email`, `inputmode=email`, `maxlength=254`, `autocomplete=email`.
- `loginPassword` — `type=password`, `autocomplete=current-password`.
- `loginBtn` — кнопка отправки.

**Поля формы логина с 2FA:**

- `totpLoginCode` — `type=text`, `maxlength=6`, `inputmode=numeric`, `autocomplete=one-time-code`.
- `backupLoginCode` — `type=text`, `maxlength=9`, `autocomplete=off`.
- `login2FABtn` — кнопка отправки.
- `useBackupLoginBtn` — переключение на резервный код.
- `backToLoginFrom2FA` — возврат к обычному логину.

**Поля формы регистрации:**

- `regName` — `type=text`, `maxlength=100`, `minlength=2`, `pattern=[A-Za-zА-Яа-яЁё\s\-]+`.
- `regEmail` — email, `maxlength=254`.
- `regPassword` — `type=password`, `minlength=8`, с подсказкой `passwordHint` (длина, заглавная, строчная, цифра).
- `registerBtn` — кнопка отправки (изначально `disabled`).

**Валидация:**

- `regPassword` в реальном времени обновляет `passwordHint` через `hintLength`, `hintUpper`, `hintLower`, `hintDigit`.
- Кнопка `registerBtn` заблокирована (`disabled`), до выполнения всех требований к паролю.
- При сабмите формы запрос уходит в `api.register()` → `POST /api/v1/register`.

**Верификация email:**

- После успешной регистрации показывается `verifyForm` с email-адресом (`verifyEmail`).
- Если нет SMTP / в режиме разработки — показывается секция `devTokenSection` с токеном (`devToken`, `verifyToken`).
- Для production: пользователь переходит по ссылке из письма → токен передаётся в `POST /api/v1/auth/confirm`.
- `confirmBtn` — кнопка подтверждения токена.
- `backToLogin` — возврат к логину.

**Переходы:**

- `toRegister` — из логина в регистрацию.
- `toLogin` — из регистрации в логин.
- `backToLogin` — из верификации в логин.
- `backToLoginFrom2FA` — из 2FA в обычный логин.

### 2.2. Главное приложение (`mainScreen`)

Структура:

```text
div#app
  div#authScreen.screen (скрыт после авторизации)
  div#mainScreen.screen
    header.top-bar
      h2#pageTitle
      button#logoutBtn
    main.content
      section.view (один активный)
        div#dashboardView
        div#profileView
        div#trainingView
        div#devicesView
        div#achievementsView
        div#dietView
        div#mlView (оверлей, не в tab-bar)
        div#adminView (скрыт display:none для не-админов)
    nav.tab-bar
      button.tab × 7 (admin скрыт)
```

**Логика переключения view:**

- По клику на tab: удалить `active` у текущего view, добавить текущему, обновить `pageTitle`.
- При открытии view вызывается соответствующий `init*()` из `app.js`.
- `adminTab` отображается только для пользователей с ролью `admin`.

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
- Данные: массив `{timestamp, value}` из `GET /biometrics?metric_type=heart_rate`.

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

**Здоровье и предпочтения:**

- Аллергии (`profAllergies`) — текстовое поле.
- Медицинские противопоказания (`profContraindications`) — текстовое поле.
- ИМТ калькулятор: `bmiHint`, `bmiValue`, `bmiCategory`, `bmiRecommendation`.

**Цели** — radio-chips (`name="goal"`):

- `weight_loss` — Похудение
- `muscle_gain` — Набор мышц
- `endurance` — Выносливость
- `flexibility` — Гибкость

### 4.2. Безопасность

Кнопки:

- `changePasswordBtn` — открывает форму смены пароля (`changePasswordForm`).
- `changeEmailBtn` — открывает форму смены email (`changeEmailForm`).

**Форма смены пароля (`changePasswordForm`):**

- `currentPassword` + `currentPasswordError`
- `newPassword` + `newPasswordError` + `passwordHint` (с `hintLength`, `hintUpper`, `hintLower`, `hintDigit`)
- `confirmPassword` + `confirmPasswordError`
- Кнопки: `cancelChangePassword`, submit → `PUT /profile`

**Форма смены email (`changeEmailForm`):**

- `newEmail` + `newEmailError`
- `emailConfirmPassword` + `emailConfirmPasswordError`
- Кнопки: `cancelChangeEmail`, submit → `PUT /profile`

**Двухфакторная аутентификация:**

- `twoFAStatus` — статус 2FA.
- `enable2FABtn` / `disable2FABtn` — кнопки включения/отключения.
- `totpSetupPanel` — панель настройки (QR-код `totpQRCode`, секрет `totpManualSecret`, резервные коды `totpBackupCodes`, код подтверждения `totpSetupCode` + `totpSetupError`).
- `confirm2FABtn` — подтверждение включения 2FA.
- `disable2FAPanel` — панель отключения (`disable2FACode` + `disable2FAError`).
- Endpoints: `POST /auth/2fa/setup`, `POST /auth/2fa/confirm`, `GET /auth/2fa/status`, `POST /auth/2fa/disable`.

### 4.3. Опасная зона (`danger-zone`)

- Заголовок красным: «Опасная зона».
- `deleteProfileBtn` → модалка подтверждения через пароль → `DELETE /profile`.

### 4.4. Эндпоинты

|Действие|Метод|Путь|
|---|---|---|
|Загрузить профиль|GET|`/profile`|
|Обновить профиль|PUT|`/profile`|
|Удалить профиль|DELETE|`/profile`|

Примечание: смена пароля и email выполняются через `PUT /profile` с соответствующими полями, отдельные endpoints `/profile/security/*` не используются.

---

## 5. Тренировки (`trainingView`)

**View ID:** `trainingView`
**Заголовок:** `Тренировки`

### 5.1. Список планов (`plansList`)

- Карточки планов: название, дата создания, статус (`active` / `completed` / `archived`).
- Пустое состояние: `empty-state` с иконкой `🏃` и текстом «AI создаст персональный план...».
- Эндпоинт: `GET /training/plans?page=&page_size=`.

### 5.2. Кнопка генерации

- `generatePlanBtn` — плавающая кнопка (FAB) внизу экрана.
- При нажатии открывается **модалка / отдельный экран** с формой параметров:
  - Длительность (недели), `min=1`, `max=12`
  - Доступные дни (чекбоксы Пн–Вс)
  - Класс тренировки (select)
  - Уверенность/интенсивность
- Submit → `POST /training/generate`.

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

- `GET /devices` — список устройств.
- `POST /devices` — регистрация устройства.
- `POST /devices/{id}/ingest` — приём данных.
- Управление состоянием через локальное состояние приложения.

### 6.2. Выбор устройства (`deviceSelector`)

Сетка доступных устройств для подключения. При выборе происходит регистрация и переключение статуса.

### 6.3. Интеграции

- Fitbit OAuth: `GET /devices/fitbit/auth`, `GET /devices/fitbit/callback`, `POST /devices/fitbit/disconnect`.
- Withings OAuth: `GET /devices/withings/auth`, `GET /devices/withings/callback`, `POST /devices/withings/disconnect`, `POST /api/v1/devices/withings/webhook`.
- Список провайдеров: `GET /devices/providers`.

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

### 8.1. Настройки питания (`dietSettings`)

- `dietAllergies` — аллергии/непереносимость.
- `dietDislikes` — нелюбимые продукты.
- `dietMealsCount` — количество приёмов пищи (3–6).
- `dietFirstMealTime` — время первого приёма пищи.
- `applyDietSettingsBtn` — применить и обновить план.

### 8.2. План питания (`dietPlanContainer`)

Карточки приёмов пищи:

- Завтрак / Обед / Ужин / Перекусы
- Калории, белки/жиры/углеводы.
- Ингредиенты/примеры блюд.

Источник: `POST /ml/generate-plan` → диетная часть ответа.

---

## 9. ML-анализ (`mlView`)

**View ID:** `mlView` (оверлей/экран, не в нижнем tab-bar)

### 9.1. Классификация состояния

- `mlResult` — контейнер результата.
- `mlClassifyBtn` — плавающая кнопка запуска классификации.
- Показывает текущие параметры: пульс, HRV, SpO₂, температура, давление.
- Кнопка «Классифицировать» → `POST /ml/classify`.
- Результат: класс состояния, уверенность, рекомендация.

### 9.2. Генерация плана (в разделе «Тренировки»)

См. раздел 5.2 (кнопка `generatePlanBtn`).

Результат:

- План тренировок (JSON) с днями, упражнениями, подходами.
- Диетический план (калории, БЖУ, приёмы пищи).

---

## 10. Админка (`adminView`)

**View ID:** `adminView`
**Заголовок:** `Панель администратора`
**Примечание:** view скрыта по умолчанию (`display: none`), отображается только для роли `admin`.

### 10.1. Создание invite-кода

- `newInviteRole` — select: `client`, `admin`.
- `newInviteMaxUses` — число использований (min=1).
- Кнопка «Создать» → `POST /invites`.

### 10.2. Список invite-кодов (`invitesList`)

- Карточки кодов: код, роль, статус, использовано/макс.
- Эндпоинт: `GET /invites?page=&page_size=&used=`.

---

## 11. Обработка ошибок

### 11.1. Кастомные страницы ошибок

Серверные ошибки отображаются через `web/static/errors/`:

- `403.html` — «Доступ запрещён»
- `404.html` — «Страница не найдена»
- `500.html` — «Внутренняя ошибка»
- `error.html` / `error-500.html` — шаблоны для серверных ошибок.

### 11.2. Страница подтверждения email (`/confirm`)

- Серверный шаблон `web/templates/confirm.html` рендерится по `GET /confirm?token=`.
- Fallback HTML встроен в `handlers_auth.go` при отсутствии шаблона.

### 11.3. Сетевые ошибки в SPA

- Таймаут `fetch` — 10 секунд (AbortController + setTimeout).
- При 403 — показ `403.html` (сервер возвращает 403 как 404 для предотвращения перечисления ресурсов; в SPA показ 403.html используется для внутренних проверок прав).
- При 401 — попытка refresh токена → если fail → logout.
- При 5xx — показ `500.html` + retry-кнопка.

---

## 12. Безопасность интерфейса

|#|Мера|Реализация|
|---|---|---|
|1|XSS-защита|Везде `textContent` вместо `innerHTML` для пользовательских данных.|
|2|CSP|`Content-Security-Policy` через NGINX + мета-тег в `index.html`.|
|3|HTTPS-only|Все запросы идут на `https://` (HSTS).|
|4|JWT в `httpOnly` cookie|Хранится в cookie с флагами `HttpOnly`, `Secure`, `SameSite=Strict`. При logout — сервер возвращает `Set-Cookie` с `Max-Age=0` (клиент не может удалить `httpOnly` cookie через JS).|
|5|Валидация на клиенте|Все поля имеют `type`, `min`, `max`, `pattern`, `required`.|
|6|Подпись ответов|Сервер подписывает HMAC-SHA256 критические JSON-ответы (login, register, profile, biometrics, plans). Клиентская верификация в SPA не применяется: симметричный ключ не может безопасно храниться в публичном JS-бандле.|
|7|Rate limit UI|При 429 — показ сообщения «Слишком много запросов, попробуйте через минуту».|

---

## 13. API-интеграция

Все запросы централизованы в `web/static/js/api.js`:

|Функция|Метод|Путь|
|---|---|---|
|`register(name, email, password)`|POST|`/api/v1/register`|
|`registerWithInvite(code, name, email, password)`|POST|`/api/v1/register/invite`|
|`validateInvite(code)`|POST|`/api/v1/invite/validate`|
|`login(email, password)`|POST|`/api/v1/login`|
|`confirmEmail(token)`|POST|`/api/v1/auth/confirm`|
|`checkVerificationStatus(email)`|GET|`/api/v1/auth/verify-status`|
|`refreshToken(refresh_token)`|POST|`/api/v1/auth/refresh`|
|`verify2FA(temp_token, passcode, isBackupCode)`|POST|`/api/v1/auth/2fa/verify`|
|`googleLogin()`|GET|`/api/v1/auth/google`|
|`logout()`|POST|`/logout`|
|`getProfile()`|GET|`/profile`|
|`updateProfile(data)`|PUT|`/profile`|
|`deleteProfile(password)`|DELETE|`/profile`|
|`addBiometric(metric_type, value, timestamp, device_type)`|POST|`/biometrics`|
|`getBiometrics(metric_type, from, to, limit)`|GET|`/biometrics`|
|`generatePlan(duration_weeks, available_days, class, confidence)`|POST|`/training/generate`|
|`getPlans(page, page_size)`|GET|`/training/plans`|
|`getPlan(plan_id)`|GET|`/training/plans/{plan_id}`|
|`completeTraining(plan_id, workout_id, rating, feedback)`|POST|`/training/complete`|
|`getProgress()`|GET|`/training/progress`|
|`classifyState(biometrics)`|POST|`/ml/classify`|
|`generateMLPlan(training_class, user_profile, goal, constraints)`|POST|`/ml/generate-plan`|
|`registerDevice(device_type, device_name)`|POST|`/devices/register`|
|`ingestDevice(device_id, metrics)`|POST|`/devices/{device_id}/ingest`|
|`listDevices()`|GET|`/devices`|
|`registerDeviceSimple(device_type)`|POST|`/devices`|
|`fitbitAuth()`|GET|`/devices/fitbit/auth`|
|`withingsAuth()`|GET|`/devices/withings/auth`|
|`getProviders()`|GET|`/devices/providers`|
|`setupTOTP()`|POST|`/auth/2fa/setup`|
|`confirmTOTP(passcode, temp_secret, backup_codes)`|POST|`/auth/2fa/confirm`|
|`getTOTPStatus()`|GET|`/auth/2fa/status`|
|`disableTOTP(passcode)`|POST|`/auth/2fa/disable`|
|`createInvite(role, specialty, max_uses)`|POST|`/invites`|
|`listInvites(page, page_size, used)`|GET|`/invites`|
|`revokeInvite(code)`|POST|`/invites/{code}/revoke`|

Все ответы — JSON. Ошибки имеют формат `{error: string}`.

---

## 14. Фактические файлы проекта

|Путь|Назначение|
|---|---|
|`web/index.html`|SPA: auth + все views (dashboard, profile, training, devices, achievements, diet, ml, admin)|
|`web/templates/confirm.html`|Шаблон страницы подтверждения email (рендерится Go)|
|`web/static/css/main.css`|Основные стили, CSS-переменные|
|`web/static/css/modules.css`|Модульные стили|
|`web/static/fonts/fonts.css`|Self-hosted шрифты (JetBrains Mono, Inter)|
|`web/static/js/api.js`|Функции HTTP-запросов к Gateway|
|`web/static/js/app.js`|Логика SPA: переключение view, инициализация модулей|
|`web/static/js/modules.js`|Модули: устройства, достижения, тренировки, ML|
|`web/static/errors/403.html`|Страница 403|
|`web/static/errors/404.html`|Страница 404|
|`web/static/errors/500.html`|Страница 500|
|`web/static/errors/error.html`|Общий шаблон ошибки|
|`web/static/errors/error-500.html`|Шаблон 500 ошибки|

Примечание: Более половины файлов в `web/templates/` (`achievements.html`, `base.html`, `dashboard.html`, `ml-classify.html`, `ml-generate.html`, `profile.html`, `training.html`) не используются в текущей версии кода. Основной фронтенд — это `web/index.html` (SPA).

---

## 15. Дизайн-токены (CSS-переменные)

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
  --font-mono: 'JetBrains Mono', 'Fira Code', monospace;
}
```

Шрифты self-hosted: `web/static/fonts/fonts.css` подключает локальные `.woff2` JetBrains Mono и Inter. Без внешних запросов к Google Fonts.

Тёмная тема (dark-mode-only). Контраст WCAG AA.

---

## 16. Приоритеты реализации

1. **P0** — Auth (login/register/confirm/2FA), Dashboard (биометрия + график), Profile (форма + смена пароля/email + 2FA).
2. **P1** — Training: список планов + генерация + завершение.
3. **P2** — Devices: подключение + интеграции.
4. **P3** — Achievements, Diet, ML-классификация.
5. **P4** — Admin panel: invite-коды, список пользователей.
6. **UX/Polish** — скелетон-экраны, pull-to-refresh, offline-индикатор, skeleton loaders.
