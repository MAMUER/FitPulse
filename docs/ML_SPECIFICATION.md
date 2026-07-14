# FitPulse — ML/GAN Спецификация

## Обзор

FitPulse использует два компонента:

1. **Classifier** — правила-на-основе классификация состояния пользователя, реализованная на Go. Работает только с биометрическими данными с носимых устройств.
2. **Generator (Conditional Diffusion Model)** — генерация индивидуальных тренировочных планов с учётом полного профиля пользователя. Реализован на Python (PyTorch + Lightning, inference через ONNX Runtime).

Генерация плана использует 3-tier fallback:
1. **Primary**: Conditional Diffusion Model (DDPM) с 32-dim условным вектором
2. **Fallback 1**: Rule-based генерация на основе шаблонов по классу состояния
3. **Fallback 2**: Статический beginner-план

---

## Архитектура моделей

### Classifier (Go-правила, 6 классов)

**Назначение:** Определение состояния пользователя по биометрическим данным с браслета.

**Входные признаки (7 признаков):**

| # | Признак | Тип | Единица | Примечание |
|---|---------|-----|---------|------------|
| 1 | `heart_rate` | float64 | уд/мин | Текущий или средний за последний час |
| 2 | `heart_rate_variability` | float64 | мс | Heart Rate Variability (RMSSD) |
| 3 | `spo2` | float64 | % | 70–100 |
| 4 | `temperature` | float64 | °C | 35.5–38.5; ключевой признак для класса "Заболевание" |
| 5 | `blood_pressure_systolic` | float64 | мм рт.ст. | 80–200 |
| 6 | `blood_pressure_diastolic` | float64 | мм рт.ст. | 50–130 |
| 7 | `sleep_hours` | float64 | часы | 0–24 |

**Дополнительный контекст (не используется в классификации, передаётся в Generator):**
- Менструальные данные (`GET /health/menstrual-cycles`)
- Заболевания (`GET /health/conditions`)
- История тренировок (`GET /training/plans`, `POST /training/complete`)
- Состав тела (`GET /health/body-composition`)

**Источники данных:**
- `GET /api/v1/biometrics?metric_type=heart_rate,hrv,spo2,temperature,blood_pressure,sleep_hours`
- `GET /api/v1/health/menstrual-cycles`
- `GET /api/v1/health/conditions`
- `GET /api/v1/training/plans?status=active` + `POST /api/v1/training/complete` история
- `GET /api/v1/devices` — проверка последнего ingestion

**Выход:**

```json
{
  "predicted_class": "recovery",
  "predicted_class_ru": "Восстановление",
  "confidence": 0.87,
  "probabilities": {
    "recovery": 0.87,
    "endurance_basic": 0.08,
    "endurance_threshold": 0.03,
    "power_hiit": 0.01,
    "overtraining": 0.01,
    "illness": 0.0
  },
  "description": "Низкая нагрузка + высокий HRV + хорошее восстановление",
  "hr_range": "50-65% HRmax",
  "recommendations": [
    "Лёгкая активность (ходьба, йога)",
    "Растяжка и мобилизация",
    "Плавание в лёгком темпе"
  ],
  "personalized_notes": "Учитывая фолликулярную фазу, рекомендуется избегать высокоинтенсивных нагрузок до овуляции."
}
```

**6 классов (имена как в коде):**

| # | Класс (slug) | Название RU | Ключевые правила |
|---|--------------|-------------|------------------|
| 1 | `recovery` | Восстановление | HRV > 80 И (HR < 60% HRmax Или sleep > 8) |
| 2 | `endurance_basic` | Базовая выносливость E1-E2 | HRV 50–80, HR 65–80% HRmax, sleep 6–8 |
| 3 | `endurance_threshold` | Пороговая выносливость E3 | HRV 40–50, HR 80–90% HRmax |
| 4 | `power_hiit` | Силовая/HIIT | HRV > 60, HR > 90% HRmax, sleep > 7 |
| 5 | `overtraining` | Перетренированность | HRV < 30 И HR < 60% HRmax |
| 6 | `illness` | Заболевание | Температура > 37.5°C Или HRV < 30 с признаками болезни |

**Endpoint:** `POST /classify` (service на порту 8001, gateway: `POST /api/v1/ml/classify`)

---

### Generator (Conditional Diffusion Model — DDPM)

**Входные данные (условная генерация):**

DDPM получает 32-dim conditional vector, построенный из полного профиля пользователя:

| Index | Feature | Range | Описание |
|-------|---------|-------|----------|
| 0 | age_normalized | 0–1 | (age - 18) / (100 - 18) |
| 1 | bmi_normalized | 0–1 | (bmi - 15) / (40 - 15) |
| 2 | fitness_level | 0–1 | beginner=0.0, intermediate=0.5, advanced=1.0 |
| 3 | goal_strength | 0–1 | one-hot: muscle_gain |
| 4 | goal_endurance | 0–1 | one-hot: endurance |
| 5 | goal_weight_loss | 0–1 | one-hot: weight_loss |
| 6 | goal_flexibility | 0–1 | one-hot: flexibility |
| 7 | health_factor | 0–1 | 1.0 - classifier confidence |
| 8 | menstrual_phase_luteal | 0–1 | 1.0 если luteal |
| 9 | menstrual_phase_menstruation | 0–1 | 1.0 если menstruation |
| 10 | menstrual_phase_ovulation | 0–1 | 1.0 если ovulation |
| 11 | active_conditions_count_normalized | 0–1 | min(conditions, 5) / 5 |
| 12 | has_contraindications | 0–1 | 1.0 если есть противопоказания |
| 13 | has_allergies | 0–1 | 1.0 если есть аллергии |
| 14 | recovery_needed | 0–1 | 1.0 если classifier=recovery/overtraining |
| 15 | days_since_last_workout | 0–1 | min(days, 7) / 7 |
| 16 | workout_frequency | 0–1 | completed_workouts / 30 |
| 17 | sleep_quality | 0–1 | sleep_hours / 9 |
| 18 | hrv_factor | 0–1 | hrv / 100 |
| 19 | temperature_normalized | 0–1 | (temp - 35.5) / (38.5 - 35.5) |
| 20 | spo2_factor | 0–1 | spo2 / 100 |
| 21 | available_days_count | 0–1 | count / 7 |
| 22 | preferred_morning | 0–1 | 1.0 если preferred_time=morning |
| 23 | preferred_evening | 0–1 | 1.0 если preferred_time=evening |
| 24 | equipment_dumbbell | 0–1 | 1.0 если есть |
| 25 | equipment_resistance_band | 0–1 | 1.0 если есть |
| 26 | equipment_barbell | 0–1 | 1.0 если есть |
| 27 | equipment_none | 0–1 | 1.0 если нет оборудования |
| 28–31 | reserved | 0–1 | Зарезервировано |

**Architecture:**
- **Framework:** PyTorch 2.5+ + Lightning
- **Model type:** Conditional Diffusion Model (DDPM)
- **Noise predictor:** `plan_dim + condition_dim + 1 (time) → 512 → 512 → plan_dim`
- **Input:** x_t (19-dim noisy plan) + t (timestep) + condition (32-dim)
- **Output:** noise prediction (19-dim)
- **Sampling:** 50-step DDPM reverse process
- **Model file:** `models/generator.onnx` (ONNX Runtime для inference)
- **Fallback:** rule-based → static beginner plan

**Plan Features (19 dimensions):**

| Index | Feature | Range | Условная логика |
|-------|---------|-------|-----------------|
| 0 | duration_minutes | 0–100 | Recovery: 20–40; Overtraining/Illness: 0; Power: 60–100 |
| 1 | intensity_level | 0–1 | Menstruation: -0.2; Illness: 0.0; Recovery: 0.3 |
| 2 | rest_ratio | 0–1 | Overtraining: 0.5–0.7; HIIT: 0.3–0.4 |
| 3 | weekly_frequency | 0–7 | Illness: 0; Recovery: 2–3; Normal: 3–5 |
| 4–11 | equipment_tools | 8 values | На основе `equipment` из preferences |
| 12 | warmup_ratio | 0–1 | Menstruation: +0.1; Illness: +0.2; age>60: +0.1 |
| 13 | cooldown_ratio | 0–1 | Всегда >= 0.1; Recovery: +0.1; sleep<6: +0.1 |
| 14 | age_factor | 0–1 | age > 50: -0.2; age < 25: +0.1 |
| 15 | fitness_factor | 0–1 | beginner: -0.2; advanced: +0.2 |
| 16 | health_factor | 0–1 | active_conditions > 0: -0.3; illness: 0.0 |
| 17 | goal_strength | 0–1 | goal=muscle_gain: 0.8–1.0 |
| 18 | goal_endurance | 0–1 | goal=endurance: 0.8–1.0 |

**Условная пост-обработка (правила после DDPM):**

1. **Менструация**: если `menstrual_phase == menstruation`, снизить `intensity_level` на 20%, увеличить `warmup_ratio` и `cooldown_ratio` на 10%.
2. **Беременность**: если в `contraindications` есть "pregnancy", `intensity_level` = 0.3–0.5, исключить упражнения на живот.
3. **Заболевание**: если `classifier.predicted_class == illness`, `duration_minutes` = 0, `intensity_level` = 0.0.
4. **Перетренированность**: если `classifier.predicted_class == overtraining`, снизить `weekly_frequency` на 30%, увеличить `rest_ratio`.
5. **Противопоказания**: если в `contraindications` есть "спина", исключить упражнения с осевой нагрузкой; если "колени", исключить прыжки.
6. **Аллергии**: если в `allergies` есть "латекс", исключить оборудование с латексными ремешками.
7. **Возраст**: если age > 60, снизить `intensity_level` на 20%, увеличить `warmup_ratio` до 0.3.
8. **BMI**: если BMI > 35, снизить интенсивность, исключить прыжки и бег.
9. **Recovery**: если `classifier.predicted_class == recovery`, `duration_minutes` = 20–30, `intensity_level` = 0.3–0.4.
10. **Sleep**: если `sleep_hours < 6`, снизить `intensity_level` на 15%, увеличить `cooldown_ratio`.

**Fallback chain:**

1. **Primary:** DDPM-генерация с 32-dim conditional vector (50-step sampling)
2. **Fallback 1:** Rule-based генерация на основе шаблонов по классу состояния (`build_rule_based_plan`)
3. **Fallback 2:** Статический beginner-план (`build_static_beginner_plan`)

---

## Generator Details

- **Input:** 19-dimensional noisy plan (x_t) + 32-dimensional conditional vector + timestep
- **Output:** 19-dimensional plan vector (normalized 0-1)
- **Model location:** `models/generator.onnx`
- **Inference:** ONNX Runtime (CPUExecutionProvider)
- **Fallback:** rule-based generator + static beginner plan

**Installation:**

```bash
pip install -r cmd/ml_generator/requirements.txt
```

**Usage:**

#### Загрузка модели при старте FastAPI (global scope)

```python
from cmd.ml_generator.main import load_generator, app

@app.on_event("startup")
async def startup():
    await load_generator()
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
    "gender": "male",
    "weight": 70.0,
    "height": 170.0,
    "fitness_level": "intermediate",
    "goals": ["endurance"],
    "allergies": [],
    "contraindications": []
  },
  "health_status": {
    "predicted_class": "endurance_basic",
    "confidence": 0.87,
    "hrv": 65.0,
    "sleep_hours": 7.5,
    "active_conditions_count": 0,
    "menstrual_phase": "follicular",
    "day_of_cycle": 8,
    "cycle_length": 28,
    "body_composition": {
      "bmi": 24.2,
      "body_fat_pct": 18.5,
      "muscle_mass_kg": 55.0
    }
  },
  "training_history": {
    "completed_workouts_count": 12,
    "avg_intensity": 0.6,
    "last_workout_date": "2024-01-15T08:00:00Z"
  },
  "preferences": {
    "time": "morning",
    "equipment": ["dumbbell", "resistance_band"],
    "available_days": ["mon", "wed", "fri"]
  },
  "constraints": {
    "duration_weeks": 4,
    "max_sessions_per_week": 4
  }
}
```

**Response:**

```json
{
  "plan_vector": [45, 0.6, 0.4, 3, 1, 0, 0, 0, 0, 0, 0, 0, 0.15, 0.15, 0.5, 0.5, 0.7, 0.2, 0.8],
  "plan_metadata": {
    "training_class": "endurance_basic",
    "model_version": "diffusion_v1",
    "vector_length": 19
  }
}
```

**TRAINING_TEMPLATES (6 классов):**

| Класс | duration_range | intensity_range | exercises | rest_ratio |
|-------|----------------|-----------------|-----------|------------|
| recovery | 20–45 | 0.3–0.5 | walking, yoga, stretching, light_swimming, mobility | 0.7 |
| endurance_basic | 45–90 | 0.5–0.7 | running, cycling, swimming, rowing, hiking | 0.4 |
| endurance_threshold | 30–60 | 0.7–0.85 | tempo_run, threshold_intervals, fartlek, critical_power | 0.3 |
| power_hiit | 20–45 | 0.85–1.0 | hiit, strength, sprints, crossfit, plyometrics | 0.5 |
| overtraining | 0–20 | 0.0–0.3 | rest, walking, stretching, yoga, mobility | 0.8 |
| illness | 0–0 | 0.0–0.0 | rest | 1.0 |

---

## Требования к GPU и обучению

|Параметр|Значение|
|---|---|
|Минимум GPU|CUDA-capable, 4GB VRAM|
|Рекомендуется|8GB+ VRAM (RTX 3070+)|
|Данные для обучения|Исторические планы + биометрия + фидбек пользователей + менструальные циклы + заболевания + состав тела|
|Частота переобучения|Раз в 2 недели (incremental)|
|Валидация|Hold-out 20%, метрики: val_loss (MSE noise prediction)|
|Версионирование|DVC для данных и моделей (`datasets/`, `models/`)|

---

## Интеграция

### Classifier

**Endpoint:** `POST /classify` (service на порту 8001, gateway: `POST /api/v1/ml/classify`)

- **Вход:** физиологические данные
  ```json
  {
    "physiological_data": {
      "heart_rate": 72.0,
      "heart_rate_variability": 65.0,
      "spo2": 98.0,
      "temperature": 36.6,
      "blood_pressure_systolic": 120.0,
      "blood_pressure_diastolic": 80.0,
      "sleep_hours": 7.5
    },
    "user_profile": {
      "age": 30,
      "gender": "male",
      "fitness_level": "intermediate",
      "health_conditions": ["гипертония"],
      "goals": ["endurance"]
    }
  }
  ```
- **Выход:** класс, уверенность, вероятности, рекомендации, персонализированные заметки

### Generator

**Endpoint:** `POST /generate-plan` (service на порту 8002, gateway: `POST /api/v1/ml/generate-plan`)

- **Вход:** полный профиль пользователя + контекст здоровья + история тренировок
- **Выход:** 19-dim plan vector + metadata
- **Fallback:** rule-based → static beginner

---

## Безопасность и ограничения

1. **Medical disclaimer**: Все планы носят рекомендательный характер. При заболеваниях/травмах — консультация врача обязательна.
2. **Privacy**: Персональные данные пользователя (менструальные циклы, заболевания) не используются для обучения моделей без явного согласия.
3. **Bias mitigation**: Модель обучается на разнообразных данных (пол, возраст, уровень подготовки), чтобы избежать генерации планов, не подходящих для特定ных групп.
4. **Audit**: Каждый generated plan содержит `model_version` для отслеживания версий моделей и аудита.

---

## Open Questions / TODO

- [ ] Переобучить модель с CONDITION_DIM=32 и сохранить новый `generator.onnx`
- [ ] Реализовать DDIM sampling для ускорения inference (< 10 шагов)
- [ ] Добавить A/B тестирование планов (качество планов vs фидбек пользователей)
- [ ] Реализовать incremental training с DVC pipeline (см. `docs/phase2-roadmap.md` раздел 16)
- [ ] Добавить валидацию plan_vector (диапазоны, суммы) перед возвратом клиенту
