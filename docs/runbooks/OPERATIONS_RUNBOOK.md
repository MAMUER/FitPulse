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

2. **Проверить свободное место**

   ```bash
   df -h /var/lib/postgresql/data
   ```

3. **Запустить бэкап вручную**

   ```bash
   kubectl exec -n fitness-platform-production postgres-0 -- \
     pg_dump -U postgres -d fitness -F c > /tmp/fitness-manual-$(date +%Y%m%d_%H%M%S).dump
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

**Симптомы**: пользователи не могут подключить Fitbit/Garmin/Withings, webhook'и не доставляются.

**Шаги**:

1. **Проверить logs device-aggregator**

   ```bash
   kubectl logs -f deployment/device-aggregator -n fitness-platform-production | grep -i "error\|panic"
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

### PostgreSQL: восстановление из pg_dump

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

---

## Контакты и эскалация

- **Tech Lead**: [tech-lead@fitpulse.app](mailto:tech-lead@fitpulse.app)
- **CTO**: [cto@fitpulse.app](mailto:cto@fitpulse.app) (только SEV-1, эскалация после 15 мин)

---

## Справочник сервисов

|Сервис|Namespace label|Health endpoint|Логи|
|---|---|---|---|
|Gateway|`app=gateway`|`https://fittpulse.duckdns.org/health`|`kubectl logs -f deployment/gateway`|
|User Service|`app=user-service`|gRPC health|`kubectl logs -f deployment/user-service`|
|Biometric Service|`app=biometric-service`|gRPC health|`kubectl logs -f deployment/biometric-service`|
|Training Service|`app=training-service`|gRPC health|`kubectl logs -f deployment/training-service`|
|Device Connector|`app=device-connector`|`http://device-connector:8082/health`|`kubectl logs -f deployment/device-connector`|
|Device Aggregator|`app=device-aggregator`|`http://device-aggregator:8083/health`|`kubectl logs -f deployment/device-aggregator`|
|Classifier|`app=classifier`|`http://classifier:8001/health`|`kubectl logs -f deployment/classifier`|
|ML Generator|`app=ml-generator`|`http://ml-generator:8002/health`|`kubectl logs -f deployment/ml-generator`|

---

**Последнее обновление**: 2026-07-15  
**Ведёт**: Platform Team
