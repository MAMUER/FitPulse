# ADR 0012: Device Aggregator Service — Webhook Relay for Open Wearables

## Статус

Принято / Устарел

## Контекст

Изначально архитектура предполагала выделенный `device-aggregator` для унифицированной обработки OAuth авторизации носимых устройств (Fitbit, Withings) с хранением токенов и webhook-интеграцией. В рамках миграции на Open Wearables прямая OAuth-интеграция с отдельными провайдерами удалена из кодовой базы.

## Решение

`device-aggregator` сохранён как легковесный relay/validator для webhook от Open Wearables. Прямые OAuth-роуты (`/devices/fitbit/*`, `/devices/withings/*`) удалены. Основная логика сохранения метрик перенесена в `biometric-service` через `POST /api/v1/integrations/open-wearables/webhook`.

## Последствия

- **Плюсы**: упрощение архитектуры, единая точка сохранения биометрики, снижение количества секретов.
- **Нейтрально**: `device-aggregator` продолжает жить как отдельный deployment, но его нагрузка снижена.
- **Риски**: при отказе Open Wearables интеграция теряется; требуется мониторинг доставки webhook.

## Реализация

- `cmd/device-aggregator/webhooks.go` — общий обработчик webhook для Open Wearables.
- `cmd/device-aggregator/main.go` — роутинг только `/api/v1/integrations/open-wearables/webhook`.
- K8s deployment оставлен без секретов `FITBIT_*`/`WITHINGS_*`.
