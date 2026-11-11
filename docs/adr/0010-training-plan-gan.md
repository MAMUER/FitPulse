# ADR 0010: Генерация тренировочных планов с использованием GAN

## Статус

Утверждено

## Контекст

ML Generator service должен производить персонализированные тренировочные планы на основе профилей пользователей и биометрических данных.

Требования:

- генерировать 19-мерные векторы тренировочных планов;
- поддерживать 10,000+ тренировочных семплов;
- обучаться в Docker-контейнере;
- использовать Keras 3 с TensorFlow backend;
- интегрироваться в `training-service` через HTTP API.

## Решение

Реализован Conditional GAN-based генератор тренировочных планов со следующей архитектурой:

### Компоненты

1. **gan-service** (`cmd/gan-service/main.go`) — HTTP-сервис на Go, который вызывает локальный Python-скрипт `cmd/ml_generator/gan_train.py generate` с загруженной моделью `models/gan_generator.pt`. Экспонирует endpoint `POST /gan/generate`.

2. **training-service** (`cmd/training-service/main.go`) — вызывает `gan-service` через переменную `GAN_SERVICE_URL` (в K8s: `http://ml-generator:8002`). Классификатор состояния определяет класс бустинга, который передается как условие в GAN.

3. **Модель** — `models/gan_generator.pt`, Conditional GAN с генератором `R^100 × R^12 → R^19` (latent 100 + класс + 5 демографических признаков → 19 признаков плана).

4. **Датысет** — `datasets/processed/training_plans.csv` (25 колонок: `p0–p18`, `class`, `age`, `gender`, `weight`, `height`, `goal`), 300 строк, сбалансирован по 6 классам состояния.

### Фоллбек

Если GAN-сервис недоступен, `training-service` использует Go template-based движок (`internal/planner/engine.go` + `internal/templates/training_templates.go`) с 6 хардкодными шаблонами.

## Архитектура модели

- Generator: 64-dim latent → 256 → 512 → 256 → 19 (sigmoid)
- Discriminator: 19 → 512 → 256 → 128 → 1 (sigmoid)
- Loss: MSE для generator, binary_crossentropy для discriminator
- Optimizer: Adam (lr=0.0002, beta_1=0.5)
- Условие: one_hot(класс состояния, 7 классов) + 5 демографических признака (возраст, пол, вес, рост, цель)

## Конфигурация обучения

- 500 epochs, batch size 64
- Docker-based обучение с TensorFlow backend
- Модель сохранена в `models/gan_generator.pt`

## Интеграция в training-service

В `cmd/training-service/main.go`:

1. `classToGANIndex(classification)` — маппинг 7 классов состояния на индекс GAN (0–5).
2. `loadUserProfile(ctx, userID)` — загрузка демографии из PostgreSQL (возраст, пол, вес, рост, цель).
3. `callGANService(classIndex, profile, numSamples)` — HTTP POST на `gan-service /gan/generate`.
4. `vectorToGANParams(vec)` — преобразование 19-мерного вектора GAN в параметры плана (интенсивность, длительность, частота).
5. `resolveGANParams(ctx, userID, classificationClass)` — обёртка с фоллбеком на шаблоны.
6. Ключ `GAN_SERVICE_URL` добавлен в `configs/k8s/base/deployments/training-service.yaml` (`http://ml-generator:8002`).

## Последствия

- **Плюсы**: персонализированные планы, адаптированные под класс состояния и демографию пользователя;
- **Плюсы**: совместимость с Keras 3 обеспечивает future-proof реализацию;
- **Плюсы**: 19-мерный выход поддерживает богатые признаки тренировочного плана;
- **Плюсы**: фоллбек на Go-шаблоны обеспечивает отказоустойчивость;
- **Нейтрально**: обучение требует ~5 минут на CPU;
- **Нейтрально**: модель генерирует нормализованные векторы (0-1), требующие пост-обработки;
- **Минусы**: дополнительная сетевая зависимость между training-service и gan-service;
- **Минусы**: поддержка двух кодовых путей (GAN + шаблоны).

## Реализация

- `cmd/training-service/main.go` — интеграция GAN, метод `resolveGANParams`, `callGANService`, `loadUserProfile`;
- `cmd/gan-service/main.go` — HTTP-сервис, вызов Python GAN;
- `cmd/ml_generator/gan_model.py` — Conditional GAN модель;
- `cmd/ml_generator/gan_train.py` — скрипт обучения и генерации;
- `cmd/ml_generator/gan_train_test.py` — 20+ тестов с моком torch;
- `cmd/training-service/gan_integration_test.go` — 14 юнит-тестов интеграции;
- `datasets/processed/training_plans.csv` — обучающий датысет (300 строк);
- `internal/templates/training_templates.go` — шаблоны тренировок (фоллбек);
- `internal/planner/engine.go` — движок генерации планов на основе шаблонов (фоллбек).

## Использование

```python
import tensorflow as tf
import numpy as np

model = tf.keras.models.load_model('models/generator.keras')
plan = model.predict(np.random.randn(1, 64), verbose=0)[0]
```

```go
// В training-service
ganParams := s.resolveGANParams(ctx, userID, classificationClass)
// ganParams.Intensity, ganParams.DurationMinutes, ganParams.SessionsPerWeek
```

## Альтернативы

1. **Ml-generator, но переписать на Go/ONNX**: отклонено — всё равно операционный overhead.
2. **Объединить ml-generator с classifier service**: отклонено — classifier должен оставаться single-responsibility.
3. **Только template-based (без GAN)**: отклонено — ограниченное разнообразие планов (6 шаблонов), нет персонализации под класс состояния.
