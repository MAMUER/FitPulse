# ADR 0003: Матрица окружений — Dev/Test/Staging/Prod

## Статус

Принято

## Контекст

Проект требует единообразной конфигурации для окружений разработки, staging и production. В текущей реализации присутствуют Dev (docker-compose), Staging и Production (Kubernetes через Kustomize overlays). Отдельное постоянное тестовое окружение не развёрнуто и не планируется из-за ограниченных ресурсов (один VPS). Интеграционные тесты используют testcontainers-go локально и в CI через ephemeral namespace. Staging используется для ручного/автоматизированного тестирования перед production.

## Решение

Реализовать матрицу конфигураций для Dev/Staging/Production:

- **Dev**: `docker-compose.yml` для локальной разработки, HTTP, без бэкапов.
- **Staging**: Kustomize overlay в `configs/k8s/overlays/staging`, Let's Encrypt staging, HPA для core-сервисов. Используется для ручного/автоматизированного тестирования перед production.
- **Production**: Kustomize overlay в `configs/k8s/overlays/production`, Let's Encrypt production, daily backups + WAL archive.

Отдельное **Test** окружение как постоянное окружение **не планируется** из-за ограниченных ресурсов (один VPS). Интеграционные тесты используют `testcontainers-go` локально и в CI через ephemeral namespace.

Ключевые параметры:

- K8s подов на сервис (base: 1-2, HPA в staging/production: 2-5 с CPU autoscaling)
- PostgreSQL топология (StatefulSet с 1 инстансом + WAL archive + daily full backup в S3; replicas не развёрнуты)
- Valkey топология (Deployment с 1 инстансом; Sentinel/Cluster не развёрнуты)
- GPU ресурсы (ml-generator работает на CPU; T4/A10 не настроены)
- Стек мониторинга (docker-compose: Prometheus+Grafana; K8s: ELK+Prometheus+Alertmanager+OnCall)
- Стратегия бэкапов (Dev: нет; Staging/Prod: daily full backup + WAL archive в S3/MinIO)
- SSL/TLS (Dev: HTTP; Staging: Let's Encrypt staging; Production: Let's Encrypt production)
- Контроль доступа (Dev: локальный; Staging/Prod: Ingress + basic auth; VPN/2FA/hardware token не настроены)

## Последствия

- обеспечивает единообразные практики между Dev и K8s окружениями;
- облегчает переход от локальной разработки к staging/production через Kustomize overlays;
- Staging и Production имеют WAL archiving и daily backups; Dev работает без бэкапов.

## Реализация

- `docker-compose.yml` — Dev окружение: все сервисы в одном хосте, HTTP на порту 8080, Valkey/PostgreSQL/RabbitMQ без кластеризации.
- `configs/k8s/base/` — базовые Kubernetes-манифесты: StatefulSet PostgreSQL (1 replica), Deployment Valkey (1 replica), deployments для сервисов (1-2 replica), backup jobs, cert-manager, monitoring.
- `configs/k8s/overlays/staging/` — Staging overlay: Kustomize с images tags, staging ingress, HPA (2-5 replicas), Let's Encrypt staging issuer.
- `configs/k8s/overlays/production/` — Production overlay: Kustomize с images tags, production ingress, HPA, jaeger ingress, TLS role, Let's Encrypt production issuer.
- `configs/k8s/base/jobs/` — backup cronjobs: daily full backup + WAL archive в S3/MinIO.

## Рассмотренные альтернативы

- Единая конфигурация с оверрайдами: менее чёткое разделение окружений.
- Ручная конфигурация под каждое окружение: подвержена ошибкам и неконсистентности.
