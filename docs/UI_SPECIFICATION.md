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
|Здоровье|Heart pulse|`health`|`healthView`|Здоровье|—|
|Админка|Shield|`admin`|`adminView`|Админка|Скрыт по умолчанию (`display: none`)|

Дополнительные view без tab-bar:

- `mlView` (AI-анализ) — открывается как модалка/оверлей или отдельный экран;

### 1.3. Технологии

- **HTML5**, vanilla JS (ES2026), CSS-переменные.
- **Chart.js** (`chart.umd.min.js`) — график пульса на Dashboard.
- **Fetch API** через `web/static/js/api.js` для всех запросов к Gateway.
- JWT хранится в `localStorage` и отправляется в заголовке `Authorization: Bearer`. Refresh token хранится в `session` cookie.

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
        div#healthView
        div#mlView (оверлей, не в tab-bar)
        div#adminView (скрыт display:none для не-админов)
    nav.tab-bar
      button.tab × 8 (admin скрыт)
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
- `deleteProfileBtn` → `confirm()` диалог с запросом пароля → `DELETE /profile` с `{password}` в теле запроса.

### 4.4. Эндпоинты

|Действие|Метод|Путь|
|---|---|---|
|Загрузить профиль|GET|`/profile`|
|Обновить профиль|PUT|`/profile`|
|Удалить профиль|DELETE|`/profile`|

Смена пароля и email выполняются через `PUT /profile` с соответствующими полями, отдельные endpoints `/profile/security/*` не используются.

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

- График прогресса (Chart.js, столбчатая диаграмма) — `GET /training/progress`. Отображается в разделе «Достижения» (`achievementsView`) под списком челленджей.

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

Сетка карточек (`achievements-grid`). Данные загружаются через `GET /api/v1/achievements` и отображаются все достижения из БД с статусом получено/заблокировано:

- Первый шаг — первая завершённая тренировка
- Десятка — 10 завершённых тренировок
- Полтинник — 50 завершённых тренировок
- Сто дней — 100 дней активности
- Мастер спорта — 1000 завершённых тренировок

Эндпоинт: `GET /api/v1/achievements`.

### 7.2. Соревнования (`competitionsList`)

- Список активных челленджей.
- Позиция в рейтинге.
- Призы.

Источник данных: клиентские заглушки (персональные челленджи).

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

## 8.5. Здоровье (`healthView`)

**View ID:** `healthView`
**Заголовок:** `Здоровье`

### 8.5.1. Особенности здоровья (`healthConditionsList`)

- Список заболеваний/состояний.
- Кнопка «Добавить» → `POST /health/conditions`.
- Эндпоинты: `GET /health/conditions`, `POST /health/conditions`, `DELETE /health/conditions/{condition_id}`.

### 8.5.2. Состав тела (`bodyCompositionList`)

- Журнал записей состава тела.
- Кнопка «Добавить запись» → `POST /health/body-composition`.
- Эндпоинты: `GET /health/body-composition`, `POST /health/body-composition`.

### 8.5.3. Женский цикл (`menstrualCyclesList`)

- Календарь/список циклов.
- Кнопка «Добавить цикл» → `POST /health/menstrual-cycles`.
- Эндпоинты: `GET /health/menstrual-cycles`, `POST /health/menstrual-cycles`, `PUT /health/menstrual-cycles/{cycle_id}`, `DELETE /health/menstrual-cycles/{cycle_id}`.

### 8.5.4. Синхронизация

- `syncFloBtn` → `POST /health/sync/flo`
- `syncOKOKBtn` → `POST /health/sync/okok`

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

### 10.3. Список пользователей

- Бэкенд эндпоинт `GET /api/v1/admin/users` реализован.
- UI в `adminView` отображает список пользователей в виде сетки: имя/email, роль, дата создания, дата обновления.

---

## 11. Обработка ошибок

### 11.1. Кастомные страницы ошибок

Серверные ошибки отображаются через `web/static/errors/`:

- `error.html` — общий шаблон для 404 «Страница не найдена»
- `error-500.html` — шаблон для 500 «Внутренняя ошибка»
- `403.html` — «Доступ запрещён» (уникальный, не дублирует error.html)

### 11.2. Страница подтверждения email (`/confirm`)

- Серверный шаблон `web/templates/confirm.html` рендерится по `GET /confirm?token=`.
- Fallback HTML встроен в `handlers_auth.go` при отсутствии шаблона.

### 11.3. Сетевые ошибки в SPA

- Таймаут `fetch` — 10 секунд (AbortController + setTimeout).
- При 403 — для HTML запросов показ `403.html`; для JSON API сервер возвращает 403 как 404 для предотвращения перечисления ресурсов.
- При 401 — попытка refresh токена → если fail → logout.
- При 429 — показ сообщения «Слишком много запросов, попробуйте через минуту» (или с учетом заголовка `Retry-After`).
- При 5xx — показ `error-500.html` + retry-кнопка.

---

## 12. Безопасность интерфейса

|#|Мера|Реализация|
|---|---|---|
|1|XSS-защита|В большинстве мест используется `innerHTML` для рендеринга данных. Для пользовательского ввода применяется `textContent` там, где это реализовано.|
|2|CSP|`Content-Security-Policy` генерируется серверным middleware `SecurityHeaders` (nonce-based) + `HTMLNonceInject` добавляет `nonce` в `<script>` теги.|
|3|HTTPS|Поддерживается TLS 1.3 + HSTS. Клиент использует относительные URLs (`/api/v1/...`), поэтому работает и по HTTP, и по HTTPS в зависимости от окружения.|
|4|JWT хранение|Access token хранится в `localStorage`, отправляется в заголовке `Authorization: Bearer`. Refresh token хранится в `session` cookie.|
|5|Валидация на клиенте|Все поля имеют `type`, `min`, `max`, `pattern`, `required`.|
|6|Защита ответов|Транспорт защищён TLS 1.3 + HSTS + CSP|
|7|Rate limit UI|При 429 показывается сообщение «Слишком много запросов, попробуйте через минуту» (или с учетом `Retry-After`).|

---

## 13. API-интеграция

Все запросы централизованы в `web/static/js/api.js`. Базовый путь: `/api/v1`.

|Функция|Метод|Путь|
|---|---|---|
|`register(email, password, fullName, role)`|POST|`/api/v1/register`|
|`login(email, password)`|POST|`/api/v1/login`|
|`getProfile()`|GET|`/api/v1/profile`|
|`updateProfile(profile)`|PUT|`/api/v1/profile`|
|`changePassword(currentPassword, newPassword)`|POST|`/api/v1/auth/change-password`|
|`changeEmail(newEmail, password)`|POST|`/api/v1/auth/change-email`|
|`get2FAStatus()`|GET|`/api/v1/auth/2fa/status`|
|`setup2FA()`|POST|`/api/v1/auth/2fa/setup`|
|`confirm2FA(passcode, tempSecret, backupCodes)`|POST|`/api/v1/auth/2fa/confirm`|
|`verify2FA(tempToken, passcode, isBackupCode)`|POST|`/api/v1/auth/2fa/verify`|
|`disable2FA(passcode)`|POST|`/api/v1/auth/2fa/disable`|
|`deleteProfile(password)`|DELETE|`/api/v1/profile`|
|`addBiometricRecord(metricType, value, timestamp, deviceType)`|POST|`/api/v1/biometrics`|
|`getBiometricRecords(metricType, from, to, limit)`|GET|`/api/v1/biometrics`|
|`generateTrainingPlan(durationWeeks, availableDays, classificationClass, confidence)`|POST|`/api/v1/training/generate`|
|`getTrainingPlans(page, pageSize)`|GET|`/api/v1/training/plans`|
|`getPlan(planId)`|GET|`/api/v1/training/plans/{plan_id}`|
|`completeWorkout(planId, workoutId, rating, feedback)`|POST|`/api/v1/training/complete`|
|`getProgress()`|GET|`/api/v1/training/progress`|
|`getAchievements()`|GET|`/api/v1/achievements`|
|`logout()`|POST|`/api/v1/logout`|
|`listHealthConditions(conditionType)`|GET|`/api/v1/health/conditions`|
|`upsertHealthCondition(data)`|POST|`/api/v1/health/conditions`|
|`deleteHealthCondition(conditionId)`|DELETE|`/api/v1/health/conditions/{conditionId}`|
|`listBodyComposition(from, to, limit)`|GET|`/api/v1/health/body-composition`|
|`createBodyComposition(data)`|POST|`/api/v1/health/body-composition`|
|`listMenstrualCycles()`|GET|`/api/v1/health/menstrual-cycles`|
|`createMenstrualCycle(data)`|POST|`/api/v1/health/menstrual-cycles`|
|`updateMenstrualCycle(cycleId, data)`|PUT|`/api/v1/health/menstrual-cycles/{cycleId}`|
|`deleteMenstrualCycle(cycleId)`|DELETE|`/api/v1/health/menstrual-cycles/{cycleId}`|
|`syncFlo(accessToken, refreshToken)`|POST|`/api/v1/health/sync/flo`|
|`syncOKOK(accessToken, refreshToken)`|POST|`/api/v1/health/sync/okok`|
|`fitbitAuth()`|GET|`/api/v1/devices/fitbit/auth`|
|`withingsAuth()`|GET|`/api/v1/devices/withings/auth`|
|`getProviders()`|GET|`/api/v1/devices/providers`|
|`classifyState(biometrics)`|POST|`/api/v1/ml/classify`|
|`generateMLPlan(trainingClass, user_profile, goal, constraints)`|POST|`/api/v1/ml/generate-plan`|
|`registerWithInvite(code, name, email, password)`|POST|`/api/v1/register/invite`|
|`validateInvite(code)`|POST|`/api/v1/invite/validate`|
|`createInvite(role, specialty, maxUses)`|POST|`/api/v1/admin/invites`|
|`listInvites(page, pageSize, used)`|GET|`/api/v1/admin/invites`|
|`revokeInvite(code)`|POST|`/api/v1/admin/invites/{code}/revoke`|
|`listUsers(page, pageSize)`|GET|`/api/v1/admin/users`|

Все ответы — JSON. Ошибки имеют формат `{error: string}` или `{message: string}`.

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
  --bg-primary: #000000;
  --bg-secondary: #1c1c1e;
  --bg-card: #2c2c2e;
  --bg-input: #3a3a3c;
  --text-primary: #ffffff;
  --text-secondary: #8e8e93;
  --text-tertiary: #636366;
  --accent: #ff375f;
  --accent-secondary: #ff6b81;
  --green: #30d158;
  --blue: #0a84ff;
  --orange: #ff9f0a;
  --purple: #bf5af2;
  --teal: #64d2ff;
  --radius-sm: 12px;
  --radius-md: 16px;
  --radius-lg: 20px;
  --radius-xl: 24px;
  --safe-top: env(safe-area-inset-top, 0px);
  --safe-bottom: env(safe-area-inset-bottom, 0px);
  --font-mono: 'JetBrains Mono', 'Fira Code', monospace;
  --font-body: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
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
5. **P4** — Admin panel: invite-коды (создание, список, отзыв), список пользователей.
6. **UX/Polish** — скелетон-экраны, pull-to-refresh, offline-индикатор, skeleton loaders.
