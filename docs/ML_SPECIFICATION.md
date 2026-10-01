# FitPulse — ML/Classifier Спецификация

> **Status:** This document describes the ML architecture of FitPulse.
> **Phase 1 (production):** Rule-based classifier (Go) + template-based plan engine (Go).
> **Phase 2:** Optional ONNX classifier inference with rule-based fallback.

## Обзор

FitPulse использует два компонента:

1. **Classifier** — правила-на-основе классификация состояния пользователя, реализованная на Go. Работает только с биометрическими данными с носимых устройств. **Phase 1 production.**
2. **Plan Engine** — генерация индивидуальных тренировочных планов на основе шаблонов в Go. **Phase 1 production.**

Генерация плана использует 2-tier fallback:

1. **Primary**: Template-based генерация в Go с модификаторами
2. **Fallback**: Rule-based генерация на основе шаблонов по классу состояния

---

## Архитектура моделей

### Classifier (Go-правила, 6 классов)

**Назначение:** Определение состояния пользователя по биометрическим данным с браслета.

**Входные признаки (7 признаков):**

| # | Признак | Тип | Единица | Примечание |
| - | ------- | --- | ------- | ---------- |
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
- Анкета пользователя (`GET /api/v1/survey`)

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
| - | ------------ | ----------- | ---------------- |
| 1 | `recovery` | Восстановление | HRV > 80 И (HR < 60% HRmax Или sleep > 8) |
| 2 | `endurance_basic` | Базовая выносливость E1-E2 | HRV 50–80, HR 65–80% HRmax, sleep 6–8 |
| 3 | `endurance_threshold` | Пороговая выносливость E3 | HRV 40–50, HR 80–90% HRmax |
| 4 | `power_hiit` | Силовая/HIIT | HRV > 60, HR > 90% HRmax, sleep > 7 |
| 5 | `overtraining` | Перетренированность | HRV < 30 И HR < 60% HRmax |
| 6 | `illness` | Заболевание | Температура > 37.5°C Или HRV < 30 с признаками болезни |

**Endpoint:** `POST /classify` (service на порту 8001, вызывается из gateway через `POST /api/v1/classify`)

---

### Plan Engine (Go-шаблоны)

**Назначение:** Генерация тренировочных планов на основе предопределённых шаблонов с применением модификаторов.

**Входные данные:**

- `classification` — класс состояния от классификатора
- `user` — профиль пользователя (возраст, пол, уровень подготовки, вес, рост)
- `survey` — данные анкеты (JSONB из `user_profiles.survey_data`)
- `biometrics` — последние биометрические показатели
- `constraints` — ограничения (длительность в неделях, доступные дни)

**Шаблоны (`internal/templates/training_templates.go`):**

| Класс | duration_range | intensity_range | exercises | rest_ratio |
| -------- | -------------- | --------------- | --------- | ---------- |
| recovery | 20–40 | 0.3–0.5 | лёгкая разминка, растяжка, дыхательные упражнения, йога | 0.7 |
| endurance_basic | 45–90 | 0.5–0.7 | бег, велосипед, плавание, лыжи | 0.4 |
| endurance_threshold | 30–60 | 0.7–0.85 | темновой бег, интервалы на пороге, фартлек, критическая мощность | 0.3 |
| power_hiit | 20–45 | 0.85–1.0 | HIIT, силовые, спринты, кроссфит | 0.5 |
| overtraining | 0–20 | 0.0–0.3 | отдых, ходьба, растяжка, йога | 0.8 |
| illness | 0–0 | 0.0–0.0 | полный отдых | 1.0 |

**Модификаторы:**

1. **Возраст**: если `age > 60`, снизить `intensity_level` на 20%, увеличить `warmup_ratio`.
2. **BMI**: если `BMI > 35`, снизить интенсивность, исключить прыжки и бег.
3. **Менструация**: если в анкете `menstruation: true`, снизить интенсивность на 10–20%.
4. **Противопоказания**: исключить упражнения на больную область.
5. **Сон**: если `sleep_hours < 5`, снизить интенсивность на 15%.
6. **HRV**: если `HRV < 20`, ограничить интенсивность до 0.4.

**Fallback chain:**

1. **Primary:** Template-based генерация в Go (`internal/planner/engine.go`)
2. **Fallback:** Rule-based генерация на основе шаблонов по классу состояния (`generateBasicWeeklyWorkouts`)

---

## Интеграция

### Classifier

**Endpoint:** `POST /classify` (service на порту 8001, вызывается из gateway через `POST /api/v1/classify`)

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

- **Выход:** класс, уверенность, вероятности, рекомендации

### Plan Engine

**Endpoint:** Встроен в `training-service` (gRPC `GeneratePlan`)

- **Вход:** `user_id`, `classification`, `duration_weeks`, `available_days`, `plan_data` (опционально)
- **Выход:** `plan_id`, `plan_data` с неделями и днями
- **Fallback:** rule-based → static beginner

---

## Требования к данным

| Параметр | Значение |
| -------- | -------- |
| Минимум данных для классификации | 7 признаков + user_id |
| Минимум данных для плана | classification + user_id + duration_weeks |
| Версионирование шаблонов | template_version в `training_plans` таблице |
| Логирование | classifier_logs таблица с feedback_rating |

---

## Безопасность и ограничения

1. **Medical disclaimer**: Все планы носят рекомендательный характер. При заболеваниях/травмах — консультация врача обязательна.
2. **Privacy**: Анкета пользователя хранится как JSONB в `user_profiles.survey_data`. Персональные данные не используются для обучения моделей без явного согласия.
3. **Audit**: Каждый generated plan содержит `classification` и `template_version` для отслеживания.
4. **Fallback**: При отсутствии анкеты используются дефолтные параметры (`beginner`, `available_days: [1,3,5]`).

---

## Open Questions

- [ ] Добавить ONNX-инференс для классификатора с порогом переключения 0.85
- [ ] Реализовать incremental training для ONNX модели
- [ ] Добавить A/B тестирование планов (качество планов vs фидбек пользователей)
- [ ] Реализовать внешний LLM с explicit opt-in (`ai_assistant_enabled` в `user_profiles`)

> **Примечание:** Эти задачи отслеживаются в `docs/phase2-roadmap.md` и требуют отдельного планирования sprint'а.
