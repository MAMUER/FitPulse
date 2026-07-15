# FitPulse — Нормализация базы данных

## Цель
Обеспечить непротиворечивость, масштабируемость и соответствие нормальным формам для всей схемы PostgreSQL.

---

## Полный перечень таблиц

### Ядро и аутентификация
- `users`
- `email_verifications`
- `invite_codes`
- `invite_code_uses`
- `oauth_states`
- `device_provider_accounts`
- `device_sync_log`
- `refresh_tokens`

### Профиль и здоровье
- `user_profiles`
- `user_goals`
- `user_contraindications`
- `user_health_conditions`
- `user_body_composition`
- `user_menstrual_cycles`
- `user_menstrual_symptoms`
- `user_menstrual_moods`

### Устройства и биометрия
- `devices`
- `biometric_data`
- `device_ingest_log`

### Тренировки
- `training_plans`
- `training_plan_weeks`
- `training_plan_days`
- `training_exercises`
- `workout_completions`

### Достижения
- `achievements`
- `user_achievements`

---

## 1NF — Первая нормальная форма

- Все атрибуты атомарны, без повторяющихся групп и массивов в бизнес-логике.
- Примеры:
  - `user_goals` и `user_contraindications` вынесены из `user_profiles` в отдельные таблицы.
  - `user_menstrual_symptoms`, `user_menstrual_moods` — атомарные строки вместо списков.
  - `invite_code_uses` — отдельная строка на каждое использование, вместо счётчика.

## 2NF — Вторая нормальная форма

- Устранены частичные зависимости от составного ключа.
- Все таблицы имеют явный суррогатный `id` (`UUID`).
- Примеры:
  - `training_exercises` зависит от `id`, а не от пары `(day_id, exercise_name)`.
  - `user_achievements` зависит от `(user_id, achievement_id)` как составного PK, нет частичных зависимостей.

## 3NF — Третья нормальная форма

- Устранены транзитивные зависимости.
- Примеры:
  - `user_body_composition` хранит измерения напрямую, без вычисляемых полей из `user_profiles`.
  - `device_sync_log` ссылается на `device_provider_accounts`, а не дублирует данные провайдера.
  - `training_plan_weeks` и `training_plan_days` не дублируют данные плана.

## BCNF — Нормальная форма Бойса-Кодда

- Каждая таблица удовлетворяет BCNF:
  - единственный определитель ключа — первичный ключ (`id`) или натуральный ключ;
  - нет неключевых функциональных зависимостей.
- Примеры:
  - `users.email_hash` — уникальный, но не PK; вычисляется как HMAC-SHA256 с глобальной солью для детерминированного поиска без утечки паттернов, PK остаётся `id`.
  - `users.full_name_hash`, `users.nickname_hash` — blind index для поиска по рандомизированно зашифрованным full_name и nickname; позволяют искать пользователя без расшифровки ciphertext, PK остаётся `id`.
  - `device_provider_accounts(user_id, provider)` — уникальность обеспечена UNIQUE, PK — `id`.

## 4NF — Четвёртая нормальная форма

- Устранены многозначные зависимости.
- Примеры:
  - симптомы и настроения цикла вынесены в `user_menstrual_symptoms` и `user_menstrual_moods`.
  - `device_provider_accounts.scopes` — массив, но OPTIONAL; в 4NF допускается, если провайдер возвращает список как единое целое.

## 5NF — Пятая нормальная форма (проектно-следовательно)

- Нормализация до 5NF выполнена декомпозицией на основе:
  - пользователь → профиль, цели, противопоказания, состояния здоровья, устройства, biometrics, планы, достижения;
  - план → недели → дни → упражнения;
  - invite_code → использования.
- Нет нестопроизводных соединений; Join-декомпозиция не теряет информацию.

---

## Итоговые таблицы

### `users`
```
id (PK), email, email_encrypted, email_hash, password_hash, full_name, full_name_encrypted,
full_name_hash, full_name_nonce, nickname, nickname_encrypted, nickname_hash, nickname_nonce,
profile_photo_url, role, provider, external_id,
email_confirmed, totp_secret_encrypted, totp_enabled, totp_backup_codes_hash,
totp_backup_codes_remaining, created_at, updated_at
```

### `email_verifications`
```
id (PK), user_id (FK), email, email_encrypted, token, token_encrypted,
expires_at, used, created_at
```

### `invite_codes`
```
id (PK), code, role, specialty, max_uses, created_by (FK), expires_at, is_active, created_at
```

### `invite_code_uses`
```
id (PK), invite_code_id (FK), user_id (FK), used_at
```

### `oauth_states`
```
state (PK), user_id (FK), provider, expires_at, created_at
```

### `device_provider_accounts`
```
id (PK), user_id (FK), provider, provider_user_id, access_token, refresh_token,
token_expires_at, scopes, webhook_subscription_id, last_sync_at, is_active, created_at, updated_at
UNIQUE(user_id, provider)
```

### `device_sync_log`
```
id (PK), provider_account_id (FK), sync_type, records_count, started_at, completed_at,
status, error_message, created_at
```

### `refresh_tokens`
```
id (PK), token UNIQUE, user_id (FK), expires_at, revoked, created_at
```

### `user_profiles`
```
user_id (PK, FK), age, gender, height_cm, weight_kg, fitness_level, nutrition, sleep_hours,
created_at, updated_at
```

### `user_goals`
```
user_id (PK, FK), goal (PK), created_at
```

### `user_contraindications`
```
user_id (PK, FK), contraindication (PK), created_at
```

### `user_health_conditions`
```
id (PK), user_id (FK), condition_type CHECK (...), condition_name, severity CHECK (...),
diagnosed_at, is_active, notes, created_at, updated_at
UNIQUE (user_id, condition_type, condition_name)
```

### `user_body_composition`
```
id (PK), user_id (FK), recorded_at, weight_kg CHECK (...), height_cm CHECK (...),
bmi, body_fat_percentage CHECK (...), muscle_mass_percentage CHECK (...),
bone_mass_percentage CHECK (...), water_percentage CHECK (...),
visceral_fat_rating CHECK (...), metabolic_age CHECK (...),
source CHECK (...), created_at
```

### `user_menstrual_cycles`
```
id (PK), user_id (FK), cycle_start_date, cycle_end_date CHECK (...), flow_intensity CHECK (...),
notes, created_at, updated_at
```

### `user_menstrual_symptoms`
```
id (PK), cycle_id (FK), symptom, severity CHECK (...), created_at
```

### `user_menstrual_moods`
```
id (PK), cycle_id (FK), mood, created_at
```

### `devices`
```
id (PK), user_id (FK), device_type, device_name, token UNIQUE, is_connected, last_sync, created_at
```

### `biometric_data`
```
id (PK), user_id (FK), metric_type, value CHECK (value >= 0), timestamp, device_type, created_at
```

### `device_ingest_log`
```
id (PK), device_id (FK), metric_type, timestamp, quality DEFAULT 'good', created_at
```

### `training_plans`
```
id (PK), user_id (FK), name, training_goal CHECK (...), training_location CHECK (...),
available_time CHECK (...), duration_weeks CHECK (...), generated_at, start_date, end_date,
status CHECK (...), created_at
```

### `training_plan_weeks`
```
id (PK), training_plan_id (FK), week_number CHECK (>0), total_training_days,
total_duration_minutes, average_intensity, UNIQUE(training_plan_id, week_number)
```

### `training_plan_days`
```
id (PK), week_id (FK), day_of_week CHECK (0-6), training_date, training_type,
is_rest_day, total_duration_minutes, intensity_level, notes
```

### `training_exercises`
```
id (PK), day_id (FK), exercise_name, duration_minutes, intensity, sets, reps,
rest_seconds DEFAULT 60, description, video_url, sort_order DEFAULT 0
```

### `workout_completions`
```
id (PK), user_id (FK), training_plan_id (FK), workout_id, scheduled_date DEFAULT CURRENT_DATE,
completed BOOLEAN DEFAULT FALSE, completed_at, feedback, rating CHECK (1-5), created_at
```

### `achievements`
```
id (PK), name, description, criteria JSONB, icon_url, created_at
```

### `user_achievements`
```
user_id (PK, FK), achievement_id (PK, FK), earned_at
```

---

## Миграции
- V1 — extensions
- V2 — users, email_verifications, invite_codes, invite_code_uses
- V3 — user_profiles, user_goals, user_contraindications
- V4 — devices
- V5 — biometric_data, device_ingest_log
- V6 — training_plans, training_plan_weeks, training_plan_days, training_exercises, workout_completions
- V7 — achievements, user_achievements
- V8 — views (invite_code_stats, user_profiles_with_goals)
- V9 — functions (create_invite_code, use_invite_code)
- V10 — training_plans.classification_class
- V11 — oauth_states, device_provider_accounts, device_sync_log
- V12 — users.provider, users.external_id
- V13 — users TOTP columns
- V14 — users PII encrypted columns, email_verifications encrypted columns
- V15 — user_health_conditions
- V16 — user_body_composition
- V17 — user_menstrual_cycles, user_menstrual_symptoms, user_menstrual_moods
- V18 — pgsodium extension
- V19 — PII blind indexes (full_name_hash, full_name_nonce, nickname_hash, nickname_nonce)

> `configs/k8s/base/jobs/init-db.sql` содержит все миграции V1–V19 в одном файле. Для инициализации базы данных в production примените этот файл через `kubectl apply -f configs/k8s/base/jobs/init-db.yaml` после создания namespace и secrets.

---

## Валидация и бизнес-правила

- `users.role` ограничен CHECK (`client`, `admin`).
- `user_profiles.gender` ограничен CHECK (`male`, `female`, `other`).
- `user_health_conditions.condition_type` ограничен CHECK (`allergy`, `disease`, `disability`, `other`).
- `user_health_conditions` имеет UNIQUE (`user_id`, `condition_type`, `condition_name`) для поддержки `ON CONFLICT` в upsert.
- `user_body_composition.weight_kg` ограничен диапазоном `[1, 500]`.
- `user_body_composition.height_cm` ограничен диапазоном `[50, 300]`.
- `user_body_composition.body_fat_percentage` ограничен диапазоном `[1, 100]`.
- `user_body_composition.muscle_mass_percentage` ограничен диапазоном `[1, 100]`.
- `user_body_composition.bone_mass_percentage` ограничен диапазоном `[1, 100]`.
- `user_body_composition.water_percentage` ограничен диапазоном `[1, 100]`.
- `user_body_composition.visceral_fat_rating` ограничен диапазоном `[1, 59]`.
- `user_body_composition.metabolic_age` ограничен диапазоном `[10, 100]`.
- `user_body_composition.source` ограничен CHECK (`okok`, `manual`).
- `user_menstrual_cycles.cycle_end_date >= cycle_start_date` через `CONSTRAINT chk_cycle_dates`.
- `training_plans.duration_weeks` ограничен диапазоном `(0, 52]`.
- `training_plan_weeks.week_number` ограничен `CHECK (week_number > 0)`.
- `training_plan_days.day_of_week` ограничен `CHECK (day_of_week >= 0 AND day_of_week <= 6)`.
- `biometric_data.value` >= 0.
- `workout_completions.rating` ограничен `CHECK (rating IS NULL OR (rating >= 1 AND rating <= 5))`.
- `invite_code_uses` обеспечивает точный учёт использований без `used_count` в `invite_codes`.

---

## Согласованность с кодом

- `cmd/device-connector/main.go` содержит `initDatabase()` с inline DDL для `devices` и `device_ingest_log`. Эти таблицы должны совпадать с миграциями V4 и V5: `user_id` типа `UUID`, `device_ingest_log.quality` имеет `DEFAULT 'good'`.
- `cmd/user-service/main.go` использует `ON CONFLICT (user_id, condition_type, condition_name)` в `UpsertHealthCondition`. Миграция V15 содержит `UNIQUE (user_id, condition_type, condition_name)` для поддержки этого upsert.
- `training_plans.training_goal` CHECK включает значения `endurance_e1e2`, `threshold_e3`, `strength_hiit` для совместимости с ML-классификатором.
