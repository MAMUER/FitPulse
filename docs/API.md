# FitPulse — API Reference

> Полная спецификация REST/gRPC endpoints. Base URL: `https://fittpulse.duckdns.org:8443/` (development). Production: `https://fitpulse.example.com` (платный домен).

## Аутентификация

Все защищённые запросы требуют JWT access token (ES256, ECDSA P-256) в заголовке:

```text
Authorization: Bearer <access_token>
```

- **access_token TTL**: 15 минут
- **refresh_token TTL**: 7 дней (Absolute Timeout, после требуется повторный login). Реализована rotation (один раз на использование) и reuse detection (инвалидация всей сессии при попытке повторного использования отозванного токена).
- **JWKS endpoint**: `GET /.well-known/jwks.json` (для публичного ключа)

Refresh token используется для ротации через `POST /api/v1/auth/refresh`.

## Публичные endpoints

|Метод|Путь|Описание|Входные данные|Выходные данные|
|---|---|---|---|---|
|POST|`/api/v1/register`|Регистрация пользователя|`{email, password, full_name, role}`|`{status, message?}`|
|POST|`/api/v1/register/invite`|Регистрация через invite-код|`{email, password, full_name, invite_code}`|`{status, message?}`|
|POST|`/api/v1/invite/validate`|Валидация invite-кода|`{code}`|`{is_valid, role, specialty, error_message}`|
|POST|`/api/v1/login`|Вход|`{email, password}`|`{status, access_token?, token_type, expires_in, requires_2fa?, temp_token?, refresh_token?, user_id?, role?}`|
|POST|`/api/v1/auth/confirm`|Подтверждение email|`{token}`|`{status, message}`|
|GET|`/api/v1/auth/verify-status`|Проверка статуса подтверждения email|Query: `?email=`|`{email_confirmed, email}`|
|POST|`/api/v1/auth/refresh`|Ротация refresh token|`{refresh_token}`|`{status, access_token, refresh_token, token_type, expires_in}`|
|POST|`/api/v1/auth/2fa/verify`|Проверка TOTP после логина|`{temp_token, passcode, is_backup_code?}`|`{status, access_token, refresh_token, token_type, expires_in, backup_codes_remaining?}`|
|GET|`/api/v1/auth/google`|Google OAuth логин|—|Redirect to Google|
|GET|`/api/v1/auth/google/callback`|Google OAuth callback|—|`{status, access_token?, user_id?, role?}`|
|POST|`/api/v1/devices/withings/webhook`|Webhook для Withings (публичный)|`{signature, body}`|`{status}`|
|GET|`/health`|Health check|—|`200 OK`|
|GET|`/confirm`|Страница подтверждения email (рендерится `web/templates/confirm.html`)|Query: `?token=`|HTML|

## Защищённые endpoints (JWT required)

|Метод|Путь|Описание|Входные данные|Выходные данные|
|---|---|---|---|---|
|POST|`/logout`|Выход с инвалидацией сессии|—|`{status}`|
|GET|`/profile`|Получить профиль|—|`{status, profile}`|
|PUT|`/profile`|Обновить профиль|`{full_name?, age?, gender?, height_cm?, weight_kg?, fitness_level?, goals?, contraindications?, nutrition?, sleep_hours?}`|`{status}`|
|DELETE|`/profile`|Удалить профиль|`{password}`|`{status, message}`|
|POST|`/biometrics`|Добавить биометрию|`{metric_type, value, timestamp, device_type?}`|`{status}` (201)|
|GET|`/biometrics`|Получить биометрию|Query: `?metric_type=&from=&to=&limit=`|`{status, records: [{type, value, timestamp, device_type}]}`|
|GET|`/training/plans`|Список планов|Query: `?page=&page_size=`|`{status, plans: [{plan_id, plan_data, status, duration_weeks, training_goal, created_at}], total, page, page_size}`|
|GET|`/training/plans/{plan_id}`|Получить план по ID|—|`{status, plan_id, plan_data}`|
|POST|`/training/generate`|Сгенерировать план|`{duration_weeks, available_days, class?, confidence?}`|`{status, plan_id, plan_data, training_type}`|
|POST|`/training/complete`|Завершить тренировку|`{plan_id, workout_id, rating?, feedback?}`|`{status}`|
|GET|`/training/progress`|Прогресс|—|`{status, progress_data}`|
|POST|`/ml/classify`|Классификация состояния|`{biometrics: {hr, hrv, spo2, temp, bp}}`|`{status, state, confidence, recommendation, fatigue_level?, motivation_score?, recovery_quality?}`|
|POST|`/ml/generate-plan`|Генерация плана (GAN)|`{training_class, user_profile, goal?, constraints?}`|`{status, training_plan, diet_plan}`|
|POST|`/devices/register`|Регистрация устройства|`{device_type, device_name?}`|`{status, device_id, device_type, device_name, is_connected, last_sync}`|
|POST|`/devices/{device_id}/ingest`|Приём данных с устройства|`{metrics: [{metric_type, value, timestamp, device_type}]}`|`{status, synced_samples}`|
|GET|`/devices`|Список устройств|—|`{status, devices: [{id, user_id, device_type, created_at}]}`|
|POST|`/devices`|Зарегистрировать новое устройство|`{device_type}`|`{status, device_id, device_type, device_name, is_connected, last_sync}`|
|GET|`/devices/fitbit/auth`|Fitbit OAuth|—|Redirect to Fitbit|
|GET|`/devices/fitbit/callback`|Fitbit callback|—|`{status}`|
|POST|`/devices/fitbit/disconnect`|Disconnect Fitbit|—|`{status}`|
|GET|`/devices/providers`|List providers|—|`{status, providers}`|
|GET|`/devices/withings/auth`|Withings OAuth|—|Redirect to Withings|
|GET|`/devices/withings/callback`|Withings callback|—|`{status}`|
|POST|`/devices/withings/disconnect`|Disconnect Withings|—|`{status}`|
|POST|`/auth/2fa/setup`|Настройка TOTP|—|`{status, qr_code_url, qr_code_base64, secret, backup_codes}`|
|POST|`/auth/2fa/confirm`|Подтверждение TOTP|`{passcode, temp_secret?, backup_codes?}`|`{status, message}`|
|GET|`/auth/2fa/status`|Статус TOTP|—|`{enabled, backup_codes_remaining}`|
|POST|`/auth/2fa/disable`|Отключение TOTP|`{passcode}`|`{status, message}`|

## Админ endpoints (JWT + role=admin)

|Метод|Путь|Описание|Входные данные|Выходные данные|
|---|---|---|---|---|
|GET|`/users`|Список пользователей|Query: `?page=&page_size=`|`{status, users: [UserProfile], total, page, page_size}`|
|GET|`/invites`|Список invite-кодов|Query: `?page=&page_size=&used=`|`{status, invites: [{code, role, specialty, used, created_at, max_uses?}], total, page, page_size}`|
|POST|`/invites`|Создать invite-код|`{role, specialty?, max_uses?}`|`{status, code, role, specialty, max_uses, created_at}`|
|POST|`/invites/{code}/revoke`|Отозвать invite-код|—|`{status, message}`|

## gRPC services

|Service|Порт|Описание|
|---|---|---|
|User Service|50051|Регистрация, логин, профили, email-верификация, invite-коды|
|Biometric Service|50052|Приём и хранение биометрических данных|
|Training Service|50053|Управление тренировочными планами|

## Модели данных

### UserProfile

```json
{
  "user_id": "uuid",
  "email": "string",
  "full_name": "string",
  "nickname": "string",
  "role": "client|admin",
  "email_confirmed": true,
  "age": 25,
  "gender": "male|female",
  "height_cm": 180,
  "weight_kg": 75.5,
  "fitness_level": "beginner|intermediate|advanced",
  "goals": ["weight_loss", "muscle_gain"],
  "contraindications": [],
  "nutrition": "balanced",
  "sleep_hours": 7.5,
  "profile_photo_url": "url",
  "created_at": "ISO8601",
  "updated_at": "ISO8601"
}
```

### BiomRecord

```json
{
  "type": "heart_rate|spo2|temperature|ecg|blood_pressure|sleep|steps|hrv",
  "value": 72,
  "timestamp": "ISO8601",
  "device_type": "string"
}
```

### TrainingPlan

```json
{
  "plan_id": "uuid",
  "plan_data": {},
  "status": "active|completed|archived",
  "duration_weeks": 4,
  "training_goal": "endurance_e1e2",
  "created_at": "ISO8601"
}
```

### LoginResponse

```json
{
  "status": "ok",
  "access_token": "JWT (ES256, 15min)",
  "refresh_token": "opaque string (7d)",
  "token_type": "Bearer",
  "expires_in": 900,
  "user_id": "uuid",
  "role": "client|admin"
}
```

### Error response

Все ошибки возвращаются в формате:

```json
{
  "error": "Описание ошибки"
}
```

|HTTP код|Описание|
|---|---|
|400|Некорректный запрос|
|401|Unauthorized (неверный/истёкший токен)|
|403|Forbidden (нет прав) — возвращается как 404|
|404|Not Found|
|429|Rate limit exceeded|
|500|Внутренняя ошибка|
|503|Сервис временно недоступен|
