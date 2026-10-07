# ADR 0010: Генерация тренировочных планов с использованием GAN

## Статус

Принято

## Контекст

ML Generator service должен был производить персонализированные тренировочные планы на основе профилей пользователей и биометрических данных.

Требования:

- генерировать 19-мерные векторы тренировочных планов;
- поддерживать 10,000+ тренировочных семплов;
- обучаться в Docker-контейнере;
- использовать Keras 3 с TensorFlow backend.

## Решение

Реализован GAN-based генератор тренировочных планов со следующей архитектурой:

1. **Предобработка данных**
   - конвертация данных об упражнениях из `datasets/raw/exercisedb` в векторы тренировочных планов;
   - генерация 10,000 тренировочных планов с 19 признаками;
   - признаки: duration, intensity, rest_ratio, weekly_freq, equipment (8 dims), warmup, cooldown, progression, age/fitness/health/goal факторы.

2. **Архитектура модели**
   - Generator: 64-dim latent → 256 → 512 → 256 → 19 (sigmoid)
   - Discriminator: 19 → 512 → 256 → 128 → 1 (sigmoid)
   - Loss: MSE для generator, binary_crossentropy для discriminator
   - Optimizer: Adam (lr=0.0002, beta_1=0.5)

3. **Конфигурация обучения**
   - 500 epochs, batch size 64
   - Docker-based обучение с TensorFlow backend
   - Модель сохранена в `models/generator.keras`

## Последствия

- **Плюсы**: использованы реальные данные об упражнениях для обучения;
- **Плюсы**: совместимость с Keras 3 обеспечивает future-proof реализацию;
- **Плюсы**: 19-мерный выход поддерживает богатые признаки тренировочного плана;
- **Нейтрально**: обучение требует ~5 минут на CPU;
- **Нейтрально**: модель генерирует нормализованные векторы (0-1), требующие пост-обработки.

## Реализация

- `internal/templates/training_templates.go` — шаблоны тренировок;
- `internal/planner/engine.go` — планировщик планов на основе шаблонов.

## Использование

```python
import tensorflow as tf
import numpy as np

model = tf.keras.models.load_model('models/generator.keras')
plan = model.predict(np.random.randn(1, 64), verbose=0)[0]
```

## Выбор template-based генерации планов на Go

### Контекст выбора Go engine

На старте разработки FitPulse требовалось выбрать подход к генерации тренировочных планов

### Решение по Go engine

На старте выбран **Go template-based engine**:

1. **Training templates** размещены в `internal/templates/training_templates.go` как Go-константы (6 шаблонов).
2. **Plan engine** реализован в `internal/planner/engine.go` с модификаторами (age, BMI, contraindications, menstruation, biometrics).
3. **Survey storage** использует JSONB в `user_profiles.survey_data` с `survey_completed` и `survey_completed_at`.
4. **Classifier service** (`cmd/classifier`, порт 8001) остаётся отдельным сервисом для классификации состояния.
5. **Rule-based chat** реализует FAQ-матчинг в `internal/chat/faq.go`.

### Последствия выбора Go engine

- **Плюсы**: упрощённый deployment (на один сервис меньше);
- **Плюсы**: меньшая задержка (генерация происходит in-process в `training-service`);
- **Плюсы**: легче тестировать (Go-шаблоны и engine можно unit-тестировать без сетевых моков);
- **Плюсы**: сокращение времени CI/CD (нет Python линтинга, pip-compile, сериализации моделей);
- **Минусы**: ограниченное разнообразие планов (6 hardcoded шаблонов);
- **Минусы**: нет GPU-инференса сложных генеративных моделей;
- **Минусы**: при необходимости смены стратегии генерации требуется передеploy Go-сервисов.

### Реализация Go engine

- `internal/templates/training_templates.go` — 6 шаблонов тренировок на Go;
- `internal/planner/engine.go` — движок генерации планов с модификаторами;
- `internal/chat/faq.go` — FAQ база и матчинг;
- `cmd/gateway/handlers_survey.go` — HTTP endpoints для анкеты;
- `cmd/gateway/handlers_chat.go` — HTTP endpoint для чата;
- `cmd/gateway/handlers_ml_feedback.go` — HTTP endpoint для обратной связи по ML;
- `cmd/training-service/main.go` — импортирует planner package;
- `docs/ML_SPECIFICATION.md` — обновлён под Go-based архитектуру.

### Альтернативы для Go engine

1. **Ml-generator, но переписать на Go/ONNX**: отклонено — всё равно операционный overhead.
2. **Объединить ml-generator с classifier service**: отклонено — classifier должен оставаться single-responsibility.
3. **Ml-generator как есть**: отклонено — maintenance burden слишком высок для template-based выхода.
