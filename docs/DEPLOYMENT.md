# FitPulse — Развертывание

## Содержание

- [Быстрый старт](#быстрый-старт)
- [Требования к серверу](#требования-к-серверу)
- [Развертывание на Kubernetes](#развертывание-на-kubernetes)
- [Оптимизация для слабых серверов](#оптимизация-для-слабых-серверов)
- [Настройка SMTP (Yandex)](#настройка-smtp-yandex)
- [Регистрация первого администратора](#регистрация-первого-администратора)
- [Troubleshooting](#troubleshooting)

## Быстрый старт

```bash
# 1. Клонировать репозиторий
git clone https://github.com/MAMUER/fitpulse.git && cd fitpulse

# 2. Создать namespace и применить манифесты
kubectl create namespace fitness-platform-production
# 2.1. Создать секреты (обязательно перед apply манифестов!)
kubectl create secret generic app-secrets -n fitness-platform-production \
    --from-literal=POSTGRES_USER=postgres \
    --from-literal=POSTGRES_PASSWORD=<your-password> \
    --from-literal=POSTGRES_DB=fitness
kubectl apply -k configs/k8s/base/ -n fitness-platform-production

# 3. Применить миграции БД
kubectl apply -f configs/k8s/base/jobs/migrate-db.yaml -n fitness-platform-production

# 4. Проверить статус подов
kubectl get pods -n fitness-platform-production

# 5. Health check (порт-forward к поду gateway на 8443)
kubectl port-forward -n fitness-platform-production \
    $(kubectl get pod -n fitness-platform-production -l app=gateway -o jsonpath='{.items[0].metadata.name}') \
    8443:8443
curl -k https://localhost:8443/health
```

## Требования к серверу

### Полный стек

- **ОС**: Linux (Ubuntu 26+, Debian 13+)
- **Kubernetes**: 1.36+ (k3s, k8s)
- **CPU**: 1+ ядро
- **RAM**: 2+ ГБ
- **Диск**: 30+ ГБ SSD
- **Сеть**: HTTPS (порт 8443), TLS 1.3


## Развертывание на Kubernetes

### 1. Установка k3s (рекомендуется для VPS)

```bash
# Установка k3s
curl -sfL https://get.k3s.io | sh -s - --write-kubeconfig-mode 644

# Проверка
kubectl get nodes
kubectl get pods -n kube-system
```

### 2. Применение манифестов

```bash
# Создать namespace
kubectl create namespace fitness-platform-production

# Применить все манифесты
kubectl apply -k configs/k8s/base/ -n fitness-platform-production

# Применить миграции БД
kubectl apply -f configs/k8s/base/jobs/migrate-db.yaml -n fitness-platform-production

# Дождаться готовности
kubectl wait --for=condition=ready pod --all -n fitness-platform-production --timeout=300s
```

### 3. Создание secrets

Создайте секрет `app-secrets` в namespace `fitness-platform-production`:

```bash
# Создание секретов из файлов (предотвращает утечку в bash history / ps)
kubectl create secret generic app-secrets -n fitness-platform-production \
    --from-literal=POSTGRES_USER=postgres \
    --from-literal=POSTGRES_PASSWORD=<your-password> \
    --from-literal=POSTGRES_DB=fitness \
    --from-file=JWT_PRIVATE_KEY_PEM=./key.pem \
    --from-file=JWT_PUBLIC_KEY_PEM=./key.pub \
    --from-literal=RABBITMQ_URL=amqp://user:pass@rabbitmq:5672/ \
    --from-literal=VALKEY_PASSWORD=<valkey-password> \
    --from-literal=SMTP_HOST=smtp.yandex.ru \
    --from-literal=SMTP_PORT=465 \
    --from-literal=SMTP_USER=<your-email> \
    --from-literal=SMTP_PASSWORD=<app-password> \
    --from-literal=SMTP_FROM=<your-email> \
    --from-literal=SMTP_TLS=true \
    --from-literal=APP_BASE_URL=https://your-domain.com \
    --from-literal=SEED_ADMIN_EMAIL=<admin-email> \
    --from-literal=SEED_ADMIN_PASSWORD=<admin-password> \
    --from-literal=TOTP_ENCRYPTION_KEY=<32-byte-key>
```

### 4. Настройка ingress (опционально)

Если используете внешний ingress controller:

```bash
kubectl apply -f configs/k8s/base/ingress-nginx/ -n fitness-platform-production
```

## Оптимизация для слабых серверов

### Ресурсные ограничения

Для серверов с <2GB RAM примените resource quotas:

```bash
kubectl apply -f configs/k8s/base/resource-quota.yaml -n fitness-platform-production
```

### Отключение ML-сервисов

На слабых серверах отключите ML-сервисы:

```bash
kubectl scale deployment classifier --replicas=0 -n fitness-platform-production
kubectl scale deployment ml-generator --replicas=0 -n fitness-platform-production
```

### Swap для серверов с <2GB RAM

```bash
# Создать swap файл 2GB
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

## Настройка SMTP (Yandex)

1. Создать пароль приложения: [passport.yandex.ru/profile](https://passport.yandex.ru/profile) → Безопасность → Пароли приложений
2. Указать параметры SMTP в secrets:
   - `SMTP_HOST=smtp.yandex.ru`
   - `SMTP_PORT=465`
   - `SMTP_TLS=true` — **обязательно для Yandex на порту 465**
   - `SMTP_USER=ваш_email@yandex.ru`
   - `SMTP_PASSWORD=пароль_приложения`
   - `SMTP_FROM=ваш_email@yandex.ru`

## Регистрация первого администратора

```sql
-- Выполнить в БД после применения init-db.yaml:
SELECT create_invite_code('admin', NULL, 1, 365);
-- Вернёт код вида: ADMIN-2026-<hash>
```

Затем зарегистрироваться через `POST /api/v1/register/invite` с этим кодом.

## Troubleshooting

### Pods в статусе CrashLoopBackOff

```bash
# Проверить логи
kubectl logs <pod-name> -n fitness-platform-production --previous

# Проверить события
kubectl describe pod <pod-name> -n fitness-platform-production
```

### ML-сервисы потребляют слишком много памяти

Отключите их (см. раздел "Отключение ML-сервисов" выше).

### База данных недоступна

```bash
# Проверить статус PostgreSQL
kubectl get pods -l app=postgres -n fitness-platform-production

# Выполнить миграции повторно
kubectl delete job migrate-db -n fitness-platform-production
kubectl apply -f configs/k8s/base/jobs/migrate-db.yaml -n fitness-platform-production
```

### Gateway не отвечает на health check

В текущих манифестах gateway служит HTTPS на порту `8443`, а Service `gateway-service` и probes указывают порт `8080`. Для проверки работоспособности используйте port-forward непосредственно к поду:

```bash
# Найти под gateway
POD=$(kubectl get pod -n fitness-platform-production -l app=gateway -o jsonpath='{.items[0].metadata.name}')

# Открыть port-forward к поду на 8443
kubectl port-forward -n fitness-platform-production "$POD" 8443:8443

# Проверить
curl -k https://localhost:8443/health
```

Исправление манифестов: в `configs/k8s/base/deployments/gateway.yaml` замените `containerPort: 8080` на `containerPort: 8443` и обновите `livenessProbe`/`readinessProbe` на порт `8443`. В `configs/k8s/base/services/gateway-service.yaml` замените `targetPort: 8080` на `targetPort: 8443`.

### TLS

Используйте self-signed сертификаты для тестирования:

```bash
# Генерация self-signed сертификата
openssl req -x509 -nodes -days 365 -newkey ec -pkeyopt ec_paramgen_curve:P-256 \
  -keyout deploy/tls/certs/server.key \
  -out deploy/tls/certs/server.crt \
  -subj "/CN=localhost"
```

> **Примечание:** директория `deploy/tls/certs/` в репозитории отсутствует; создайте её локально перед генерацией сертификатов.

## Рекомендации

- Используйте **PersistentVolume** для PostgreSQL и RabbitMQ (не in-memory) — в текущих манифестах уже настроено через `volumeClaimTemplates`
- Настройте **ResourceQuota** и **LimitRange** для namespace
- Используйте **HorizontalPodAutoscaler** для Gateway при высокой нагрузке
- Настройте **backup** PostgreSQL через WAL-архивацию
