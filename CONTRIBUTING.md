# Руководство по внесению вклада (Contributing Guide)

Спасибо за интерес к проекту FitPulse! Это руководство поможет вам понять, как внести свой вклад в развитие проекта.

## Содержание

1. [С чего начать](#с-чего-начать)
2. [Как внести изменения](#как-внести-изменения)
3. [Стандарты кода](#стандарты-кода)
4. [Тестирование](#тестирование)
5. [Pull Request процесс](#pull-request-процесс)
6. [Code Review](#code-review)
7. [Сообщество и коммуникация](#сообщество-и-коммуникация)

## С чего начать

### Требования для разработки

- **Go**: версия 1.26+
- **Python**: версия 3.14+ (для ML-сервисов)
- **Docker**: Docker Desktop / Docker Engine с поддержкой BuildKit (современные версии)
- **Git**: для управления версиями

### Первая настройка

1. **Fork репозитория** на GitHub
2. **Клонируйте ваш fork**:

   ```bash
   git clone https://github.com/your-username/fitpulse.git
   cd fitpulse
   ```

3. **Добавьте upstream remote** (опционально, если работаете через fork):

   ```bash
   git remote add upstream https://github.com/MAMUER/fitpulse.git
   ```

   Если вы работаете напрямую в репозитории, этот шаг можно пропустить.

 4. **Установите зависимости**:

    ```bash
    go mod tidy
    pip install -r cmd/ml_generator/requirements.txt  # для ML-сервисов
    ```

 5. **Настройте окружение**: переменные окружения задаются через GitHub Secrets и Variables. Локальный запуск сервисов не поддерживается — deploy только на VPS. Для локальных интеграционных тестов используйте `testcontainers-go` (зависимости поднимаются автоматически при запуске `go test -tags=integration`).

## Как внести изменения

### Ветвление

Мы используем модель ветвления **Trunk-Based Development** для CI/CD с частыми релизами:

- `main` — единственная долгоживущая ветка (trunk), всегда деплоибельна
- `feature/*` — короткоживущие ветки функций (≤ 2 дней), напрямую в `main` через PR
- `bugfix/*` — исправления ошибок (≤ 1 дня), напрямую в `main` через PR
- `hotfix/*` — срочные исправления для production через отдельный PR в `main`

> **Нет** веток `develop` и `release/*` — все изменения попадают в `main` через короткоживущие ветки с feature flags.

### Создание ветки

```bash
# Всегда начинайте от актуального main
git checkout main
git pull upstream main

# Создайте новую короткоживущую ветку
git checkout -b feature/your-feature-name
```

### Именование веток

| Тип | Формат | Пример |
| ----- | --------- | -------- |
| Feature | `feature/<описание>` | `feature/email-verification` |
| Bugfix | `bugfix/<описание>` | `bugfix/login-timeout` |
| Hotfix | `hotfix/<описание>` | `hotfix/security-patch` |

### Коммиты

Мы следуем стандарту [Conventional Commits](https://www.conventionalcommits.org/):

```text
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Типы коммитов:**

- `feat`: новая функция
- `fix`: исправление ошибки
- `docs`: изменение документации
- `style`: форматирование, не влияющее на логику
- `refactor`: рефакторинг кода
- `test`: добавление/изменение тестов
- `chore`: обслуживание (dependencies, build)

**Примеры:**

```bash
feat(auth): добавить поддержку invite-кодов для регистрации админов

docs(readme): обновить документацию
```

## Стандарты кода

### Go код

1. **Форматирование**: Используйте `go fmt` перед каждым коммитом

   ```bash
   go fmt ./...
   ```

2. **Линтинг**: Запустите golangci-lint

   ```bash
   make lint
   ```

3. **Структура проекта**: Следуйте структуре, описанной в README.md
   - Domain модели в `internal/domain/`
   - Repository слой в `internal/repository/`
   - Domain/Service логика в `internal/*/`, entrypoints в `cmd/*/`
   - Adapters в `internal/*/adapters/`

4. **Обработка ошибок**:

   ```go
   // Правильно
   if err != nil {
       return fmt.Errorf("failed to process biometric data: %w", err)
   }
   
   // Используйте errors.Is и errors.As для проверки типов ошибок
   if errors.Is(err, context.Canceled) {
       return nil
   }
   ```

5. **Контекст**: Всегда передавайте context первым параметром

   ```go
   func (s *Service) GetUser(ctx context.Context, id string) (*User, error)
   ```

### Python код (ML-сервисы)

1. **PEP 8**: Следуйте руководству по стилю Python
2. **Type hints**: Используйте аннотации типов
3. **Docstrings**: Документируйте публичные функции

### Протоколы (Protobuf)

1. **Версионирование**: Не удаляйте и не изменяйте существующие поля
2. **Резервируйте номера**: Для удалённых полов используйте `reserved`
3. **Комментарии**: Документируйте каждое сообщение и сервис

#### Генерация кода (`make proto`)

Перед изменением `.proto` файлов убедитесь, что установлены зависимости:

- **`protoc`** (компилятор Protocol Buffers) — версия, совместимая с `protoc-gen-go`/`protoc-gen-go-grpc`.
- **Плагины Go**:
  ```bash
  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
  ```
  и убедитесь, что `$GOPATH/bin` (обычно `~/go/bin`) добавлен в `PATH`, иначе `protoc` не найдёт плагины.

Установка `protoc` по платформам:

```bash
# macOS (Homebrew)
brew install protobuf

# Ubuntu / Debian
sudo apt-get update && sudo apt-get install -y protobuf-compiler

# Windows (Chocolatey)
choco install protoc

# Через Go (альтернатива, требует C-инструментарий)
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

После правок в `api/proto/*.proto` сгенерируйте код и закоммитьте результат:

```bash
make proto
```

Сгенерированные файлы попадают в `api/gen/**` (см. `docs/ARCHITECTURE.md`, раздел структуры проекта).

## Тестирование

### Запуск тестов

```bash
# Unit-тесты (без внешних зависимостей)
make test

# С покрытием (проверка порога 75% для business-logic пакетов)
make test-cover

# Интеграционные тесты (требуют Docker и тег `-tags=integration`)
# Зависимости (PostgreSQL, Valkey, RabbitMQ) поднимаются автоматически через testcontainers-go
go test -v -tags=integration ./...

# API тесты
make api-test

# Нагрузочное тестирование (требует k6)
make load-test

# Полный набор проверок (tidy + fmt + vet + lint + tests + утилиты)
make check
```

### Покрытие кодом

Порог покрытия **75%** считается только для пакетов бизнес-логики: `internal/` (кроме инфраструктурных пакетов: `grpc`, `db`, `queue`, `middleware`, `crypto`, `totp`, `telemetry`, `testcontainers`). Исключаются сгенерированный код (`api/gen/`) и моки (`mocks/`).

Проверка покрытия:

```bash
make test-cover
```

Или вручную:

```bash
go test -count=1 -v -coverprofile=coverage.out ./internal/...
go tool cover -html=coverage.out -o coverage.html
```

### Написание тестов

1. **Unit-тесты**: Тестируйте изолированную логику без внешних зависимостей
2. **Integration-тесты**: Тестируйте взаимодействие с БД, Valkey, RabbitMQ
3. **E2E-тесты**: Тестируйте полный поток через API

Пример unit-теста:

```go
func TestMedicalService_ClassifyState(t *testing.T) {
    // Arrange
    mockRepo := mocks.NewMockBiometricRepository(t)
    service := NewMedicalService(mockRepo)
    
    // Act
    result, err := service.ClassifyState(context.Background(), userID)
    
    // Assert
    assert.NoError(t, err)
    assert.Equal(t, ExpectedClass, result.Class)
}
```

## Pull Request процесс

### Перед отправкой PR

1. **Обновите ветку** от upstream main:

    ```bash
    git fetch upstream
    git rebase upstream/main
    ```

2. **Запустите все проверки**:

   ```bash
   make check
   ```

   Эта команда запускает: `go mod tidy`, `go fmt`, `go vet`, импорты, `golangci-lint` и unit-тесты с проверкой покрытия.

   При необходимости запустите интеграционные тесты локально (требуют Docker):
   ```bash
   go test -v -tags=integration ./...
   ```

3. **Проверьте покрытие тестами**:

    ```bash
    make test-cover
    ```

    Минимальное требование: **75%** для пакетов бизнес-логики.

4. **Обновите документацию**, если изменили API или поведение (README.md, API docs, etc.)

5. **Убедитесь, что CI/CD пайплайн проходит** (lint, test, build, security scan)

### Создание PR

1. Перейдите на GitHub и создайте Pull Request
2. Выберите базовую ветку: `main`
3. Заполните шаблон PR:
   - Описание изменений
   - Связанные issue
   - Тип изменений (feat/fix/docs/etc.)
   - Чеклист проверок
4. Дождитесь прохождения всех CI/CD проверок (lint, test, build, security scan)
5. Получите approval от мейнтейнера

### Шаблон PR

```markdown
## Описание
Краткое описание изменений

## Тип изменений
- [ ] Новая функция (feat)
- [ ] Исправление ошибки (fix)
- [ ] Документация (docs)
- [ ] Рефакторинг (refactor)
- [ ] Тесты (test)
- [ ] Обслуживание (chore)

## Связанные issue
Fixes #123

## Чеклист
 - [ ] Код отформатирован (`go fmt ./...`)
 - [ ] Линтер пройден (`make lint`)
 - [ ] Покрытие тестами >= 75% для бизнес-логики (проверяется через `make test-cover`)
 - [ ] Документация обновлена
 - [ ] Изменения протестированы (`make check`)
 - [ ] Интеграционные тесты запущены локально при необходимости (`go test -v -tags=integration ./...`)
 - [ ] Все CI/CD проверки прошли успешно
```

## Code Review

### Критерии acceptance

PR будет принят, если:

1. Все CI/CD проверки прошли успешно
2. Код соответствует стандартам проекта
3. Покрытие тестами не уменьшилось
4. Нет замечаний от ревьюеров
5. Документация актуальна

### Время ревью

- Обычные PR: в течение 2–3 рабочих дней
- Критические hotfix: best effort, в течение 1–3 рабочих дней (без гарантий 24/7)

### Процесс ревью

1. Автоматические проверки (CI/CD) — все джобы должны пройти успешно (lint, test, security scan)
2. Проверка на соответствие стандартам проекта
3. Ревью от минимум одного мейнтейнера
4. Исправление замечаний
5. Approval и merge

**Merge frequency**: approved PRы мержатся в `main` по готовности, обычно в течение 1 рабочего дня после approval. Hotfixы мержатся приоритетно.

## Сообщество и коммуникация

### Где задать вопросы

- **GitHub Issues**: для багов и фич
- **GitHub Discussions**: для общих вопросов
- **GitHub Security Advisory**: для конфиденциальных сообщений об уязвимостях [https://github.com/MAMUER/fitpulse/security/advisories](https://github.com/MAMUER/fitpulse/security/advisories)
- **Email**: <mihnikolaenko12@yandex.ru>

### Кодекс поведения

Будьте уважительны и конструктивны в общении. Мы приветствуем участников любого уровня опыта.

### Признание вклада

Все контрибьюторы автоматически отображаются на странице Contributors в GitHub. Для дополнительного признания используйте [all-contributors bot](https://allcontributors.org/docs/en/overview).

---

## Дополнительные ресурсы

- [Документация архитектуры](docs/ARCHITECTURE.md)
- [Архитектурные решения](docs/adr/)
- [Runbooks](docs/runbooks/)
- [Swagger спецификация](api/rest/swagger.yaml)

Спасибо за ваш вклад в развитие FitPulse!
