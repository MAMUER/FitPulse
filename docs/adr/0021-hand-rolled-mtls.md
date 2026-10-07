# ADR 0021: Hand-rolled mTLS для gRPC вместо Service Mesh (Phase 1)

## Статус

Принято

## Контекст

FitPulse состоит из нескольких микросервисов (gateway, user-service, biometric-service, training-service, classifier, ml-generator, device-aggregator, data-processor), которые общаются между собой по gRPC. Требуется:

- Аутентификация и авторизация между сервисами
- Защита от lateral movement
- Возможность ротации сертификатов без даунтайма

Доступные подходы:

1. **Service Mesh** (Istio/Linkerd) — автоматический mTLS, sidecar-инъекция, traffic management.
2. **Hand-rolled mTLS** — ручная выработка и валидация сертификатов через Kubernetes Secrets.
3. **No mTLS** — только аутентификация на уровне приложения (JWT).

## Решение

Использовать **hand-rolled mTLS на TLS 1.3** для gRPC-коммуникаций в Phase 1:

1. Сертификаты генерируются один раз и хранятся в Kubernetes Secret (`tls-certs`).
2. Каждый сервис использует `credentials.NewServerTLSFromFile` и `credentials.NewClientTLSFromFile`.
3. Gateway валидирует сертификаты сервисов через `x509.NewCertPool`.
4. Переход на Service Mesh (Istio/Linkerd) запланирован на Phase 2.

## Последствия

- **Плюсы**: полный контроль над TLS-конфигурацией, минимальный overhead, не требуется sidecar-инъекция.
- **Нейтрально**: требуется ручная ротация сертификатов в Phase 1; автоматизация через cert-manager уже реализована (`configs/k8s/base/cert-manager/grpc-selfsigned-ca.yaml`, `configs/k8s/base/deployments/grpc-server-cert.yaml`) с 90-дневным сроком и автообновлением за 30 дней.
- **Риски**: при росте числа сервисов управление сертификатами становится сложнее. Митигация: переход на Service Mesh в Phase 2.

## Рассмотренные альтернативы

- **Istio/Linkerd**: слишком heavy для Phase 1 (1+ ГБ RAM на control plane), сложность настройки для 1-узельного k3s.
- **No mTLS**: риск lateral movement, сниженная безопасность.

## Реализация

- `internal/grpc/tls.go` — утилиты для загрузки сертификатов.
- `cmd/gateway/main.go` — gateway TLS-конфигурация с валидацией сертификатов сервисов.
- `configs/k8s/base/secrets/tls-certs.yaml` — Kubernetes Secret с сертификатами.
