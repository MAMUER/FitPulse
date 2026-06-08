# ADR 0010: Рефакторинг схемы базы данных и service layer

## Контекст

Биометрический и тренировочный сервисы требовали улучшений в consistency данных, покрытии тестами и надёжности сервисов. Существующие репозиторные паттерны lacked proper timestamp tracking, а покрытие тестами было недостаточным для валидации критических путей.

## Решение

1. **Усиление биометрического репозитория**
   - добавлено поле `created_at` в метод Save в `biometric_repository.go`;
   - гарантирует, что все сохранённые биометрические записи имеют timestamps для аудита и дебага.

2. **Расширение схемы тренировочного сервиса**
   - расширены data models тренировочного сервиса дополнительными полями и связями;
   - улучшено представление данных для тренировочных планов и трейкинга прогресса.

3. **Улучшение покрытия тестами**
   - добавлены комплексные unit-тесты для data processor с обработкой environment variables;
   - созданы integration-тесты для training service (GeneratePlan, GetProgress);
   - реализованы mock database interactions для изолированного unit-тестирования.

4. **Очистка зависимостей**
   - удалена неиспользуемая зависимость `postgres` из `go.mod`.

## Последствия

- **Плюсы**: лучшая consistency данных с automatic timestamp tracking;
- **Плюсы**: комплексное покрытие тестами (unit + integration) повышает надёжность;
- **Плюсы**: mock-based unit-тесты ускоряют циклы разработки;
- **Плюсы**: более чистый граф зависимостей снижает время сборки и security surface;
- **Нейтрально**: требуется миграция схемы БД для существующих данных.

## Реализация

- изменён `internal/repository/biometric_repository.go`;
- обновлён `cmd/biometric-service/biometric_service_test.go`;
- добавлен `cmd/data-processor/data_processor_unit_test.go`;
- создан `cmd/training-service/training_service_integration_test.go`;
- добавлен `cmd/training-service/training_service_unit_test.go`;
- обновлён `go.mod` для удаления неиспользуемой зависимости.

## Рассмотренные альтернативы

- Использование database triggers для timestamps: добавляет coupling с БД, менее portable.
- Полное reliance на integration-тесты: более медленная обратная связь, сложнее дебажить.
- Сохранение неиспользуемых зависимостей: увеличивает размер бинарника и поверхность уязвимостей.
