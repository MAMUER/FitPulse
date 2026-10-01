# ADR 0009: Рефакторинг схемы базы данных и service layer

## Статус

Принято

## Контекст

Биометрический и тренировочный сервисы требовали улучшений в consistency данных, покрытии тестами и надёжности сервисов. Существующие репозиторные паттерны не имели корректного трекинга timestamp'ов, а покрытие тестами было недостаточным для валидации критических путей.

## Решение

1. **Усиление биометрического репозитория**
   - в схеме БД `biometric_data` добавлено поле `created_at TIMESTAMPTZ DEFAULT NOW()` (миграция `V1__full_schema.sql`);
   - репозиторий `internal/repository/postgres/biometric_repository.go` возвращает `created_at` через `RETURNING` в запросах `GetLatest`, `Update`;
   - гарантирует, что все сохранённые биометрические записи имеют timestamps для аудита и дебага.

2. **Расширение схемы тренировочного сервиса**
   - расширены data models тренировочного сервиса дополнительными полями и связями;
   - улучшено представление данных для тренировочных планов и трейкинга прогресса.

3. **Улучшение покрытия тестами**
   - добавлены комплексные unit-тесты для data processor с обработкой environment variables;
   - созданы integration-тесты для training service (GeneratePlan, GetProgress);
   - реализованы mock database interactions для изолированного unit-тестирования.

## Последствия

- **Плюсы**: лучшая consistency данных с automatic timestamp tracking;
- **Плюсы**: комплексное покрытие тестами (unit + integration) повышает надёжность;
- **Плюсы**: mock-based unit-тесты ускоряют циклы разработки;
- **Нейтрально**: требуется миграция схемы БД для существующих данных.

## Реализация

- изменён `internal/repository/postgres/biometric_repository.go` — запросы возвращают `created_at` через `RETURNING`;
- добавлен `internal/repository/pgx/training_repository.go` с методами `CreatePlan`, `GetPlan`, `ListPlans`, `GetProgress`, `DeletePlan`, `UpdatePlan`;
- добавлены unit-тесты: `cmd/data-processor/data_processor_unit_test.go`, `cmd/training-service/training_service_unit_test.go`;
- добавлены integration-тесты: `cmd/data-processor/data_processor_integration_test.go`, `cmd/training-service/training_service_integration_test.go`.

## Выбор JSONB для хранения данных анкеты

### Контекст выбора JSONB

FitPulse требует хранить данные анкеты пользователя (онбординг) со следующими требованиями:

1. **Гибкая схема**: вопросы анкеты могут меняться со временем; нужно добавлять/удалять поля без миграций.
2. **Queryability**: необходимо фильтровать/сортировать по полям анкеты (например, найти всех пользователей с целью "похудение").
3. **Производительность**: данные анкеты читаются при каждой генерации плана; записи происходят один раз при онбординге.
4. **Atomicity**: флаг завершения анкеты и timestamp должны обновляться атомарно с данными.

### Решение по JSONB

Хранить данные анкеты как **JSONB** в `user_profiles.survey_data` с отдельными колонками для трекинга завершения:

```sql
ALTER TABLE user_profiles
    ADD COLUMN IF NOT EXISTS survey_data JSONB DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS survey_completed BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS survey_completed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_user_profiles_survey_data
    ON user_profiles USING GIN (survey_data);
```

### Последствия выбора JSONB

- **Плюсы**: гибкость схемы (добавление/удаление вопросов без ALTER TABLE);
- **Плюсы**: query power (GIN индекс позволяет быстрые JSONB запросы, например `@>` containment);
- **Плюсы**: атомарные обновления (данные, флаг и timestamp обновляются в одном UPDATE);
- **Плюсы**: типобезопасность на уровне Go service (валидация/маршаллинг survey как `map[string]interface{}`);
- **Минусы**: нет column-level constraints (нельзя enforce `survey_data.goals IS NOT NULL` на уровне БД);
- **Минусы**: больший размер строки (JSONB хранится inline; большие анкеты увеличивают размер строки);
- **Минусы**: валидация структуры в Go service layer (нельзя делегировать валидацию БД).

### Реализация JSONB

- `db/migrations/V1__full_schema.sql` — добавлены survey колонки и GIN индекс;
- `internal/repository/postgres/profile_repository.go` — реализованы `SaveSurvey`/`LoadSurvey`;
- `internal/domain/service/user_service.go` — добавлены `SaveSurvey`/`LoadSurvey`;
- `api/proto/user.proto` — добавлены `SaveSurvey`/`LoadSurvey` сообщения;
- `cmd/gateway/handlers_survey.go` — HTTP endpoints `GET/POST /api/v1/survey`.

### Альтернативы для JSONB

1. **Нормализованные таблицы**: `survey_responses` таблица с `user_id`, `question_key`, `value`. Отклонено — слишком много join'ов для read-heavy workload.
2. **JSON колонка без GIN индекса**: отклонено — нельзя эффективно query'ить.
3. **Отдельные колонки**: `survey_goal`, `survey_level` и т.д. Отклонено — изменения схемы требуют миграций.

## Рассмотренные альтернативы

- Использование database triggers для timestamps: добавляет coupling с БД, менее portable.
- Полное reliance на integration-тесты: более медленная обратная связь, сложнее дебаггинг.
