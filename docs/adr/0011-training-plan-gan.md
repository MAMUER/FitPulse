# ADR 0011: Генерация тренировочных планов с использованием GAN

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

- `cmd/ml_generator/preprocess_exercises.py` — предобработка данных;
- `cmd/ml_generator/train_gan.py` — скрипт обучения GAN;
- `cmd/ml_generator/main.py` — FastAPI сервис с упрощённой генерацией;
- `models/generator.keras` — обученная модель (1.2MB).

## Использование

```python
import tensorflow as tf
import numpy as np

model = tf.keras.models.load_model('models/generator.keras')
plan = model.predict(np.random.randn(1, 64), verbose=0)[0]
```
