# FitPulse — ML/GAN Спецификация

## Обзор

FitPulse использует два компонента:

1. **Classifier** — алгоритм классификации состояния пользователя (6 классов), реализованный на Go.
2. **Generator (GAN)** — генерация индивидуальных тренировочных планов и диеты

## Архитектура моделей

### Classifier (Go-алгоритм, 6 классов)

**Входные признаки (64-dim):**

- HR (средний, min, max, вариабельность): 4 признака
- HRV (средний, RMSSD, pNN50): 3 признака
- SpO₂ (средний, мин): 2 признака
- Температура (средняя, пик): 2 признака
- Артериальное давление (систолическое, диастолическое): 2 признака
- Сон (длительность, deep sleep, REM): 3 признака
- Шаги (сумма, активные периоды): 2 признака
- Время суток (час, день недели): 2 признаки
- Возраст, пол, рост, вес, опыт тренировок: 5 признаков
- Неделя плана (прогресс): 1 признак
- Историческая нагрузка (7д/14д/30д): 3 признака

**Выход (19-dim):**

- Вероятности 6 классов состояния (softmax)
- Уверенность (max probability, 0–1)
- Рекомендация по интенсивности (0–1)
- Рекомендация по типу тренировки (one-hot 5: cardio, strength, hiit, yoga, rest)
- Длительность рекомендованной тренировки (мин)
- Сдвиг плана (days: -3..+3)
- Флаг «обратиться к врачу» (binary)
- Уровень усталости (fatigue_level, 0–1)
- Мотивация (motivation_score, 0–1)
- Качество восстановления (recovery_quality, 0–1)

**6 классов:**

1. Восстановление (Recovery)
2. Базовая выносливость E1-E2 (Endurance Basic)
3. Пороговая выносливость E3 (Endurance Threshold)
4. Силовая/HIIT (Power/HIIT)
5. Перетренированность (Overtraining)
6. Заболевание (Illness)

### Generator (Conditional GAN)

**Generator:**

- Вход: latent vector (64-dim) + conditioning vector (19-dim, output Classifier)
- Архитектура: 3 Dense-блока (256→512→1024→19), LeakyReLU, BatchNorm
- Выход: 19-dim plan vector

**Discriminator:**

- Вход: 19-dim plan vector + 19-dim conditioning vector
- Архитектура: Dense 256→128→1, sigmoid

**Loss:**

- adversarial loss (binary crossentropy)
- L1 distance between generated plan и средний план для класса
- penalty за выход за пределы безопасных диапазонов

### Generator Details

- **Input:** 64-dimensional noise vector
- **Output:** 19-dimensional plan vector (normalized 0-1)
- **Model location:** `models/generator.keras`

**Installation:**

```bash
pip install -r cmd/ml_generator/requirements.txt
```

**Usage:**

Direct generation:

```python
import tensorflow as tf
import numpy as np

model = tf.keras.models.load_model('models/generator.keras')
plan = model.predict(np.random.randn(1, 64), verbose=0)[0]
```

Via `TrainingPlanGAN`:

```python
from cmd.ml_generator.train_gan import TrainingPlanGAN

gan = TrainingPlanGAN(latent_dim=64, plan_dim=19)
plan = gan.generate_plan(seed=42)
plan_dict = gan.generate_plan_dict(seed=42)
```

Via API:

```bash
uvicorn cmd.ml_generator.main:app --host 0.0.0.0 --port 8002
```

`POST /ml/generate-plan` with:

```json
{
  "training_class": "endurance_e3",
  "user_profile": {
    "age": 30,
    "fitness_level": "intermediate"
  }
}
```

**Plan Features (19 dimensions):**

| Index | Feature | Range |
| ------- | --------- | ------- |
| 0 | duration_minutes | 0-100 |
| 1 | intensity_level | 0-1 |
| 2 | rest_ratio | 0-1 |
| 3 | weekly_frequency | 0-7 |
| 4-11 | equipment_tools | 8 values |
| 12 | warmup_ratio | 0-1 |
| 13 | cooldown_ratio | 0-1 |
| 14 | age_factor | 0-1 |
| 15 | fitness_factor | 0-1 |
| 16 | health_factor | 0-1 |
| 17 | goal_strength | 0-1 |
| 18 | goal_endurance | 0-1 |

## Требования к GPU и обучению

|Параметр|Значение|
|---|---|
|Минимум GPU|CUDA-capable, 4GB VRAM|
|Рекомендуется|8GB+ VRAM (RTX 3070+)|
|Данные для обучения|Исторические планы + биометрия + фидбек пользователей|
|Частота переобучения|Раз в 2 недели (incremental)|
|Валидация|Hold-out 20%, метрики: accuracy, F1, confusion matrix|

## Интеграция

**Endpoint:** `POST /ml/classify`

- Вход: текущая биометрия + контекст
- Выход: класс, уверенность, рекомендации

**Endpoint:** `POST /ml/generate-plan`

- Вход: пользовательский профиль + цель + ограничения
- Выход: тренировочный план + диетический план

**Fallback:** если ML-сервис недоступен — используется rule-based движок.
