# Device Emulator — Эмуляция носимых устройств

Этот сервис эмулирует работу носимых устройств для тестирования и разработки, когда реальные устройства недоступны.

## Поддерживаемые устройства

| Устройство | Пульс | ЭКГ | Давление | SpO₂ | Температура | Сон | Шаги | HRV |
|-----------|-------|-----|----------|------|-------------|-----|------|-----|
| Apple Watch | ОК | ОК | НЕТ | ОК | НЕТ | ОК | ОК | ОК |
| Samsung Galaxy Watch | ОК | ОК | НЕТ | ОК | ОК | ОК | ОК | ОК |
| Huawei Watch D2 | ОК | ОК | ОК | ОК | ОК | ОК | ОК | ОК |
| Amazfit T-Rex 3 | ОК | НЕТ | НЕТ | ОК | НЕТ | ОК | ОК | ОК |

## Генерация реалистичных данных

Эмулятор генерирует биометрические данные, основанные на физиологических моделях:

### Пульс (Heart Rate)
- Базовое значение: 72 BPM
- Вариации: активность (+40 BPM), стресс (±15 BPM), циркадный ритм (±5 BPM)
- Фитнес-уровень снижает пульс в покое (до -10 BPM)
- Диапазон: 40-200 BPM

### Вариабельность сердечного ритма (HRV)
- Базовое значение: 50 мс
- Стресс снижает HRV (до -20 мс)
- Качество сна повышает HRV (до +10 мс)
- Диапазон: 10-100 мс

### Сатурация крови (SpO₂)
- Базовое значение: 98%
- Небольшой гауссовский шум (±0.5%)
- Диапазон: 90-100%

### Температура тела
- Базовое значение: 36.6°C
- Циркадный ритм (±0.3°C)
- Диапазон: 35.5-38.5°C

### Артериальное давление (только Huawei Watch D2)
- Базовое: 120/80 мм рт.ст.
- Стресс повышает давление (+15/+10)
- Активность повышает систолическое (+20)
- Диапазон: 80-200 / 50-130 мм рт.ст.

### Фазы сна
- Глубокий сон (deep): 10% времени
- Лёгкий сон (light): 40% времени
- REM фаза: 20% времени
- Бодрствование: 30% времени

### Шаги
- Зависят от времени суток (пик днём)
- Зависят от уровня активности пользователя
- Диапазон: 0-1500 шагов за интервал

## Запуск

### Локально

```bash
go run ./cmd/device-emulator \
  --user-id=user-123 \
  --device-type=apple_watch \
  --connector-url=http://localhost:8082 \
  --sync-interval=30s
```

### Docker

```bash
docker build -t device-emulator -f cmd/device-emulator/Dockerfile .
docker run device-emulator \
  --user-id=user-123 \
  --device-type=huawei_watch_d2 \
  --connector-url=http://device-connector:8082
```

### Docker Compose

Добавьте в `docker-compose.yml`:

```yaml
device-emulator:
  build:
    context: .
    dockerfile: cmd/device-emulator/Dockerfile
  environment:
    - USER_ID=user-123
    - DEVICE_TYPE=apple_watch
    - CONNECTOR_URL=http://device-connector:8082
    - SYNC_INTERVAL=30s
  depends_on:
    - device-connector
  restart: unless-stopped
```

## Параметры запуска

| Флаг | Описание | По умолчанию |
|------|----------|--------------|
| `--user-id` | ID пользователя (обязательно) | - |
| `--device-type` | Тип устройства | `apple_watch` |
| `--connector-url` | URL device-connector | `http://localhost:8082` |
| `--sync-interval` | Интервал синхронизации | `30s` |
| `--auto-register` | Автоматическая регистрация | `true` |

## API интеграция

### Реальные API (заглушки)

Эмулятор включает заглушки для реальных API производителей устройств:

- **Apple HealthKit API** — для Apple Watch
- **Samsung Health API** — для Samsung Galaxy Watch
- **Huawei Health Kit API** — для Huawei Watch D2
- **Zepp API** — для Amazfit

Когда реальные API станут доступны, замените заглушки на реальные HTTP запросы в файлах:
- `internal/wearableemulator/emulator.go` (классы `*HealthClient`)

### Формат данных

Эмулятор отправляет данные в формате, идентичном реальным устройствам:

```json
{
  "device_type": "apple_watch",
  "device_token": "token-123",
  "sync_interval_ms": 30000,
  "records": [
    {
      "device_type": "apple_watch",
      "metric_type": "heart_rate",
      "value": 78.5,
      "timestamp": "2026-04-09T10:30:00Z",
      "quality": "good"
    }
  ]
}
```

## Настройка физиологического состояния

Для изменения базовых параметров пользователя отправьте POST запрос на эмулятор:

```bash
curl -X POST http://localhost:8080/api/v1/emulator/state \
  -H "Content-Type: application/json" \
  -d '{
    "base_heart_rate": 65,
    "age": 35,
    "weight": 80,
    "height": 180,
    "fitness_level": 0.8
  }'
```

## Мониторинг

Эмулятор логирует каждую синхронизацию:

```
2026-04-09T10:30:00Z INFO Synced 8 samples: 8 forwarded, 0 duplicates, 0 failed
```

## Troubleshooting

### "Failed to register device"
- Убедитесь, что device-connector запущен и доступен
- Проверьте URL в параметре `--connector-url`

### "Sync failed"
- Проверьте, что устройство зарегистрировано
- Убедитесь, что token корректный

### Данные не меняются
- Эмулятор использует случайные значения, но они основаны на базовом состоянии
- Измените состояние через API `/api/v1/emulator/state`
