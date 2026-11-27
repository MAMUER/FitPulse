# FitPulse — ML/Classifier Спецификация

> **Статус:** документ описывает **реализованную** ML-архитектуру FitPulse (обновлено 2026-11-22).
> **Продакшн:** гибридный классификатор — rule-based (Go, первичный) + ML-деревья HistGradientBoosting (Go, для `unknown`).

---

## 1. Обзор

FitPulse использует **гибридный классификатор**:

1. **Rule-based (Go, первичный)** — детерминированные правила на физиологических признаках. Всегда доступен, не требует модели.
2. **ML (Python → JSON → Go, для `unknown`)** — HistGradientBoosting, экспортированный в JSON-деревья. Применяется только тогда, когда правила не могут дать ответ.

**Почему правила первичны.** Публичные датасеты (WESAD, ADARP, PPG-DaLiA и др.) измеряют **кожную температуру запястья** (~33 °C), а SpO2 присутствует менее чем в 0.5% строк, сон и давление — почти отсутствуют. Модель, обученная на таких данных, вне распределения ведёт себя произвольно. Правила кодируют предметную семантику (core-температура, SpO2, сон) и работают во всём диапазоне.

**Что делает ML.** Разрешает состояния `unknown` (низкий SpO2, субфебрилитет), где правила намеренно не дают класса, и служит основой для персонализации (дельты Δ_u).

---

## 2. Архитектура

```text
┌─────────────────────────────────────────────────────────────────┐
│                    POST /classify Request                       │
└─────────────────────────┬───────────────────────────────────────┘
                          ▼
        ┌──────────────────────────────────────────┐
        │ buildClassifierFeatures (7 признаков)    │
        │ temperature_source: wrist | core         │
        │ missing_metrics / data_quality           │
        └──────────────────┬───────────────────────┘
                           ▼
        ┌──────────────────────────────────────────┐
        │ classifyState (Go rules)                 │
        └──────────────────┬───────────────────────┘
                           ▼
                 Класс == unknown?
                           │
              ┌────────────┴────────────┐
              ▼                         ▼
            ДА                         НЕТ
              │                         │
              ▼                         ▼
   ┌────────────────────┐      ┌──────────────────┐
   │ ML: деревья + Δ_u  │      │  Используем      │
   │ (tree_model.go)    │      │  правило         │
   └─────────┬──────────┘      └────────┬─────────┘
             │                          │
             ▼                          ▼
   confidence × quality-штраф     confidence × quality-штраф
             │                          │
             └────────────┬─────────────┘
                          ▼
        ┌──────────────────────────────────────────┐
        │ Ответ: state, confidence, probabilities, │
        │ model, data_quality, missing_metrics     │
        └──────────────────────────────────────────┘
```

---

## 3. Входные признаки

| # | Признак | Тип | Единица | Примечание |
| --- | --------- | ----- | --------- | ------------ |
| 1 | `heart_rate` | float64 | уд/мин | обязателен, валидация 20–250 |
| 2 | `heart_rate_variability` | float64 | мс | HRV (SDNN/RMSSD) |
| 3 | `spo2` | float64 | % | 70–100 |
| 4 | `temperature` | float64 | °C | зависит от `temperature_source` |
| 5 | `blood_pressure_systolic` | float64 | мм рт.ст. | часто отсутствует |
| 6 | `blood_pressure_diastolic` | float64 | мм рт.ст. | часто отсутствует |
| 7 | `sleep_hours` | float64 | часы | часто отсутствует |

### 3.1. Температура: wrist или core

Браслеты измеряют **кожу запястья** (на 2–3 °C ниже ядра). Поэтому ML-модель обучается на коже, а правила должны это учитывать.

Поле `temperature_source`:

| Источник | Субфебрилитет | Лихорадка | Высокая |
| ---------- | --------------- | ----------- | --------- |
| `core` | 37.3 | 37.5 | 38.0 |
| `wrist` (абсолютные) | 35.8 | 36.0 | 36.5 |
| `wrist` + `temperature_baseline` $b$ | $b+0.4$ | $b+0.7$ | $b+1.2$ |

По умолчанию — `wrist`. `temperature_baseline` — скользящее среднее температуры пользователя (персонализация измерения).

### 3.2. Качество данных

Ответ `/classify` содержит:

- `missing_metrics` — список отсутствующих показателей (`hrv`, `spo2`, `temperature`, `sleep_hours`, `systolic_pressure`, `diastolic_pressure`);
- `data_quality` — `high` (ничего не пропущено) / `medium` (≤2 пропуска) / `low` (>2);
- confidence умножается на 1.0 / 0.9 / 0.8 соответственно.

Отсутствующие значения подставляются нейтральными (HRV 50, SpO2 98, t 37, АД 120/80, сон 7), но это отражается в качестве.

---

## 4. Классы (7)

| Индекс | Slug | Название RU | Ключевые правила |
| -------- | ------ | ------------- | ------------------ |
| 0 | `recovery` | Восстановление | HRV > 80 И (HR < 60% HRmax ИЛИ сон > 8) |
| 1 | `endurance_basic` | Базовая выносливость | HR 65–80% HRmax, HRV 50–80 |
| 2 | `endurance_threshold` | Пороговая выносливость | HR 80–90% HRmax |
| 3 | `power_hiit` | Силовая/HIIT | HR > 90% HRmax |
| 4 | `overtraining` | Перетренированность | HRV < 30 ИЛИ сон < 5 |
| 5 | `illness` | Заболевание | t > порога ИЛИ SpO2 < 95 + признаки |
| 6 | `unknown` | Неопределено | SpO2 < 93, субфебрилитет, default |

---

## 5. ML-модель

### 5.1. Полная модель (Python)

- **Файл:** `models/classifier.pkl` (joblib, HistGradientBoostingClassifier).
- **Данные:** 12.9 млн строк из 13 датасетов → после очистки 6.18 млн → балансировка.
- **Признаки:** `heart_rate`, `hrv`, `spo2`, `temperature` (BP и сон отсутствуют в данных).
- **Классы:** 7.
- **Accuracy на валидации:** ≈ 0.999.

### 5.2. Экспорт для Go — деревья (histgb-v1)

**Файл:** `models/classifier_trees.json` (~1.9 МБ, 700 деревьев, 39 800 узлов).

Формат:

```json
{
  "format": "histgb-v1",
  "n_classes": 7,
  "learning_rate": 0.05,
  "baseline": [...],
  "feature_keys": ["heart_rate","hrv","spo2","temperature"],
  "feature_min": [25.0, 1.28, 83.0, 30.01],
  "feature_max": [210.47, 390.61, 100.0, 37.43],
  "classes": [...],
  "trees": [{"feature":[...],"threshold":[...],"left":[...],"right":[...],
             "missing_left":[...],"value":[...],"leaf":[...]}]
}
```

**Важно:**

- значения листьев **уже содержат** `learning_rate` — Go не умножает повторно;
- пороги внутренних узлов могут быть `+inf` — экспортируются как `1e308` (JSON не поддерживает Infinity);
- пропуски в признаках кодируются как `NaN` и обрабатываются через `missing_go_to_left`.

### 5.3. Экспортёр

**Скрипт:** `scripts/export_go_model.py`

```powershell
python scripts/export_go_model.py
```

Создаёт:

- `models/classifier_trees.json` — деревья для Go;
- `models/classifier_golden.json` — 400 эталонных случаев (сырые значения + ожидаемый класс) для Go-теста.

### 5.4. Линейная модель (legacy)

- `models/classifier_weights.json` — веса LogisticRegression (линейная аппроксимация бустинга). Совпадает с бустингом ~71%, поэтому **не используется**, если есть деревья.
- `scripts/train_classifier_linear.py` — обучение линейной модели на дистиллированном датасете (без внешних зависимостей, только stdlib). Нужна, если деревья недоступны.

---

## 6. Go-инференс

**Файлы:**

- `cmd/classifier/tree_model.go` — обход деревьев HistGradientBoosting (`TreeModel`, формат `histgb-v1`);
- `cmd/classifier/onnx_classifier.go` — линейная модель (legacy) + min-max нормализация + softmax;
- `cmd/classifier/personal_delta.go` — персональные дельты Δ_u (работают и с деревьями, и с линейной моделью);
- `cmd/classifier/main.go` — HTTP-сервер, правила, качество данных.

### 6.1. Приоритет модели

```go
func (s *classifierServer) activeModel() scoreModel {
    if s.treeModel != nil { return s.treeModel }
    if s.onnxClassifier != nil { return s.onnxClassifier }
    if s.personalClassifier != nil { return s.personalClassifier.classifier }
    return nil
}
```

### 6.2. Формула

Деревья:

$$\text{score}_c(x) = b_c + \sum_{m:\,c(m)=c} h_m(x), \qquad \hat{c} = \arg\max_c \text{score}_c(x)$$

Персональная дельта:

$$\widetilde{\text{score}}_c(x) = \text{score}_c(x) + \sum_j \Delta_u[c,j] \cdot x_j$$

### 6.3. Порог переключения

```go
const mlSwitchThreshold = 0.25
```

ML применяется **только** для `unknown`; если ML-уверенность ниже порога, остаётся `unknown`.

---

## 7. Персонализация плана

**Файл:** `internal/planner/personalize.go`

Вход `PersonalizationInput`: возраст, пол, вес, рост, цель, менструальная фаза, стресс, сон, аллергии, ограничения, биометрия.

Оператор $\Pi$ домножает вектор GAN и шаблон:

| Фактор | Эффект |
| -------- | -------- |
| Возраст <18 / >60 / ≥75 | ×0.85 / ×0.8 / ×0.65 + меньше подходов |
| ИМТ <18.5 / 25–30 / 30–35 / ≥35 | ×0.9 / ×0.95 / ×0.85 / ×0.7 |
| Менструация / лютеиновая | ×0.8 / ×0.9 (только для женщин) |
| Стресс ≥7 / 5–7 | ×0.75 / ×0.9 |
| Сон <5 / <6.5 | ×0.7 / ×0.85 |
| HRV <20 / <40 | потолок 0.4 / ×0.85 |
| ЧСС покоя >100 / >85 | ×0.7 / ×0.9 |
| SpO2 <94 | ×0.7 |
| t > 37.5 | ×0.6, длительность ×0.8 |
| Сердце / гипертония / диабет / почки / печень / ортопедия | ×0.6 … ×0.85 |

Результат попадает в `plan_data.personalization` — список применённых ограничений.

---

## 8. Меню

**Файл:** `internal/chat/menu_profile.go` (`BuildDayMenuForProfile`)

Учитывает:

- цель → шаблон (похудение / масса / баланс);
- вес, рост, возраст, пол → калораж (Миффлин–Сан Жеор) × активность × цель;
- аллергии → замена блюд на безопасные (`allergenSafeMeals`);
- диабет → перекус без сахара;
- менструация → акцент на железо; лютеиновая фаза → магний и сложные углеводы;
- стресс ≥ 7 → меньше кофеина, больше магния.

---

## 9. Pipeline: Data → Training → Deployment

### 9.1. Подготовка данных

**Скрипты:** `scripts/prepare_raw_datasets.py`, `scripts/prepare_classifier_dataset.py`

```powershell
python -u scripts\prepare_raw_datasets.py       # 13 датасетов -> classifier_dataset.csv (сырые значения)
python -u scripts\prepare_classifier_dataset.py # очистка + min-max + балансировка
```

`prepare_classifier_dataset.py`:

- удаляет дубликаты и строки с выбросами (ЧСС 25–230, HRV 1–400, SpO2 70–100, t 30–43);
- сохраняет `models/classifier_scaler.json` (min/max);
- балансирует классы (до 20 000 на класс, редкие — до 5× oversampling).

**Выход:** `datasets/processed/classifier_dataset.csv` (~100 000 строк, сбалансировано).

### 9.2. Обучение

```powershell
python -u scripts\train_classifier.py `
    --dataset datasets\processed\classifier_dataset.csv `
    --output models\classifier.onnx --max-iter 100 --lr 0.05
```

Результат:

- `models/classifier.pkl` — HistGradientBoosting;
- `models/classifier_weights.json` — линейные веса (legacy);
- `models/train_metrics.json` — метрики.

### 9.3. Экспорт деревьев

```powershell
python scripts\export_go_model.py
```

Создаёт `models/classifier_trees.json` и `models/classifier_golden.json`.

### 9.4. Go-инференс

- `tree_model.go` загружает деревья и обходит их;
- `onnx_classifier.go` — fallback;
- `/health` возвращает фактическое состояние (`model_loaded`, `scaler_loaded`).

### 9.5. Сборка образа

`cmd/classifier/Dockerfile` копирует `models/` (включая деревья), иначе ML молча отключается.

---

## 10. Датасеты (используемые)

Все данные лежат в **`datasets/raw`**, которая является junction на `H:\Games\raw`.
Репозиторий не содержит сами файлы — только ссылку.

| Датасет | Папка | Ссылка | Лицензия | Что используем |
| --------- | ------- | -------- | ---------- | ---------------- |
| **WESAD** | `datasets/raw/WESAD` | <https://ubi29.informatik.uni-siegen.de/usi/data_wesad.html> | CC BY-NC 4.0 | HR, HRV, EDA, кожная температура, АКГ; 15 субъектов |
| **WESD** | `datasets/raw/WESD` | <https://utexas.box.com/s/2ef96g1r69eyox0t6t9bv2gy1p1zv1h0p> | CC BY 4.0 | Empatica E4, 210 файлов |
| **BIDMC** | `datasets/raw/bidmc-ppg-and-respiration-dataset-1.0.0` | <https://physionet.org/content/bidmc/1.0.0/> | ODC-BY 1.0 | SpO2, HR, PPG, дыхание; 53 субъекта |
| **PPG-DaLiA** | `datasets/raw/PPG_DaLiA` | <https://archive.ics.uci.edu/dataset/495/ppg+dalia> | CC BY 4.0 | BVP/PPG, EDA, температура, ACC, ЧСС; 15 субъектов |
| **ADARP** | `datasets/raw/ADARP` | <https://huggingface.co/datasets/FcmC/E4SelfLearning> | CC BY 4.0 | E4 CSV: HR, HRV, EDA, TEMP |
| **big-ideas** | `datasets/raw/big-ideas` | <https://huggingface.co/datasets/FcmC/E4SelfLearning> | CC BY 4.0 | E4 CSV с заголовками |
| **in-gauge_en-gage** | `datasets/raw/in-gauge_en-gage` | <https://huggingface.co/datasets/FcmC/E4SelfLearning> | CC BY 4.0 | E4 CSV, двойная вложенность |
| **SPD** | `datasets/raw/SPD` | <https://huggingface.co/datasets/FcmC/E4SelfLearning> | CC BY 4.0 | E4 CSV |
| **stress_nurses** | `datasets/raw/stress_detection_nurses_hospital` | <https://huggingface.co/datasets/FcmC/E4SelfLearning> | CC BY 4.0 | E4 CSV |
| **Toadstool** | `datasets/raw/Toadstool` | <https://huggingface.co/datasets/FcmC/E4SelfLearning> | CC BY 4.0 | E4 CSV |
| **ue4w** | `datasets/raw/ue4w` | <https://huggingface.co/datasets/FcmC/E4SelfLearning> | CC BY 4.0 | E4 CSV |
| **WEEE** | `datasets/raw/WEEE` | <https://huggingface.co/datasets/FcmC/E4SelfLearning> | CC BY 4.0 | Fitbit/Apple Watch JSON/CSV |

> **ADARP, big-ideas, in-gauge_en-gage, SPD, stress_nurses, Toadstool, ue4w, WEEE** — это подвыборки
> одного датасета **E4SelfLearning** (<https://huggingface.co/datasets/FcmC/E4SelfLearning/tree/main>).
> Они уже распакованы в `datasets/raw/<ИМЯ>`.

### 10.1. Данные сна и SpO2 (PhysioNet)

Эти датасеты закрывают `sleep_hours` и `spo2`, которых нет в E4/PPG:

| Датасет | Папка | Ссылка | Что берём |
| --------- | ------- | -------- | ----------- |
| **SHHS PSG** | `datasets/raw/sleep-heart-health-study-psg-database-1.0.0` | <https://physionet.org/content/shhpsgdb/1.0.0/> | SaO2, PR, гипнограмма → sleep_hours |
| **SleepAccel** | `datasets/raw/motion-and-heart-rate-from-a-wrist-worn-wearable-and-labeled-sleep-from-polysomnography-1.0.0` | <https://physionet.org/content/sleep-accel/1.0.0/> | Apple Watch HR + стадии сна → sleep_hours |
| **Apnea-ECG** | `datasets/raw/apnea-ecg-database-1.0.0` | <https://physionet.org/content/apnea-ecg/1.0.0/> | SpO2 (записи `*r`) |
| **PMData** | `datasets/raw/pmdata` | <https://datasets.simula.no/pmdata> | Fitbit minutesAsleep → sleep_hours, HR |
| **UCDDB** | `datasets/raw/ucddb-1.0.0` | <https://physionet.org/content/ucddb/1.0.0/> | SpO2 + стадии (EDF; пропускается без pyedflib) |
| **Respiratory + Oximetry** | `datasets/raw/respiratory-oximetry-apnoea-1.0.0` | <https://physionet.org/content/respiratory-oximetry-apnoea/1.0.0/> | PPG/SpO2 у здоровых |

Скрипт: `scripts/prepare_sleep_datasets.py` → `datasets/processed/sleep_dataset.csv`.

### 10.2. Питание и менструальный цикл

| Датасет | Папка | Ссылка | Скрипт | Результат |
| --------- | ------- | -------- | -------- | ----------- |
| **Open Food Facts** | `datasets/raw/food` | <https://huggingface.co/datasets/openfoodfacts/product-database> | `prepare_food_datasets.py` | аллергены/нутриенты (parquet, требует schema-парсера) |
| **USDA FoodData Central** | `datasets/raw/usda` | <https://fdc.nal.usda.gov/download-datasets.html> | `prepare_food_datasets.py` | `food_nutrients.csv` (16 301 продукт) |
| **Menstrual (factors)** | `datasets/raw/Menstrual_cycle_data_with_factors` | <https://www.kaggle.com/datasets/akshayas02/menstrual-cycle-data-with-factors-dataset> | `prepare_menstrual_datasets.py` | `menstrual_dataset.csv` |
| **Menstrual Health** | `datasets/raw/Menstrual_Health_&_Productivity` | <https://www.kaggle.com/datasets/puspitachowdhury2/menstrual-health-dataset> | `prepare_menstrual_datasets.py` | 18 871 запись (цикл, стресс, сон) |

### 10.3. Тренировочные планы (условная ГАН)

| Датасет | Путь | Ссылка | Лицензия | Что используется |
| ------- | ---- | ------ | -------- | ---------------- |
| **Gym Members Exercise Dataset** | `datasets/raw/gym_members_exercise_tracking.csv` | <https://www.kaggle.com/datasets/valakhorasani/gym-members-exercise-dataset> | Apache 2.0 | 973 записи: Max/Avg/Resting BPM, длительность, калории, тип тренировки, частота, опыт, профиль → 19-мерный вектор плана |

Скрипт: `scripts/build_training_plans_from_gym.py` → `datasets/processed/training_plans.csv` (973 строки, классы 0–5: 222/70/169/281/102/129).

### 10.4. Подготовка

Все архивы распакованы, лишние файлы удалены. Проверить:

```powershell
Get-ChildItem datasets\raw | Select-Object Name, LinkType, Target
```

Ожидаемо: `datasets\raw` → `Junction` → `H:\Games\raw`.

## 11. Пайплайн

```powershell
# 1. 13 датасетов -> classifier_dataset.csv (сырые значения)
python -u scripts\prepare_raw_datasets.py

# 2. сон + SpO2 из PhysioNet -> sleep_dataset.csv
python -u scripts\prepare_sleep_datasets.py

# 2a. питание (USDA) -> food_nutrients.csv
python -u scripts\prepare_food_datasets.py

# 2b. менструальный цикл -> menstrual_dataset.csv
python -u scripts\prepare_menstrual_datasets.py

# 2. очистка выбросов + min-max (models/classifier_scaler.json) + балансировка
python -u scripts\prepare_classifier_dataset.py

# 3. HistGradientBoosting + линейные веса
python -u scripts\train_classifier.py `
    --dataset datasets\processed\classifier_dataset.csv `
    --output models\classifier.onnx --max-iter 100 --lr 0.05

# 4. экспорт деревьев + golden set для Go
python scripts\export_go_model.py

# 5. проверка Go-инференса
go test ./cmd/classifier -run TestTreeModelMatchesGoldenSet -v
```

Ожидаемый результат:

- `datasets/processed/classifier_dataset.csv` — ~100 000 строк, сбалансировано;
- `models/classifier.pkl` — HistGradientBoosting;
- `models/classifier_trees.json` — 700 деревьев (histgb-v1);
- `models/classifier_golden.json` — 400 эталонных случаев;
- `TestTreeModelMatchesGoldenSet` — PASS (Go == Python).

---

## 12. Метрики

| Метрика | Значение |
| --------- | ---------- |
| Сырых строк (13 датасетов) | 31 227 838 |
| После очистки (выбросы, NaN) | 21 107 086 |
| После балансировки | 120 070 (по 20 000 на класс, `recovery` — 70) |
| HistGradientBoosting, validation accuracy | 0.9997 |
| HistGradientBoosting, test accuracy | 0.9994 |
| Деревья для Go | 700 деревьев, 39 800 узлов, 1.9 МБ |
| Go vs Python (golden set) | 400 / 400 |

### 12.1. Ограничение модели

Метрики 0.999+ относятся к **отложенной выборке того же распределения**, что и обучение (правила `rule_based_label` на реальных E4/PPG-данных). На синтетических «канонических» состояниях (например, HR 50 / HRV 90 / сон 8) модель тяготеет к `overtraining`, потому что в публичных датасетах почти нет примеров `recovery` (15) и `illness` (125).

Поэтому:

- **правила — первичный источник** (они покрывают весь физиологический диапазон);
- **ML применяется только для `unknown`** (низкий SpO2, субфебрилитет);
- `recovery` и `illness` недопредставлены в данных — это ограничение датасетов, а не кода.

---

## 13. API (Classifier Service: 8001)

| Метод | Путь | Описание |
| ------- | ------ | ---------- |
| POST | `/classify` | Классификация состояния |
| POST | `/classify/delta` | Обновление персональной дельты Δ_u |
| GET | `/health` | Здоровье (`model_loaded`, `scaler_loaded`) |
| GET | `/classes` | Список классов |
| GET | `/model-info` | Метаданные модели |
| GET | `/metrics` | Prometheus-метрики |

---

## 14. Структура файлов

```text
scripts/
├── prepare_raw_datasets.py        # обработка 13 датасетов
├── prepare_classifier_dataset.py  # очистка + min-max + балансировка
├── train_classifier.py            # HistGradientBoosting + линейный экспорт
├── train_classifier_linear.py     # чистая линейная модель (stdlib)
├── export_go_model.py             # экспорт деревьев + golden set
└── cleanup_raw_datasets.py        # утилита очистки

models/
├── classifier.pkl                 # HistGradientBoosting
├── classifier_trees.json          # деревья для Go (histgb-v1)
├── classifier_weights.json        # линейные веса (legacy)
├── classifier_scaler.json         # min/max нормализация
├── classifier_golden.json         # 400 эталонных случаев для теста
├── gan_generator.pt               # GAN
├── gan_norm_params.json
└── gan_history.json

cmd/classifier/
├── main.go                        # HTTP-сервер, правила, качество данных
├── tree_model.go                  # Go-инференс деревьев
├── onnx_classifier.go             # линейная модель (legacy) + нормализация
├── personal_delta.go              # дельты Δ_u
├── dataset_dump.go                # генерация дистиллированного датасета
├── db.go                          # подключение к PostgreSQL
└── delta_store_pg.go              # персистентные дельты

internal/planner/
├── engine.go                      # шаблонный план
├── gan.go                         # GAN-вектор → план
└── personalize.go                 # персонализация плана

internal/chat/
├── menu.go                        # статичное меню
└── menu_profile.go                # персонализированное меню
```

---

## 15. Известные проблемы / Планы

1. **Дисбаланс классов** — `recovery` (15) и `illness` (125) в сырых данных; компенсируется oversampling и `class_weight='balanced'`, но точность на них ниже.
2. **Температура** — в датасетах кожа запястья (~33 °C), в проде может быть core; нужна калибровка `temperature_source` на фронте/устройстве.
3. **SpO2/сон/давление** — почти отсутствуют в публичных датасетах; правила используют нейтральные значения.
4. **Нет GPU** — HistGradientBoosting обучается на CPU.
5. **Локальные пути** — часть датасетов лежит вне репозитория (`H:\Games\raw`); для CI их нужно либо кэшировать, либо исключить из пайплайна.

---

*Последнее обновление: 2026-11-22*
*Отражает реализованный пайплайн на момент коммита*

## 16. Дополнение: реальные данные (2026-11-26)

- **Open Food Facts**: `nutriments` — массив `{name, 100g}`; парсер `prepare_food_datasets.py` извлекает 50 000 продуктов. Вместе с USDA (16 301) — `food_nutrients.csv` (66 301).
- **Sleep-EDF**: PSG читается (`mne`), `Temp rectal` — core temperature → `datasets/processed/sleep_edf_dataset.csv` (18 записей). Гипнограммы в скачанной копии битые (S3 `NoSuchKey`), их нужно перекачать по `https://physionet-open.s3.amazonaws.com/sleep-edfx/1.0.0/sleep-cassette/SC4001EC-Hypnogram.edf`.
- **Core vs skin**: температура ядра (~37 °C) и кожи запястья (~33 °C) хранятся раздельно, чтобы не смешивать семантику.
- **UCDDB**: 25 записей EDF → SpO2 + стадии сна → `sleep_dataset.csv` (всего 81 строка).

### 16.1. EDF+ гипнограммы (TAL)

`scripts/edf_reader.py` читает EDF+ аннотации формата `+onset\x15duration\x14text\x14\x00`.
`process_sleep_edf()` объединяет `SC*-PSG.edf` (core temp) и `SC*-Hypnogram.edf` (стадии сна) по префиксу subject id.
Скачана только `SC4001EC-Hypnogram.edf` (1.175 ч сна); остальные гипнограммы в копии — S3 `NoSuchKey`, их нужно перекачать:
`https://physionet-open.s3.amazonaws.com/sleep-edfx/1.0.0/sleep-cassette/SC4001EC-Hypnogram.edf`

## 17. Меню из реальных продуктов и базлайн температуры

- `scripts/export_menu_food.py` отбирает 1 500 блюд из `food_nutrients.csv` → `internal/chat/food_data.json` (go:embed, 203 КБ).
- `BuildDayMenuForProfile` при наличии профиля строит меню из базы: 4 приёма пищи, калораж по Миффлину–Сан Жеору, фильтр аллергий, ограничения (диабет), менструальная фаза, стресс.
- `training-service.temperatureBaseline()` считает среднюю температуру за 14 дней и передаёт её как `temperature_baseline` — wrist-пороги становятся относительными.
