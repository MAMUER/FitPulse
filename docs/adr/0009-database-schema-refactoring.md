# ADR 0009: Рефакторинг схемы базы данных и service layer

## Статус

Принято

## Контекст

Биометрический и тренировочный сервисы требовали улучшений в consistency данных, покрытии тестами и надёжности сервисов. Существующие репозиторные паттерны  не имели корректного трекинга timestamp'ов, а покрытие тестами было недостаточным для валидации критических путей.

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

## Рассмотренные альтернативы

- Использование database triggers для timestamps: добавляет coupling с БД, менее portable.
- Полное reliance на integration-тесты: более медленная обратная связь, сложнее дебажить.
