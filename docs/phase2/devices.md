# Extended Devices

## 24. Расширение поддерживаемых устройств

### 24.1 Текущий статус Phase 1

- **Open Wearables** — production-ready, webhook `POST /api/v1/integrations/open-wearables/webhook`
- **Device Aggregator** — production-ready, легковесный forwarder
- **Withings** — production-ready, прямой OAuth `/api/v1/integrations/withings/auth` + `/api/v1/integrations/withings/callback` + `/api/v1/integrations/withings/sync`
- **Прямые OAuth-интеграции** — удалены из кодовой базы, кроме Withings (Fitbit, Flo, OKOK)

### 24.2 Поддерживаемые источники (whitelist)

Код и API принимают следующие источники через Open Wearables webhook:

| Источник | ID в webhook | Статус |
| --- | --- | --- |
| Apple Health | `apple_health` | ✅ Phase 1 |
| Garmin | `garmin` | ✅ Phase 1 |
| Google Health Connect | `google_health_connect` | ✅ Phase 1 |
| Open Wearables | `open_wearables` | ✅ Phase 1 |
| Whoop | `whoop` | ✅ Phase 1 |
| Polar | `polar` | ✅ Phase 1 |
| Suunto | `suunto` | ✅ Phase 1 |
| Strava | `strava` | ✅ Phase 1 |
| Oura | `oura` | ✅ Phase 1 |
| Ultrahuman | `ultrahuman` | ✅ Phase 1 |
| Samsung Health | `samsung_health` | ✅ Phase 1 |

### 24.3 Поддерживаемые метрики (whitelist)

Код и API принимают следующие метрики через Open Wearables webhook:

| Метрика | ID в webhook | Единицы / Формат |
| --- | --- | --- |
| Weight | `weight` | кг |
| Heart Rate | `heart_rate` | уд./мин |
| Resting Heart Rate | `resting_heart_rate` | уд./мин |
| SpO2 | `spo2` | % |
| Sleep | `sleep` | мин / score |
| Sleep Stages | `sleep_stages` | структурированные стадии сна |
| Steps | `steps` | шаги |
| Menstrual Cycle | `menstrual_cycle` | даты / интенсивность |
| Body Composition | `body_composition` | % / кг |
| HRV | `hrv` | мс |
| HRV SDNN | `hrv_sdnn` | мс |
| HRV RMSSD | `hrv_rmssd` | мс |
| Temperature | `temperature` | °C |
| Body Temperature | `body_temperature` | °C |
| Blood Pressure | `blood_pressure` | мм рт. ст. |
| Blood Glucose | `blood_glucose` | мг/дл |
| Active Energy | `active_energy` | ккал |
| Basal Energy | `basal_energy` | ккал |
| BMI | `bmi` | кг/м² |
| Lean Body Mass | `lean_body_mass` | кг |
| Flights Climbed | `flights_climbed` | этажи |
| Water Intake | `water_intake` | мл |
| VO2 Max | `vo2_max` | мл/кг/мин |
| Respiratory Rate | `respiratory_rate` | вд./мин |
| Workout | `workout` | тип тренировки + длительность |

### 24.4 План Phase 2

| Этап | Источник/устройство | Срок | Приоритет | Задачи |
| --- | --- | --- | --- | --- |
| 1 | Open Wearables (все источники выше) | ✅ Phase 1 | P0 | Агрегация данных здоровья через единый webhook |
| 2 | Samsung Galaxy Watch | 3–4 недели | P2 | Через Open Wearables / Samsung Health Connect |
| 3 | Huawei Watch D2 | 3–4 недели | P2 | Через Open Wearables / Huawei Health Kit |

### 24.5 Acceptance Criteria

- Каждое устройство имеет working Open Wearables integration (aggregator → webhook → biometric-service)
- Минимум 5 метрик синхронизируются автоматически
- Данные поступают через `POST /api/v1/integrations/open-wearables/webhook` в biometric-service
- UI отображает статус подключения и последнюю синхронизацию

### 24.6 Архитектурные ограничения

- Biometric-service остаётся универсальным: принимает webhook от Open Wearables, валидирует, сохраняет в `biometric_data`
- Device-aggregator используется как легковесный webhook-forwarder для Open Wearables
- Прямые OAuth-интеграции (Fitbit, Flo, OKOK) удалены из кодовой базы и swagger
- Withings — единственная прямая OAuth-интеграция, сохранённая в Phase 1

### 24.7 Менструальные данные

#### Источники, из которых интеграция невозможна

Следующие источники менструальных данных **недоступны** и не могут быть подключены:

- **OKOK** — нет публичного API для сторонних разработчиков; Open Wearables не поддерживает
- **Flo** — нет открытого OAuth API; есть только закрытая партнёрская программа для медицинских проектов
- **Clatch** — нет публичного API; данные остаются внутри приложения
- **Huawei Health Kit** — Open Wearables не поддерживает; требует партнёрского соглашения с Huawei; Health Connect Google не отдаёт данные Huawei

#### Источники, из которых менструальные данные доступны

- **Withings** — прямой OAuth реализован; метрика `menstrual_cycle` доступна через `/api/v1/integrations/withings/sync`
- **Garmin / Oura / Samsung Health** — через Open Wearables webhook, метрика `menstrual_cycle`

#### Fallback: менструальный календарь FitPulse

Если пользователь не использует ни один из поддерживаемых трекеров, в приложении реализован отдельный модуль менструального календаря:

- Ручной ввод цикла через UI
- Прогнозы на основе истории
- Графики и напоминания

Это обеспечивает покрытие менструального отслеживания для всех пользователей независимо от внешних устройств.
