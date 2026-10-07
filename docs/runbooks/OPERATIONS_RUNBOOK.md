# Runbook: Операции платформы FitPulse

## Содержание

1. [Экстренное реагирование](#экстренное-реагирование)
2. [Процедуры деплоя](#процедуры-деплоя)
3. [Индивидуальный ответ на инциденты](#индивидуальный-ответ-на-инциденты)
4. [Мониторинг и алерты](#мониторинг-и-алерты)
5. [Восстановление данных](#восстановление-данных)

---

## Экстренное реагирование

### SEV-1: Сервис недоступен (SLA восстановления < 5 минут)

**Симптомы**: сервис возвращает 503, health-check падает, высокий error rate.

**Шаги**:

1. **Проверить статус сервисов**

   ```bash
   kubectl get pods -n fitness-platform-production
   kubectl describe pod <pod-name> -n fitness-platform-production
   ```

2. **Проверить логи**

   ```bash
   kubectl logs <pod-name> -n fitness-platform-production --tail=100
   ```

3. **Перезапустить pod** (при `OOMKilled` или `CrashLoopBackOff`)

   ```bash
   kubectl rollout restart deployment/<deployment-name> -n fitness-platform-production
   ```

4. **Откат** если проблема появилась после недавнего деплоя

   ```bash
   kubectl rollout undo deployment/gateway -n fitness-platform-production
   ```

5. **Эскалировать Tech Lead**, если проблема не решена за 5 минут.

---

### SEV-2: Высокий error rate (15–30 минут расследования)

**Симптомы**: error rate > 1%, p95 latency > 5s.

**Шаги**:

1. Открыть Grafana дашборд «FitPulse Service Overview».
2. Изучить логи сервисов на паттерны:

   ```bash
   kubectl logs -f deployment/<service> -n fitness-platform-production | grep -i "error\|panic"
   ```

3. Масштабировать сервис, если исчерпан пул соединений к БД:

   ```bash
   kubectl scale deployment biometric-service --replicas=3 -n fitness-platform-production
   ```

4. Проверить репликацию БД:

   ```bash
   psql -h postgres -U postgres -d fitness -c "SELECT slot_name, restart_lsn FROM pg_replication_slots;"
   ```

---

## Процедуры деплоя

### Ручной откат

```bash
# Просмотреть историю rollout
kubectl rollout history deployment/gateway -n fitness-platform-production

# Откатиться на предыдущую версию
kubectl rollout undo deployment/gateway -n fitness-platform-production

# Откатиться на определённую ревизию
kubectl rollout undo deployment/gateway --to-revision=5 -n fitness-platform-production

# Проверить результат отката
kubectl get pods -n fitness-platform-production -l app=gateway
kubectl logs -f deployment/gateway -n fitness-platform-production
```

---

## Индивидуальный ответ на инциденты

### Исчерпан пул соединений к PostgreSQL (SEV-1)

**Алерт**: `db_connection_pool_usage > 0.9`

**Шаги**:

1. **Проверить активные соединения**

   ```sql
   SELECT datname, count(*) FROM pg_stat_activity GROUP BY datname;
   ```

2. **Найти долгие запросы**

   ```sql
   SELECT query, duration FROM pg_stat_statements 
   ORDER BY duration DESC LIMIT 5;
   ```

3. **Масштабировать реплики сервиса**

   ```bash
   kubectl scale deployment user-service --replicas=5 -n fitness-platform-production
   ```

4. **Мониторить восстановление пула** через Grafana дашборд «Database Performance».

---

### Неудачный бэкап (SEV-1)

**Алерт**: `backup_success{type='full'} == 0`

**Шаги**:

1. **Проверить логи job'ы бэкапа**

    ```bash
    kubectl get pods -n fitness-platform-production -l job-name=backup
    kubectl logs -f backup-job -n fitness-platform-production
    ```

2. **Проверить доступность MinIO**

    ```bash
    curl -s http://minio.minio.svc.cluster.local:9000/minio/health/ready
    ```

3. **Запустить бэкап вручную**

    ```bash
    # Скрипт бэкапа с MinIO backend
    bash scripts/backup-db-with-minio.sh
    ```

4. **Проверить, что бэкап попал в MinIO**

    ```bash
    curl -s http://minio.minio.svc.cluster.local:9000/minio/mybucket/ -u "$MINIO_ACCESS_KEY:$MINIO_SECRET_KEY"
    ```

---

### Низкая уверенность ML-модели (SEV-4)

**Алерт**: `classification_confidence < 0.7`

**Шаги**:

1. **Проверить версии моделей**

   ```bash
   kubectl logs -f deployment/classifier -n fitness-platform-production | grep -i "model version"
   ```

2. **Изучить последние предсказания**

   ```bash
   kubectl logs -f deployment/classifier -n fitness-platform-production | grep "CLASSIFY"
   ```

3. **Создать тикет** для ML-команды для расследования дрифта.

---

### Device Aggregator: OAuth/webhook сбои (SEV-2)

**Симптомы**: пользователи не видят данные от источников здоровья, webhook от Open Wearables не доставляется.

**Шаги**:

1. **Проверить logs biometric-service**

    ```bash
    kubectl logs -f deployment/biometric-service -n fitness-platform-production | grep -i "webhook\|error\|panic"
    ```

2. **Проверить health webhook**

    ```bash
    curl http://biometric-service:8085/health
    ```

3. **Проверить метрики webhook**

    ```bash
    curl http://biometric-service:8085/metrics | grep -i "webhook"
    ```

4. **Проверить статус источника** в UI и при необходимости отключить/подключить источник через Open Wearables.

---

## Мониторинг и алерты

### Ключевые метрики

|Метрика|Порог|Частота проверки|
|---|---|---|
|Error Rate|< 1%|Непрерывно (1 мин)|
|p95 Latency|< 5s|Непрерывно (1 мин)|
|Uptime|> 99.9%|Ежедневно|
|DB Pool Usage|< 80%|Каждые 5 мин|
|Backup Success|100%|Каждые 6 ч|
|ML Confidence|> 0.7|Каждые 15 мин|

### Доступ к Grafana

```text
URL: https://fittpulse.duckdns.org
Username: admin
Password: ${GRAFANA_ADMIN_PASSWORD}
```

**Стандартные дашборды**:

- `FitPulse Service Overview`: request rate, error rate, latency, ML метрики
- `Database Performance`: соединения, время запросов, репликация

---

## Восстановление данных

### PostgreSQL: восстановление из бэкапа (MinIO)

```bash
# 1. Остановить текущий инстанс PostgreSQL
kubectl scale deployment/postgres --replicas=0 -n fitness-platform-production

# 2. Восстановить из бэкапа
kubectl exec -i -n fitness-platform-production postgres-0 -- \
  pg_restore -U postgres -d fitness /tmp/fitness-backup.dump

# 3. Проверить целостность данных
psql -h localhost -U postgres -d fitness \
  -c "SELECT COUNT(*) FROM users; SELECT MAX(created_at) FROM biometric_data;"

# 4. Запустить PostgreSQL
kubectl scale deployment/postgres --replicas=1 -n fitness-platform-production
```

### Восстановление из WAL-архива (PITR, MinIO)

```bash
# 1. Скачать последний бэкап из MinIO
curl -s http://minio.minio.svc.cluster.local:9000/minio/mybucket/backup-fitness-LATEST.dump.enc \
  -u "$MINIO_ACCESS_KEY:$MINIO_SECRET_KEY" -o /tmp/backup.dump.enc

# 2. Расшифровать бэкап
openssl enc -d -aes-256-cbc -salt -pbkdf2 \
  -pass pass:"$BACKUP_KEY" \
  -in /tmp/backup.dump.enc -out /tmp/backup.dump

# 3. Скачать WAL-файлы из MinIO
mkdir -p /tmp/wal-recovery
curl -s http://minio.minio.svc.cluster.local:9000/minio/mybucket/wal/ -u "$MINIO_ACCESS_KEY:$MINIO_SECRET_KEY" \
  | grep -o 'href="[^"]*"' | cut -d'"' -f2 | while read wal; do
    curl -s "http://minio.minio.svc.cluster.local:9000${wal}" -o "/tmp/wal-recovery/$(basename $wal)"
  done

# 4. Восстановить в clone-окружении
initdb -D /tmp/clone_pgdata
cp /tmp/wal-recovery/postgresql.conf /tmp/clone_pgdata/
pg_ctl -D /tmp/clone_pgdata \
  -o "-c restore_command='cp /tmp/wal-recovery/%f %p'" start

# 5. После завершения recovery, promote к primary
pg_ctl -D /tmp/clone_pgdata promote
```

### Ежеквартальный Recovery Drill

**Цель**: Проверить, что бэкапы работают и восстановление занимает < 1 часа.

**Частота**: Раз в квартал (январь, апрель, июль, октябрь)

**Процедура**:

1. **Подготовка** (за 1 неделю до drill):
   - Уведомить команду о предстоящем drill
   - Проверить наличие свежего бэкапа (< 24 часов) в MinIO
   - Проверить доступность MinIO (`curl -s http://minio.minio.svc.cluster.local:9000/minio/health/ready`)
   - Подготовить clone-окружение (отдельный namespace или cluster)

2. **Выполнение drill**:

    ```bash
    # Запустить chaos test suite
    bash scripts/chaos-test-all.sh fitness-platform-production
    
    # Или выполнить вручную:
    # 1. Остановить PostgreSQL
    kubectl scale deployment/postgres --replicas=0 -n fitness-platform-production
    
    # 2. Восстановить из бэкапа в clone-среде
    bash scripts/restore-to-clone.sh minio://mybucket/backup-fitness-LATEST.dump.enc
    
    # 3. Проверить целостность данных
    psql -h clone-postgres -U postgres -d fitness -c "SELECT COUNT(*) FROM users;"
    
    # 4. Замерить время восстановления (RTO)
    # 5. Проверить RPO (потеря данных = время последнего бэкапа)
    ```

3. **Критерии успеха**:
   - ✅ RTO < 1 часа
   - ✅ RPO < 24 часа (с WAL-архивацией в MinIO)
   - ✅ Данные целостны (проверка по контрольным суммам)
   - ✅ Сервис доступен после восстановления

4. **Документирование**:
   - Заполнить шаблон отчёта: `docs/compliance/ШАБЛОН_ОТЧЁТА_RECOVERY_DRILL.md`
   - Опубликовать отчёт в репозитории (без чувствительных данных)
   - Создать action items при обнаружении проблем

5. **Action Items**:
   - Критические: исправить в течение 1 недели
   - Средние: исправить в течение 1 месяца
   - Низкие: добавить в backlog

---

## Контакты и эскалация

- **Tech Lead**: [mihnikolaenko12@yandex.ru](mailto:mihnikolaenko12@yandex.ru)
- **CTO**: [mihnikolaenko12@yandex.ru](mailto:mihnikolaenko12@yandex.ru) (только SEV-1, эскалация после 15 мин)

---

## Справочник сервисов

> Канонический список сервисов и endpoints: см. `INCIDENT_RESPONSE.md` → «Работа с новыми сервисами».

|Сервис|Namespace label|Health endpoint|Логи|
|---|---|---|---|
|Gateway|`app=gateway`|`https://fittpulse.duckdns.org/health`|`kubectl logs -f deployment/gateway`|
|User Service|`app=user-service`|gRPC health|`kubectl logs -f deployment/user-service`|
|Biometric Service|`app=biometric-service`|gRPC health + `http://biometric-service:8085/health`|`kubectl logs -f deployment/biometric-service`|
|Classifier|`app=classifier`|`http://classifier:8001/health`|`kubectl logs -f deployment/classifier`|
|ML Generator|`app=ml-generator`|`http://ml-generator:8002/health`|`kubectl logs -f deployment/ml-generator`|
|Device Aggregator|`app=device-aggregator`|`http://device-aggregator:8084/health`|`kubectl logs -f deployment=device-aggregator`|
|Data Processor|`app=data-processor`|gRPC health|`kubectl logs -f deployment/data-processor`|
|Admin CLI|`app=admin-cli`|CLI tool (no HTTP health)|N/A (client-side)|
|Valkey|`app=valkey`|`http://valkey:6379/health` или `redis-cli ping`|`kubectl logs -f deployment/valkey`|
|RabbitMQ|`app=rabbitmq`|`http://rabbitmq:15672/health`|`kubectl logs -f statefulset/rabbitmq`|

---

## Quarterly Access Review

Процедура quarterly access review описана в отдельном runbook: `docs/runbooks/QUARTERLY_ACCESS_REVIEW.md`.

Проверяйте доступы к:

- Kubernetes RBAC (ServiceAccounts, Roles, RoleBindings)
- PostgreSQL roles
- Vault policies и токены
- GitHub collaborators
- CI/CD secrets
- Grafana/Alertmanager/Prometheus
- Valkey (ключи, TTL)
- RabbitMQ (пользователи, очереди)

---

**Последнее обновление**: 2026-07-15  
**Ведёт**: Platform Team
