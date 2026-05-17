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
- **Python**: версия 3.11+ (для ML-сервисов)
- **Docker**: версия 24+ / Docker Compose v2
- **Git**: для управления версиями

### Первая настройка

1. **Fork репозитория** на GitHub
2. **Клонируйте ваш fork**:
   ```bash
   git clone https://github.com/your-username/fitpulse.git
   cd fitpulse
   ```
3. **Добавьте upstream remote**:
   ```bash
   git remote add upstream https://github.com/original-org/fitpulse.git
   ```
4. **Установите зависимости**:
   ```bash
   go mod tidy
   pip install -r requirements.txt  # для ML-сервисов
   ```
5. **Настройте окружение**:
   ```bash
   cp .env.example .env
   # Отредактируйте .env с необходимыми значениями
   ```
6. **Запустите локально**:
   ```bash
   scripts/run-local.bat  # Windows
   # или
   docker compose -f deployments/docker-compose.yml up -d  # Linux/Mac
   ```

## Как внести изменения

### Ветвление

Мы используем модель ветвления GitFlow:

- `main` — основная ветка production
- `develop` — ветка разработки
- `feature/*` — новые функции
- `bugfix/*` — исправления ошибок
- `hotfix/*` — срочные исправления для production
- `release/*` — подготовка релиза

### Создание ветки

```bash
# Всегда начинайте от актуальной develop
git checkout develop
git pull upstream develop

# Создайте новую ветку
git checkout -b feature/your-feature-name
```

### Именование веток

| Тип | Формат | Пример |
|-----|--------|--------|
| Feature | `feature/<описание>` | `feature/email-verification` |
| Bugfix | `bugfix/<описание>` | `bugfix/login-timeout` |
| Hotfix | `hotfix/<описание>` | `hotfix/security-patch` |
| Release | `release/<версия>` | `release/2.0.0` |

### Коммиты

Мы следуем стандарту [Conventional Commits](https://www.conventionalcommits.org/):

```
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

fix(biometric): исправить дублирование данных при эмуляции устройств

docs(readme): обновить инструкции по локальной установке
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
   - Service логика в `cmd/*/`
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

## Тестирование

### Запуск тестов

```bash
# Unit-тесты
make test

# С покрытием
make test-cover

# Интеграционные тесты
make test-integration

# API тесты
make api-test

# Нагрузочное тестирование
make load-test
```

### Покрытие кодом

Минимальное требование покрытия: **80%** для нового кода.

Проверка покрытия:
```bash
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Написание тестов

1. **Unit-тесты**: Тестируйте изолированную логику без внешних зависимостей
2. **Integration-тесты**: Тестируйте взаимодействие с БД, Redis, RabbitMQ
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

1. **Обновите ветку** от upstream develop:
   ```bash
   git fetch upstream
   git rebase upstream/develop
   ```

2. **Запустите все проверки**:
   ```bash
   make check  # fmt + vet + lint + test + build
   ```

3. **Проверьте покрытие тестами**:
   ```bash
   go test -cover ./...
   ```

4. **Обновите документацию**, если изменили API или поведение

### Создание PR

1. Перейдите на GitHub и создайте Pull Request
2. Выберите базовую ветку: `develop`
3. Заполните шаблон PR:
   - Описание изменений
   - Связанные issue
   - Тип изменений (feat/fix/docs/etc.)
   - Чеклист проверок

### Шаблон PR

```markdown
## Описание
Краткое описание изменений

## Тип изменений
- [ ]  Новая функция (feat)
- [ ] Исправление ошибки (fix)
- [ ] Документация (docs)
- [ ] Рефакторинг (refactor)
- [ ] Тесты (test)
- [ ] 🔧 Обслуживание (chore)

## Связанные issue
Fixes #123

## Чеклист
- [ ] Код отформатирован (go fmt)
- [ ] Все тесты проходят (make test)
- [ ] Покрытие тестами >= 80%
- [ ] Документация обновлена
- [ ] Изменения протестированы локально
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

- Обычные PR: в течение 2-3 дней
- Критические hotfix: в течение 24 часов

### Процесс ревью

1. Автоматические проверки (CI/CD)
2. Ревью от минимум одного мейнтейнера
3. Исправление замечаний
4. Approval и merge

## Сообщество и коммуникация

### Где задать вопросы

- **GitHub Issues**: для багов и фич
- **GitHub Discussions**: для общих вопросов
- **Email**: support@fitpulse.local (если настроено)

### Кодекс поведения

Будьте уважительны и конструктивны в общении. Мы приветствуем участников любого уровня опыта.

### Признание вклада

Все контрибьюторы будут указаны в файле CONTRIBUTORS.md (будет создан).

---

## Дополнительные ресурсы

- [Документация архитектуры](docs/ARCHITECTURE.md)
- [Архитектурные решения](docs/adr/)
- [Runbooks](docs/runbooks/)
- [Swagger спецификация](api/rest/swagger.yaml)

Спасибо за ваш вклад в развитие FitPulse! 🎯
