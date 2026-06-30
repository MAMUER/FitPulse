# ADR 0012: Device Aggregator Service — Dedicated OAuth Integration Layer

## Статус

Принято

## Контекст

Архитектура требует централизованной обработки OAuth-авторизации носимых устройств с поддержкой CSRF-защиты, хранения токенов и webhook-интеграции. Исходный проект предполагал адаптерный подход к устройствам, но не описывал отдельный сервис-агрегатор как компонент. В результате появился новый микросервис `cmd/device-aggregator`.

## Решение

Выделить отдельный микросервис `device-aggregator` (Go, chi/v5, порт 8083) для:

- унифицированного OAuth-flow по Fitbit, Garmin и последующим провайдерам;
- хранения и обновления OAuth-токенов в БД (`device_provider_accounts`);
- защиты от CSRF через `oauth_states` с TTL;
- проброса OAuth-запросов из Gateway через `proxyToDeviceAggregator`;
- приёма webhook-уведомлений от провайдеров.

Плюсы такого:

- изоляция OAuth-логики от остальных микросервисов;
- переиспользуемость: Gateway и Device Connector обращаются к тому же агрегатору;
- независимое масштабирование и мониторинг.

## Последствия

- **Плюсы**: чёткое разделение ответственности, изоляция секретов/токенов от других доменов, единая точка OAuth-логики.
- **Нейтрально**: новый сервис требует своих Deployment/Service в K8s, health-check’ов и версии в CI/CD.
- **Риски**: должен быть отказоустойчивым, так как хранит токены; при сбое — переавторизация пользователей.

## Реализация

- `cmd/device-aggregator/main.go` — сервер и маршрутизация.
- `cmd/device-aggregator/providers/fitbit.go` — Fitbit OAuth 2.0 интеграция.
- `cmd/device-aggregator/providers/garmin.go` — Garmin Health API OAuth 1.0a заглушка.
- `cmd/device-aggregator/providers/withings.go` — Withings заглушка.
- `cmd/device-aggregator/aggregator.go` — общий интерфейс провайдеров.
- `cmd/device-aggregator/webhooks.go` — обработка webhook’ов.
- `cmd/device-aggregator/Dockerfile` — контейнеризация.
- `configs/k8s/deployments/device-aggregator.yaml` — K8s манифест.
