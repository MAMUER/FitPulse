# ADR 0023: Выбор Kubernetes как платформы деплоя

## Статус

Принято

## Контекст

Система состоит из нескольких микросервисов (gateway, user-service, biometric-service, training-service, device-aggregator, ml_generator) и инфраструктурных компонентов (PostgreSQL, Valkey, RabbitMQ). Требуется платформа, которая обеспечивает:
- оркестрацию контейнеров;
- service discovery;
- declarative конфигурацию;
- самоисправление (self-healing);
- масштабирование;
- управление секретами;
- сетевую сегментацию.

## Решение

Использовать Kubernetes (k3s) как платформу деплоя:

- **Manifest-based деплой**: YAML-манифесты в `configs/k8s/base/` и окружение-специфичные overlay через Kustomize.
- **k3s**: лёгкое Kubernetes-распределение для VPS/edge; достаточно для production при наличии 2+ нод.
- **Helm**: не используется на текущем этапе; Kustomize хватает для declared-config подхода.
- **Ingress NGINX Controller**: единый ingress с ModSecurity WAF.
- **cert-manager**: автоматическое получение TLS-сертификатов от Let's Encrypt.
- **Network Policies**: сегментация на dmz/app-zone/data-zone/monitoring-zone.

## Последствия

- **Плюсы**: стандартная экосистема, большое community, готовые решения для observability и безопасности.
- **Плюсы**: Kustomize позволяет переиспользовать base и добавлять окружение-специфичные изменения без helm-шаблонов.
- **Нейтрально**: требуется обучение команды Kubernetes; YAML-манифесты могут расти в объёме.
- **Риски**: при росте числа сервисов могут понадобиться Helm chart'ы или GitOps-инструменты (ArgoCD).

## Рассмотренные альтернативы

- **Docker Compose**: простой для dev, но нет self-healing, rolling updates и production-grade networking.
- **Nomad**: проще Kubernetes, но меньше экосистемы и community.
- **HashiCorp Waypoint**: слишком opinionated, не подходит для микросервисов.
- **Fargate/EKS/GKE**: managed Kubernetes, но дороже и менее гибко для self-hosted VPS.

## Реализация

- `configs/k8s/base/` — Deployment, Service, ConfigMap, Secret, NetworkPolicy, RBAC.
- `configs/k8s/base/security-zones.yaml` — Network Policies.
- `configs/k8s/base/rbac/` — RBAC-роли.
- `configs/k8s/base/deployments/` — манифесты для каждого сервиса.
- `k3s-config-template.yaml` — шаблон k3s config с TLS SAN.
- CI/CD: `kubectl apply -k configs/k8s/base/` для деплоя.
