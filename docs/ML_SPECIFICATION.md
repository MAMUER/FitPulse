# FitPulse — ML/Classifier Спецификация

> **Статус:** Этот документ описывает **реализованную** ML-архитектуру FitPulse (на 2026-10-07).
> **Продакшн:** Гибридный классификатор — rule-based (Go) + ML fallback (HistGradientBoosting + ONNX).

---

## Обзор

FitPulse использует **гибридный классификатор**:

1. **Rule-based (Go, production)** — детерминированные правила на 7 признаках. Всегда доступен.
2. **ML (Python → ONNX → Go, experimental)** — HistGradientBoosting (full) + LogisticRegression fallback (ONNX). Используется при confidence ≥ 0.85.

---

## Архитектура

```text
┌─────────────────────────────────────────────────────────────────┐
│                    POST /classify Request                       │
└─────────────────────────┬───────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────────┐
│  ML Model Loaded? (onnx_classifier.go loads classifier_weights) │
└─────────────────────────┬───────────────────────────────────────┘
                          │
              ┌───────────┴───────────┐
              ▼                       ▼
         ДА                          НЕТ
              │                       │
              ▼                       ▼
    ┌──────────────────┐    ┌────────────────────────┐
    │ Predict via ONNX │    │ Rule-based (classifyState)│
    │ (LogisticRegression) │  │ (7 features, 7 classes)  │
    └────────┬───────────┘    └───────────┬────────────┘
             │                           │
             ▼                           ▼
    Confidence ≥ 0.85?              Always used
             │
       ┌──────┴──────┐
       ▼             ▼
     ДА             НЕТ
       │             │
       ▼             ▼
   Use ML         Rule-based
```

---

## Классификатор (Go + ML Hybrid)

### Входные признаки (7 признаков)

| # | Признак | Тип | Единица | Примечание |
| - | ------- | --- | ------- | ---------- |
| 1 | `heart_rate` | float64 | уд/мин | Текущий или средний за последний час |
| 2 | `heart_rate_variability` | float64 | мс | HRV (RMSSD) |
| 3 | `spo2` | float64 | % | 70–100 |
| 4 | `temperature` | float64 | °C | 35.5–38.5; ключевой для "illness" |
| 5 | `blood_pressure_systolic` | float64 | мм рт.ст. | 80–200 (часто NaN) |
| 6 | `blood_pressure_diastolic` | float64 | мм рт.ст. | 50–130 (часто NaN) |
| 7 | `sleep_hours` | float64 | часы | 0–24 (часто NaN) |

> **Важно:** Признаки 5-7 часто NaN в текущих датасетах. ML fallback использует только 4 признака (HR, HRV, SpO2, Temp).

### Источники данных

- `GET /api/v1/biometrics?metric_type=heart_rate,hrv,spo2,temperature,blood_pressure,sleep_hours`
- `GET /api/v1/health/menstrual-cycles` (дополнительный контекст)
- `GET /api/v1/health/conditions`
- `GET /api/v1/training/plans?status=active` + `POST /api/v1/training/complete`
- `GET /api/v1/devices` — проверка последнего ingestion

---

## ML Модель (Python → ONNX → Go)

### Полная модель (Python)

- **Файл:** `models/classifier.pkl` (joblib)
- **Алгоритм:** HistGradientBoostingClassifier (sklearn)
- **Данные:** 12.9M сэмплов из 13 датасетов
- **Признаки:** 4 usable (HR, HRV, SpO2, Temp) — остальные NaN
- **Классы:** 7 (recovery, endurance_basic, endurance_threshold, power_hiit, overtraining, illness, unknown)
- **Validation accuracy:** 0.9986
- **Validation size:** 2.57M samples

### Fallback ONNX (Go inference)

- **Файл:** `models/classifier.onnx` (700 bytes)
- **Алгоритм:** Pipeline(SimpleImputer + LogisticRegression)
- **Обучение:** на подвыборке 5000 сэмплов
- **Классы:** 5 (endurance_basic, endurance_threshold, power_hiit, overtraining, unknown)
- **Отсутствуют:** recovery (0), illness (5) — недостаточно сэмплов в подвыборке
- **Признаки:** 4 (HR, HRV, SpO2, Temp) с импутацией NaN

### Веса для Go (JSON)

- **Файл:** `models/classifier_weights.json`
- **Структура:** `{labels: [...], weights: [...], multi_class: int}`
- **Загружается:** `onnx_classifier.go` при старте

---

## Rule-based Классификатор (Go, 7 классов)

### 7 классов (как в коде)

| Index | Класс (slug) | Название RU | Ключевые правила |
| ------- | ------------- | ------------- | ------------------ |
| 0 | `recovery` | Восстановление | HRV > 80 И (HR < 60% HRmax ИЛИ sleep > 8) |
| 1 | `endurance_basic` | Базовая выносливость | HRV 50–80, HR 65–80% HRmax, sleep 6–8 |
| 2 | `endurance_threshold` | Пороговая выносливость | HRV 40–50, HR 80–90% HRmax |
| 3 | `power_hiit` | Силовая/HIIT | HRV > 60, HR > 90% HRmax, sleep > 7 |
| 4 | `overtraining` | Перетренированность | HRV < 30 И HR < 60% HRmax |
| 5 | `illness` | Заболевание | Температура > 37.5°C ИЛИ HRV < 30 с признаками болезни |
| 6 | `unknown` | Неопределено | Default fallback |

### Endpoint

`POST /classify` (service на порту 8001, вызывается через gateway `POST /api/v1/classify`)

### Выход

```json
{
  "predicted_class": "recovery",
  "confidence": 0.87,
  "probabilities": {
    "recovery": 0.87,
    "endurance_basic": 0.08,
    "endurance_threshold": 0.03,
    "power_hiit": 0.01,
    "overtraining": 0.01,
    "illness": 0.0,
    "unknown": 0.0
  },
  "description": "Низкая нагрузка + высокий HRV + хорошее восстановление",
  "hr_range": "50-65% HRmax",
  "recommendations": ["Лёгкая активность...", "..."]
}
```

---

## ML Switching Logic (Step 6.4)

### Константа

```go
const mlSwitchThreshold = 0.85
```

### Логика переключения

```go
// В classifyHandler (cmd/classifier/main.go)
if s.onnxClassifier != nil {
    mlClass, mlConfidence, _ := s.onnxClassifier.Predict(features)
    if mlConfidence >= mlSwitchThreshold {  // 0.85
        // Используем ML предсказание
        predictedClass = mlClass
        confidence = mlConfidence
        modelUsed = "ml"
    } else {
        // Fallback на rule-based
        predictedClass, confidence, probs = classifyState(data, age)
        modelUsed = "rule-based"
    }
} else {
    // ML не загружен → rule-based
    predictedClass, confidence, probs = classifyState(data, age)
    modelUsed = "rule-based"
}
```

### Поведение

| Условие | Результат |
| ----------- | ----------- |
| ONNX загружен + confidence ≥ 0.85 | ML prediction |
| ONNX загружен + confidence < 0.85 | Rule-based fallback |
| ONNX не загружен | Rule-based only |

### Метрики

- `ClassificationConfidence` с лейблами `model` (ml/rule-based) и `class`

---

## Pipeline: Data → Training → Deployment

### Step 6.1 — Подготовка датасета

**Скрипты:** `scripts/prepare_raw_datasets.py`, `scripts/prepare_classifier_dataset.py`

1. **prepare_raw_datasets.py** — обрабатывает 13 датасетов:
   - ADARP, SPD, ue4w, Toadstool, stress_nurses (E4 CSVs, headerless)
   - big-ideas (E4 CSVs с заголовками)
   - in-gauge_en-gage (E4 CSVs, двойная вложенность)
   - PPG_DaLiA (Pickle + E4 extracted)
   - WEEE (Fitbit/Apple Watch JSON/CSV)
   - WESAD (Pickle S2.pkl–S17.pkl)
   - WESD (Zip/CSV)
   - BIDMC (Numerics CSV)

2. **prepare_classifier_dataset.py** — очистка:
   - Удаление дубликатов и all-NaN строк
   - Удаление признаков без данных
   - Fallback на синтетику если < 1000 сэмплов

**Выход:** `datasets/processed/classifier_dataset.csv` (12.9M строк)

### Step 6.2 — Обучение

**Скрипт:** `scripts/train_classifier.py`

```powershell
python train_classifier.py --dataset datasets/processed/classifier_dataset.csv \
  --output models/classifier.onnx --max-iter 50 --lr 1e-2
```

**Результат:**

- `models/classifier.pkl` — полная HistGradientBoosting модель
- `models/classifier.onnx` — fallback ONNX (LogisticRegression)
- `models/classifier_weights.json` — веса для Go

### Step 6.3 — Go Инференс

**Файлы:** `cmd/classifier/onnx_classifier.go`, `cmd/classifier/main.go`

```go
// onnx_classifier.go — pure Go, без ONNX Runtime
// Загружает weights из JSON, делает линейную классификацию

// main.go — HTTP сервер с ML switching
const mlSwitchThreshold = 0.85
```

### Step 6.4 — Порог переключения

В `cmd/classifier/main.go`:

```go
const mlSwitchThreshold = 0.85
```

---

## Пайплайн данных (13 датасетов)

| Датасет | Формат | Сэмплы | Ключевое исправление |
| --------- | -------- | -------- | ---------------------- |
| ADARP | E4 CSV (без заголовков) | 6.1M | Пропуск строк с timestamp |
| big-ideas | E4 CSV (с заголовками) | 9.4M | Шаблон `HR_001.csv` |
| in-gauge_en-gage | E4 CSV (с заголовками) | 3.7M | Двойная вложенность исправлена |
| PPG_DaLiA | Pickle + E4 | 135K | Директории `_E4_extracted/` |
| SPD | E4 CSV (без заголовков) | 116K | Пропуск строк с timestamp |
| stress_nurses | E4 CSV (без заголовков) | 4.6M | Директории `*_extracted/` |
| Toadstool | E4 CSV (без заголовков) | 1.3M | Одноколоночные CSVs |
| ue4w | E4 CSV (без заголовков) | 956K | Пропуск строк с timestamp |
| WEEE | JSON/CSV | 4.4M | Множество поддиректорий |
| WESAD | Pickle | 15 субъектов | S2.pkl–S17.pkl |
| WESAD_labels | Pickle | 15 субъектов | Метки из WESAD |
| WESD | Zip/CSV | — | Data.zip + extracted |
| BIDMC | CSV | 53 файла | Numerics CSV |

---

## Результаты обучения

| Метрика | Значение |
| --------- | ---------- |
| Всего сэмплов | 12,865,107 |
| Разделение Train/Val | 80/20 (стратифицированное) |
| Точность на валидации | 0.9986 |
| Macro F1 | 0.89 |
| Weighted F1 | 1.00 |

### По классам (валидация)

| Класс | Precision | Recall | F1 | Support |
| ------- | ----------- | -------- | ----- | --------- |
| recovery | 0.33 | 0.33 | 0.33 | 3 |
| endurance_basic | 1.00 | 1.00 | 1.00 | 1,059,610 |
| endurance_threshold | 0.97 | 0.87 | 0.92 | 3,052 |
| power_hiit | 0.99 | 0.95 | 0.97 | 1,517 |
| overtraining | 1.00 | 1.00 | 1.00 | 438,471 |
| illness | 0.96 | 1.00 | 0.98 | 25 |
| unknown | 1.00 | 1.00 | 1.00 | 1,070,344 |

> **Примечание:** `recovery` (15 train / 3 val) и `illness` (125 train / 25 val) сильно недорепрезентированы.

---

## Структура файлов

```text
scripts/
├── prepare_raw_datasets.py      # Обработка 13 датасетов
├── prepare_classifier_dataset.py # Очистка + разметка
├── cleanup_raw_datasets.py       # Утилита очистки
└── train_classifier.py           # Обучение + ONNX экспорт

models/
├── classifier.pkl              # Полная HistGradientBoosting (1.2 MB)
├── classifier.onnx             # Fallback ONNX (700 байт)
├── classifier_weights.json     # Веса для Go (JSON)
└── classifier_test.onnx        # Тестовые модели

cmd/classifier/
├── main.go                     # HTTP сервер + ML switching
└── onnx_classifier.go          # Чистый Go линейный инференс

scripts/train_classifier.py     # Пайплайн обучения
scripts/prepare_raw_datasets.py # Обработка данных
scripts/prepare_classifier_dataset.py # Очистка датасета
scripts/cleanup_raw_datasets.py # Утилита очистки
```

---

## API Эндпоинты (Classifier Service: 8001)

| Метод | Путь | Описание |
| ------- | ------ | ---------- |
| POST | `/classify` | Классификация физиологического состояния |
| GET | `/health` | Проверка здоровья |
| GET | `/classes` | Список классов обучения |
| GET | `/model-info` | Метаданные модели (ML загружен?) |
| GET | `/metrics` | Prometheus метрики |

---

## Известные проблемы / Планы

1. **Дисбаланс классов** — `recovery` (15 сэмплов), `illness` (125) сильно недорепрезентированы
2. **ONNX fallback только 5 классов** — отсутствуют `recovery` и `illness`
3. **3 из 7 признаков в основном NaN** — `systolic_pressure`, `diastolic_pressure`, `sleep_hours`
4. **ONNX fallback использует 4 признака** — отсутствуют BP/sleep
5. **Нет GPU обучения** — HistGradientBoosting работает на CPU

---

*Последнее обновление: 2026-10-07*
*Отражает реализованный пайплайн на момент коммита*
