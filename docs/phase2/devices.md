# Extended Devices

## 24. Расширение поддерживаемых устройств

### 24.1 Текущий статус Phase 1

- **Open Wearables** — production-ready, webhook `POST /api/v1/integrations/open-wearables/webhook`
- **Device Aggregator** — production-ready, легковесный forwarder
- **Прямые OAuth-интеграции** — удалены из кодовой базы (Fitbit, Withings, Flo, OKOK)

### 24.2 План Phase 2

| Этап | Источник/устройство | Срок | Приоритет | Задачи |
| --- | --- | --- | --- | --- |
| 1 | Open Wearables | ✅ Phase 1 | P0 | Агрегация данных здоровья через единый webhook |
| 2 | Samsung Galaxy Watch | 3–4 недели | P2 | Через Open Wearables / Samsung Health Connect |
| 3 | Huawei Watch D2 | 3–4 недели | P2 | Через Open Wearables / Huawei Health Kit |

### 24.3 Acceptance Criteria

- Каждое устройство имеет working Open Wearables integration (aggregator → webhook → biometric-service)
- Минимум 3 метрики (heart_rate, spo2, sleep) синхронизируются автоматически
- Данные поступают через `POST /api/v1/integrations/open-wearables/webhook` в biometric-service
- UI отображает статус подключения и последнюю синхронизацию

### 24.4 Архитектурные ограничения

- Biometric-service остаётся универсальным: принимает webhook от Open Wearables, валидирует, сохраняет в `biometric_data`
- Device-aggregator используется как легковесный webhook-forwarder для Open Wearables
- Прямые OAuth-интеграции (Fitbit, Withings, Flo, OKOK) удалены из кодовой базы и swagger
