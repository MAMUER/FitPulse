# FitPulse — API Reference

> Полная спецификация REST/gRPC endpoints. Base URL: `https://fitpulse.duckdns.org:8443/` (development). Production: `https://fitpulse.example.com` (платный домен).

## Аутентификация

Все защищённые запросы требуют JWT access token (ES256, ECDSA P-256) в заголовке:

```text
Authorization: Bearer <access_token>
```

- **access_token TTL**: 15 минут
- **refresh_token TTL**: 7 дней, rotation при каждом использовании
- **JWKS endpoint**: `GET /.well-known/jwks.json` (для публичного ключа)

Refresh token используется для ротации через `POST /api/v1/auth/refresh`.

## Публичные endpoints

|Метод|Путь|Описание|Входные данные|Выходные данные|
|---|---|---|---|---|
|POST|`/api/v1/register`|Регистрация пользователя|`{email, password, full_name, role}`|`{user_id, message}`|
|POST|`/api/v1/register/invite`|Регистрация через invite-код|`{email, password, full_name, invite_code}`|`{user_id, message}`|
|POST|`/api/v1/invite/validate`|Валидация invite-кода|`{code}`|`{is_valid, role, specialty, error_message}`|
|POST|`/api/v1/login`|Вход|`{email, password}`|`{access_token, refresh_token, token_type, expires_in, user_id, role}`|
|POST|`/api/v1/auth/confirm`|Подтверждение email|`{token}`|`{user_id, message}`|
|POST|`/api/v1/auth/refresh`|Ротация refresh token|`{refresh_token}`|`{access_token, refresh_token, token_type, expires_in}`|
|POST|`/api/v1/devices/register`|Регистрация устройства|`{device_type, device_name}`|`{device_id, device_type, device_name, is_connected, last_sync}`|
|POST|`/api/v1/devices/{id}/ingest`|Приём данных с устройства|`{metrics: [{type, value, timestamp}]}`|`{message, synced_samples}`|
|POST|`/api/v1/devices/withings/webhook`|Webhook для Withings (публичный, но с секретным ключом)|`{signature, body}`|`{status}`|
|GET|`/health`|Health check|—|`200 OK`|

## Защищённые endpoints (JWT required)

|Метод|Путь|Описание|Входные данные|Выходные данные|
|---|---|---|---|---|
|POST|`/api/v1/logout`|Выход с инвалидацией сессии|—|`{message}`|
|GET|`/profile`|Получить профиль|—|`UserProfile`|
|PUT|`/profile`|Обновить профиль|`{full_name?, nickname?, age?, gender?, height_cm?, weight_kg?, fitness_level?, goals?, contraindications?, nutrition?, sleep_hours?}`|`UserProfile`|
|DELETE|`/profile`|Удалить профиль|`{password}`|`{message}`|
|POST|`/biometrics`|Добавить биометрию|`{type, value, timestamp, device_id?}`|`{message}`|
|GET|`/biometrics`|Получить биометрию|Query: `?type=&from=&to=`|`[{type, value, timestamp, device_id}]`|
|PUT|`/profile/security/password`|Сменить пароль|`{current_password, new_password}`|`{message}`|
|PUT|`/profile/security/email`|Сменить email|`{new_email, password}`|`{message}`|
|POST|`/training/generate`|Сгенерировать план|`{duration_weeks, max_duration, preferred_time, days, equipment, goal}`|`TrainingPlan`|
|GET|`/training/plans`|Список планов|Query: `?status=`|`[{plan_id, name, status, created_at}]`|
|POST|`/training/complete`|Завершить тренировку|`{plan_id}`|`{message}`|
|GET|`/training/progress`|Прогресс|Query: `?from=&to=`|`{total_workouts, completed_workouts, average_duration_minutes, total_calories_burned}`|
|POST|`/ml/classify`|Классификация состояния|`{biometrics: {hr, hrv, spo2, temp, bp}}`|`{state, confidence, recommendation}`|
|POST|`/ml/generate-plan`|Генерация плана (GAN)|`{user_profile, goal, constraints}`|`{training_plan, diet_plan}`|
|POST|`/auth/2fa/setup`|Настройка TOTP|`{user_id}`|`{qr_code_url, secret, backup_codes}`|
|POST|`/auth/2fa/confirm`|Подтверждение TOTP|`{user_id, passcode, temp_secret, backup_codes}`|`{success, message}`|
|POST|`/auth/2fa/verify`|Проверка TOTP|`{user_id, passcode, is_backup_code?}`|`{valid, backup_codes_remaining}`|
|POST|`/auth/2fa/disable`|Отключение TOTP|`{user_id, passcode}`|`{success, message}`|
|GET|`/auth/2fa/status`|Статус TOTP|—|`{enabled, backup_codes_remaining}`|

## Админ endpoints (JWT + role=admin)

|Метод|Путь|Описание|Входные данные|Выходные данные|
|---|---|---|---|---|
|GET|`/admin/users`|Список пользователей|Query: `?page=&page_size=&role=`|`{users: [UserProfile], total}`|

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
  "device_id": "string"
}
```

### TrainingPlan

```json
{
  "plan_id": "uuid",
  "name": "string",
  "status": "active|completed|archived",
  "created_at": "ISO8601",
  "days": [
    {
      "day_of_week": 1,
      "exercises": [
        {
          "name": "Push-ups",
          "sets": 3,
          "reps": 15,
          "duration_minutes": 10
        }
      ]
    }
  ]
}
```

### LoginResponse

```json
{
  "access_token": "JWT (ES256, 15min)",
  "refresh_token": "opaque string (7d)",
  "token_type": "Bearer",
  "expires_in": 900,
  "user_id": "uuid",
  "role": "client|admin"
}
```

## Ошибки

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
