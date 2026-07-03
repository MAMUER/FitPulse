# FitPulse — ML/GAN Спецификация

## Обзор

FitPulse использует два компонента:

1. **Classifier** — правила-на-основе классификации состояния пользователя, реализованный на Go.
2. **Generator (GAN)** — генерация индивидуальных тренировочных планов (обученная модель `models/generator.keras`).

## Архитектура моделей

### Classifier (Go-правила, 6 классов)

**Входные признаки (7 признаков):**

- HeartRate (float64)
- HeartRateVariability (float64)
- SpO2 (float64)
- Temperature (float64) — ключевой признак для класса "Заболевание"
- BloodPressureSystolic (float64)
- BloodPressureDiastolic (float64)
- SleepHours (float64)

**Выход:**

- PredictedClass (String)
- PredictedClassRu (String)
- Confidence (0–1)
- Probabilities (map[string]float64)
- Description (String)
- HrRange (String)
- Recommendations ([]string)
- PersonalizedNotes (*string)

**6 классов:**

1. Восстановление (Recovery)
2. Базовая выносливость E1-E2 (Endurance Basic)
3. Пороговая выносливость E3 (Endurance Threshold)
4. Силовая/HIIT (Power/HIIT)
5. Перетренированность (Overtraining) — определя через HRV < 30 и низкую частоту пульса
6. Заболевание (Illness) — определя через температуру > 37.5°C

**Endpoint:** `POST /classify`

### Generator (Conditional GAN)

**Generator:**
- Вход: latent vector (64-dim)
- Архитектура: 64 → 256 → 512 → 256 → 19 (sigmoid), BatchNorm, Dropout
- Loss: MSE
- Выход: 19-dim plan vector

**Discriminator:**
- Вход: 19-dim plan vector
- Архитектура: 19 → 512 → 256 → 128 → 1 (sigmoid)
- Loss: binary_crossentropy

### Generator Details

- **Input:** 64-dimensional noise vector
- **Output:** 19-dimensional plan vector (normalized 0-1)
- **Model location:** `models/generator.keras`

**Installation:**

```bash
pip install -r cmd/ml_generator/requirements.txt
```

**Usage:**

#### Загрузка модели (`tf.keras.models.load_model`) внутри обработчика запроса или при каждом вызове `predict` — критическая ошибка производительности (I/O операция + инициализация графа TF занимает сотни миллисекунд). Модель должна загружаться один раз при старте сервиса (global scope) и переиспользоваться

```python
# Best Practice: Загрузка модели на уровне модуля при старте FastAPI
import tensorflow as tf
from fastapi import FastAPI

model = tf.keras.models.load_model('models/generator.keras')
app = FastAPI()

@app.post("/ml/generate-plan")
async def generate_plan(payload: dict):
    plan = model.predict(np.random.randn(1, 64), verbose=0)[0]
    return {"plan": plan.tolist()}
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

`POST /generate-plan` with:

```json
{
  "training_class": "endurance_e3",
  "user_profile": {
    "age": 30,
    "fitness_level": "intermediate",
    "weight": 70,
    "height": 170,
    "gender": "male"
  },
  "preferences": {
    "time": "morning",
    "equipment": ["dumbbell", "resistance_band"]
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

**Endpoint:** `POST /classify` (service на порту 8001)

- Вход: физиологические данные (heart_rate, hrv, spo2, temperature, blood_pressure, sleep_hours)
- Выход: класс, уверенность, рекомендации

**Endpoint:** `POST /generate-plan` (service на порту 8002)

- Вход: user_id, цель, ограничения
- Выход: тренировочный план
