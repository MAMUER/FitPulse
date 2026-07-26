# ADR 0019: Рефакторинг JWT-аутентификации: ports/adapters в gateway и user-service

## Контекст

Пакет `internal/auth` использовался как общая библиотека JWT-аутентификации.
Он напрямую импортировался из `cmd/gateway`, `cmd/user-service` и `internal/middleware`.

Это создавало два проблемы:
1. **Нарушение гексагональной архитектуры**: domain/application слои зависели от инфраструктурной библиотеки.
2. **Сложность тестирования**: нельзя было подменить реализацию токен-провайдера на mock без изменения импортов.

## Решение

Разделить `internal/auth` на две части и внедрить паттерн ports/adapters в каждом сервисе:

```
internal/auth/
├── claims/          # Доменные типы (Claims, JWKSKey, JWKSResponse)
└── jwt/             # Инфраструктурная реализация (ES256, подпись, валидация)

cmd/gateway/
├── ports/auth.go    # TokenProvider интерфейс
└── infra/jwt_adapter.go  # Адаптер → internal/auth/jwt

cmd/user-service/
├── ports/auth.go    # TokenProvider интерфейс
└── infra/jwt_adapter.go  # Адаптер → internal/auth/jwt
```

### Правила

1. **Domain/application слои** зависят только от `ports.TokenProvider` (интерфейс).
2. **Infra слой** (`infra/jwt_adapter.go`) импортирует `internal/auth/jwt` и реализует порт.
3. **Composition root** (`main.go`) создаёт адаптер и внедряет его в сервис.
4. **`internal/auth`** — shared library без бизнес-логики.

## Последствия

### Положительные
- **Тестируемость**: в тестах можно подменить `TokenProvider` на mock.
- **Сменяемость**: при смене алгоритма/библиотеки JWT меняется только адаптер.
- **Чистота слоёв**: application слой не знает о криптографических деталях.
- **Документированность**: роль `internal/auth` четко определена в `docs/ARCHITECTURE.md §9`.

### Отрицательные
- Небольшой рост кода (два новых файла на сервис: порт + адаптер).
- Необходимость поддерживать консистентность интерфейсов при изменении API `internal/auth/jwt`.

## Реализация

- `internal/auth/claims/claims.go` — доменные типы.
- `internal/auth/jwt/jwt.go` — инфраструктурная реализация + тесты.
- `cmd/gateway/ports/auth.go` — интерфейс `TokenProvider`.
- `cmd/gateway/infra/jwt_adapter.go` — адаптер.
- `cmd/user-service/ports/auth.go` — интерфейс `TokenProvider`.
- `cmd/user-service/infra/jwt_adapter.go` — адаптер.
- `docs/ARCHITECTURE.md §9` — документация роли `internal/auth`.
- Обновлены все импорты в `cmd/gateway`, `cmd/user-service`, `internal/middleware`.

## Статус

Принято, реализовано, протестировано.
