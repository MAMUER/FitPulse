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

1. **Подтвердить алерт** (Slack/PagerDuty)

   ```bash
   # Реакция :ack: в Slack или подтверждение в PagerDuty
   ```

2. **Проверить статус сервисов**

   ```bash
   kubectl get pods -n fitness-platform
   kubectl describe pod <pod-name> -n fitness-platform
   ```

3. **Проверить логи**

   ```bash
   kubectl logs <pod-name> -n fitness-platform --tail=100
   # Или через Kibana: индекс "fitness-logs-*", фильтр service="gateway"
   ```

4. **Перезапустить pod** (при `OOMKilled` или `CrashLoopBackOff`)

   ```bash
   kubectl delete pod <pod-name> -n fitness-platform
   # Pod будет пересоздан deployment controller'ом
   ```

5. **Откат** если проблема появилась после недавнего деплоя

   ```bash
   kubectl rollout undo deployment/gateway -n fitness-platform
   ```

6. **Эскалировать Tech Lead**, если проблема не решена за 5 минут.

---

### SEV-2: Высокий error rate (15–30 минут расследования)

**Симптомы**: error rate > 5%, p95 latency > 5s.

**Шаги**:

1. Открыть Grafana дашборд «FitPulse Service Overview».
2. Изучить логи в Kibana на паттерны:

   ```json
   {
     "level": "ERROR",
     "service": "biometric-service",
     "timestamp": "2026-05-06T*"
   }
   ```

3. Масштабировать сервис, если исчерпан пул соединений к БД:

   ```bash
   kubectl scale deployment biometric-service --replicas=3 -n fitness-platform
   ```

4. Проверить репликацию БД:

   ```bash
   psql -h postgres -U postgres -d fitness -c "SELECT slot_name, restart_lsn FROM pg_replication_slots;"
   ```

---

## Процедуры деплоя

### Канарный деплой (9-этапный пайплайн)

```bash
# Этапы 1–3: Development, Code Review, CI Build (автоматически)

# Этап 4: Деплой в Test
kubectl set image deployment/gateway-test \
  gateway=fitness-gateway:<IMAGE_SHA> \
  -n fitness-platform

# Подождать 5 минут, проверить health
kubectl get pods -n fitness-platform -l app=gateway-test

# Этап 5: Деплой в Staging (UAT, perf/security тесты)
kubectl set image deployment/gateway-staging \
  gateway=fitness-gateway:<IMAGE_SHA> \
  -n fitness-platform

# Запустить нагрузочные и security-тесты
make k6-load-test
make owasp-scan

# Этап 6: Создание Release Candidate
git tag v2.1.0-rc1
git push origin v2.1.0-rc1

# Этап 7: Канарный деплой в Production (10% трафика, 1 час)
kubectl patch service gateway-canary -p \
  '{"spec":{"selector":{"version":"canary"}}}'

# Мониторить в течение 1 часа
kubectl get hpa -n fitness-platform -w

# Критерии успеха:
# - Error rate < 1%
# - p95 latency < 3s
# - Нет критических логов

# Этап 7b: Rolling Deploy (30% → 60% → 100%, интервалы 30 мин)
kubectl set image deployment/gateway \
  gateway=fitness-gateway:<IMAGE_SHA> \
  -n fitness-platform

# Этап 8: Пост-деплойный мониторинг (24 часа)
# Дашборды: Error Rate, Latency, ML Confidence, DB Pool, Backup Status

# Этап 9: Автоматический откат при нарушении критериев:
# - error_rate > 5% в течение 15 минут
# - latency p95 > 10s в течение 15 минут
# - КРИТИЧЕСКАЯ security-проблема
kubectl rollout undo deployment/gateway -n fitness-platform
```

### Ручной откат

```bash
# Просмотреть историю rollout
kubectl rollout history deployment/gateway -n fitness-platform

# Откатиться на предыдущую версию
kubectl rollout undo deployment/gateway -n fitness-platform

# Откатиться на определённую ревизию
kubectl rollout undo deployment/gateway --to-revision=5 -n fitness-platform

# Проверить результат отката
kubectl get pods -n fitness-platform -l app=gateway
kubectl logs -f deployment/gateway -n fitness-platform
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
   kubectl scale deployment user-service --replicas=5 -n fitness-platform
   ```

4. **Мониторить восстановление пула**

   ```bash
   # Следить за метрикой db_connection_pool_usage в Grafana
   ```

### Неудачный бэкап (SEV-1)

**Алерт**: `backup_success{type='full'} == 0`

**Шаги**:

1. **Проверить логи job'ы бэкапа**

   ```bash
   kubectl get pods -n fitness-platform -l job-name=backup
   kubectl logs -f backup-job -n fitness-platform
   ```

2. **Проверить свободное место**

   ```bash
   df -h /var/lib/postgresql/data
   ```

3. **Запустить бэкап вручную**

   ```bash
   # Через скрипт
   scripts/backup-db.sh --encrypted --s3-upload

   # Проверить
   aws s3 ls s3://fitness-backups/
   ```

### Низкая уверенность ML-модели (SEV-4)

**Алерт**: `classification_confidence < 0.7`

**Шаги**:

1. **Проверить версии моделей**

   ```bash
   curl http://classifier:8001/model-info
   ```

2. **Изучить последние предсказания**

   ```bash
   # Kibana: service="classifier" AND action="CLASSIFY" AND confidence < 0.7
   ```

3. **Запустить переобучение модели**

   ```bash
   # Через admin endpoint ML-сервиса
   curl -X POST http://classifier:8001/retrain \
     -H "Authorization: Bearer $ML_ADMIN_TOKEN"
   ```

4. **Создать тикет** для ML-команды для расследования дрифта.

### Device Aggregator: OAuth/webhook сбои (SEV-2)

**Симптомы**: пользователи не могут подключить Fitbit/Garmin/Withings, webhook'и не доставляются.

**Шаги**:

1. **Проверить logs device-aggregator**

   ```bash
   kubectl logs -f deployment/device-aggregator -n fitness-platform | grep -i "error\|panic"
   ```

2. **Проверить health**

   ```bash
   curl http://device-aggregator:8083/health
   ```

3. **Проверить токены в БД**

   ```sql
   SELECT provider, is_active, last_sync_at 
   FROM device_provider_accounts 
   WHERE is_active = FALSE 
   ORDER BY updated_at DESC LIMIT 10;
   ```

4. **Переавторизовать проблемного пользователя** через `/api/v1/devices/fitbit/auth`.

---

## Мониторинг и алерты

### Ключевые метрики

|Метрика|Порог|Частота проверки|
|---|---|---|
|Error Rate|< 5%|Непрерывно (1 мин)|
|p95 Latency|< 5s|Непрерывно (1 мин)|
|Uptime|> 99.9%|Ежедневно|
|DB Pool Usage|< 80%|Каждые 5 мин|
|Backup Success|100%|Каждые 6 ч|
|ML Confidence|> 0.7|Каждые 15 мин|

### Доступ к Grafana

```text
URL: https://grafana.fitpulse.app:3000
Username: admin
Password: ${GRAFANA_ADMIN_PASSWORD}
```

**Стандартные дашборды**:

- `FitPulse Service Overview`: request rate, error rate, latency, ML метрики
- `Database Performance`: соединения, время запросов, репликация
- `ELK Stack Health`: индексы Elasticsearch, пропускная способность Logstash

### Elasticsearch Snapshot и 90-дневная ротация

```bash
# Создать snapshot repository (единоразовая настройка)
curl -X PUT "elasticsearch:9200/_snapshot/s3-backup" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "s3",
    "settings": {
      "bucket": "fitness-logs-backup",
      "base_path": "snapshots"
    }
  }'

# Создать ежедневный snapshot
curl -X PUT "elasticsearch:9200/_snapshot/s3-backup/snapshot-$(date +%Y-%m-%d)" \
  -H "Content-Type: application/json" \
  -d '{"indices": "fitness-logs-*"}'

# Архивировать старые индексы (> 90 дней) в холодное хранилище
curl -X POST "elasticsearch:9200/fitness-logs-2026.01.01/_close"
aws s3 cp elasticsearch-snapshot-2026.01.01.tar.gz s3://fitness-logs-archive/
```

---

## Восстановление данных

### PostgreSQL Point-in-Time Recovery (PITR)

```bash
# 1. Остановить текущий инстанс PostgreSQL
kubectl scale deployment/postgres --replicas=0 -n fitness-platform

# 2. Восстановить из бэкапа
scripts/restore-db.sh --backup-file=fitness-backup-2026-05-05.sql.enc \
  --target-time="2026-05-05T14:30:00Z" \
  --use-wal-archive

# 3. Проверить целостность данных
psql -h localhost -U postgres -d fitness \
  -c "SELECT COUNT(*) FROM users; SELECT MAX(created_at) FROM biometric_data;"

# 4. Запустить PostgreSQL
kubectl scale deployment/postgres --replicas=1 -n fitness-platform

# 5. Мониторить репликацию на реплики
kubectl logs -f postgres-replica-0 -n fitness-platform
```

### Восстановление Elasticsearch

```bash
# 1. Список доступных snapshots
curl "elasticsearch:9200/_snapshot/s3-backup/_all"

# 2. Восстановить конкретные индексы
curl -X POST "elasticsearch:9200/_snapshot/s3-backup/snapshot-2026-05-05/_restore" \
  -H "Content-Type: application/json" \
  -d '{
    "indices": "fitness-logs-2026.05.05",
    "rename_pattern": "(.+)",
    "rename_replacement": "$1-restored"
  }'

# 3. Проверить восстановленные индексы
curl "elasticsearch:9200/_cat/indices?v" | grep restored

# 4. При необходимости объединить с живыми индексами
```

---

## Контакты и эскалация

- **Дежурный инженер**: расписание в PagerDuty
- **Tech Lead**: [tech-lead@fitpulse.app](mailto:tech-lead@fitpulse.app)
- **CTO**: [cto@fitpulse.app](mailto:cto@fitpulse.app) (только SEV-1, эскалация после 15 мин)

---

## Справочник сервисов

|Сервис|Namespace label|Health endpoint|Логи|
|---|---|---|---|
|Gateway|`app=gateway`|`https://<host>:8443/health`|`kubectl logs -f deployment/gateway`|
|User Service|`app=user-service`|gRPC health|`kubectl logs -f deployment/user-service`|
|Biometric Service|`app=biometric-service`|gRPC health|`kubectl logs -f deployment/biometric-service`|
|Training Service|`app=training-service`|gRPC health|`kubectl logs -f deployment/training-service`|
|Device Connector|`app=device-connector`|`http://device-connector:8082/health`|`kubectl logs -f deployment/device-connector`|
|Device Aggregator|`app=device-aggregator`|`http://device-aggregator:8083/health`|`kubectl logs -f deployment/device-aggregator`|
|Classifier|`app=classifier`|`http://classifier:8001/health`|`kubectl logs -f deployment/classifier`|
|ML Generator|`app=ml-generator`|`http://ml-generator:8002/health`|`kubectl logs -f deployment/ml-generator`|

---

**Последнее обновление**: 2026-06-07  
**Ведёт**: Platform Team
