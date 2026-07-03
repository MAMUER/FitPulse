# Playbook: Ответ на инциденты FitPulse

## Обзор

Данный документ определяет процесс реагирования на security-инциденты, простои сервисов и утечки данных. Playbook регулярно обновляется по мере добавления новых компонентов (Device Aggregator, Device Connector, ML-сервисы).

---

## Классификация инцидентов

### Уровни серьёзности

|Уровень|Влияние|Время реакции|Время исправления|Эскалация|
|---|---|---|---|---|
|**SEV-1**|Полный downtime сервиса, риск потери данных|15 мин|1 час|Немедленный PagerDuty → Tech Lead → CTO|
|**SEV-2**|Сервис деградирован, частичная потеря функциональности|1 час|4 часа|Slack → дежурный инженер → Tech Lead|
|**SEV-3**|Мелкие проблемы, влияние на UX|4 часа|2 недели|Slack → дежурный инженер|
|**SEV-4**|Нет непосредственного влияния на пользователей|24 часа|Следующий релиз|Очередь тикетов|

---

## SEV-1 Response: Критический инцидент

### Фаза 1: Триаж (0–5 минут)

1. **Подтверждение алерта**
   - PagerDuty: нажать «Acknowledge» немедленно.
   - Slack: поставить реакцию `:ack:` в канал `#alerts`.

2. **Объявление инцидента**

   ```bash
   # Создать инцидент в Slack
   /incident declare
   # Автоматически создаётся канал: #incident-YYYY-MM-DD-N
   ```

3. **Назначить Incident Commander (IC)**
   - Обычно: дежурный инженер.
   - При недоступности: следующий в ротации.

4. **Первичная оценка**
   - Что затронуто? (сервис, регион, пользователи?)
   - Сколько пользователей затронуто?
   - Есть ли риск потери данных?

### Фаза 2: Стабилизация (5–15 минут)

**IC распределяет роли**:

- **Responder** — устраняет проблему.
- **Communications** — обновляет статус-страницу и оповещает пользователей.
- **Doc Writer** — ведёт хронологию инцидента.

**Действия Responder** (используйте [Operations Runbook](./OPERATIONS_RUNBOOK.md)):

```bash
# 1. Проверить статус подов по namespace
kubectl get pods -n fitness-platform-production

# 2. Посмотреть последние deployment'ы
kubectl rollout history deployment/gateway -n fitness-platform-production

# 3. Быстрый перезапуск подозрительного пода (OOMKilled / CrashLoopBackOff)
kubectl delete pod <pod-name> -n fitness-platform-production

# 4. Если проблема после недавнего деплоя — откат
kubectl rollout undo deployment/gateway -n fitness-platform-production

# 5. Логи по сервисам (Kibana или kubectl)
kubectl logs -f deployment/gateway -n fitness-platform-production --tail=200
```

**Проверка новых сервисов**:

```bash
# Device Aggregator (OAuth, Fitbit/Garmin/Withings)
kubectl get pods -n fitness-platform-production -l app=device-aggregator
kubectl logs -f deployment/device-aggregator -n fitness-platform-production | grep -i "error\|panic"

# Проверить health endpoints
curl -k https://<gateway-host>:8443/health
curl http://device-aggregator:8083/health
```

**Действия Communications**:

- Обновить статус-страницу: [status.fitpulse.app](https://status.fitpulse.app)
- Оповестить в `#general-incidents`
- Подготовить заявление: «Идёт работа над восстановлением сервиса...»

### Фаза 3: Разрешение (15+ минут)

- **Внедрить fix**: код-патч, увеличить масштаб, откат и т.д.
- **Проверить fix**: запустить smoke-тесты, убедиться в норме метрик.
- **Мониторить регрессию**: следить за error rate и latency 15 минут.

### Фаза 4: Восстановление (пост-инцидент)

1. **Подтвердить нормальную работу**: сервис стабилен в течение 24 часов.
2. **Убрать labels инцидента** со статус-страницы.
3. **Коммуникация**: публичный post-mortem.
4. **Root Cause Analysis (RCA)**:
   - Задокументировать что произошло.
   - Объяснить почему это произошло.
   - Предложить превентивные меры.

5. **Провести разбор (debrief)**: в течение 48 часов после инцидента.

---

## SEV-2 Response: Деградация сервиса

### Хронология

- **0–15 мин**: Алерт → Подтверждение → Триаж.
- **15–60 мин**: Расследование → Митигация (масштабирование, оптимизация).
- **60+ мин**: Разрешение → Коммуникация.

### Пример: Высокий error rate

```bash
# 1. Посмотреть error rate по сервисам
curl "http://prometheus:9090/api/v1/query" \
  --data-urlencode 'query=rate(error_total[5m])' | jq .

# 2. Определить сервис с наибольшим числом ошибок
kubectl get pods -n fitness-platform-production -l app=biometric-service
kubectl logs -f deployment/biometric-service -n fitness-platform-production | grep -i "error"

# 3. Проверить соединения с БД
kubectl exec -it deploy/postgres -n fitness-platform-production -- \
  psql -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# 4. Масштабировать при необходимости
kubectl scale deployment biometric-service --replicas=5 -n fitness-platform-production

# 5. Мониторить восстановление
kubectl top pod -n fitness-platform-production --containers

# 6. Оповестить команду в #incidents
```

---

## Security Incident Response

### Утечка данных (SEV-1+)

1. **Немедленные действия** (0–1 час)
   - Ротация скомпрометированных учётных данных.
   - Отзыв токенов пользователей, если взломана auth-система.
   - Изоляция затронутого сервиса (Network Policy) при необходимости.
   - Сделать forensic-снапшоты: `kubectl cp`, `kubectl exec ... tar`.

2. **Расследование** (1–4 часа)
   - Просмотр audit-логов: Kibana запрос `level="AUDIT_*"`.
   - Проверка lateral movement: активны ли Network Policies?
   - Определение scope: какие данные были доступны?

3. **Уведомление** (4–24 часа)
   - Уведомить затронутых пользователей (email-шаблон у security-команды).
   - Подать инцидент-репорт в Legal.
   - Уведомить Роскомнадзор при необходимости (152-ФЗ).

4. **Ремедиация** (24+ часов)
   - Внедрить security-патч.
   - Перешифровать потенциально скомпрометированные данные.
   - Развернуть WAF-правила, если обнаружена атака.

### Уязвимость в коде (SEV-1, если критична)

1. **Немедленно**: исправить код, собрать новый контейнерный образ.
2. **Тестирование**: запустить security-сканы (Snyk, Trivy).
3. **Деплой**: использовать canary-деплой (Этап 7 пайплайна).
4. **Мониторинг**: следить за попытками эксплуатации в логах.

---

## Коммуникационные шаблоны

### Начальный статус

```text
ИНЦИДЕНТ: Деградация API сервиса
Начало: 2026-05-06T14:30Z
Влияние: ~10% пользователей испытывают таймауты
Статус: Идёт расследование
Обновления: https://status.fitpulse.app
```

### Разрешение

```text
РЕШЕНО: Деградация API сервиса
Длительность: 45 минут
Причина: Исчерпан пул соединений к БД из-за memory leak в biometric-service
Fix: Выпущен и развёрнут патч biometric-service v2.1.1
Мониторинг: Все метрики в норме, потеря данных отсутствует
```

### Post-Mortem

```text
Post-Mortem: Деградация API сервиса (2026-05-06)

Хронология:
- 14:30 UTC: Сработал алерт (error rate > 5%)
- 14:32 UTC: IC подключился, начато расследование
- 14:40 UTC: Определена root cause (memory leak)
- 14:50 UTC: Развёрнут патч v2.1.1
- 15:00 UTC: Сервис восстановлен

Root Cause: Memory leak в логике connection pooling

Action Items:
1. [DONE] Деплой фикса memory leak v2.1.1
2. [TODO] Добавить memory profiling в CI/CD
3. [TODO] Добавить алерт на память (> 80% utilisation)
4. [TODO] Пересмотреть конфигурацию connection pooling

Участники: Platform Team
Дата: 2026-05-08 10:00 UTC
```

---

## Инструменты и доступы

|Инструмент|URL / Путь|Назначение|
|---|---|---|
|PagerDuty|[fitpulse.pagerduty.com](https://fitpulse.pagerduty.com)|Трекинг инцидентов and on-call|
|Slack|`#incidents`, `#alerts`, `#general-incidents`|Коммуникация|
|Grafana|[grafana.fitpulse.app:3000](https://grafana.fitpulse.app:3000)|Дашборды and метрики|
|Kibana|[kibana.fitpulse.app:5601](https://kibana.fitpulse.app:5601)|Логи и анализ|
|Kubernetes|`kubectl`|Оркестрация контейнеров|
|Status Page|[status.fitpulse.app](https://status.fitpulse.app)|Публичный статус сервисов|

---

## Чек-лист инцидента

- [ ] Подтвердить алерт в PagerDuty/Slack
- [ ] Объявить инцидент, назначить IC
- [ ] Собрать первичную информацию (Что? Когда? Количество затронутых?)
- [ ] Назначить Responder, Communications, Doc Writer
- [ ] Внедрить fix (откат, масштабирование, патч и т.д.)
- [ ] Проверить разрешение (smoke-тесты, метрики)
- [ ] Обновить статус-страницу
- [ ] Запланировать RCA в течение 48 часов
- [ ] Документировать lessons learned

---

## Работа с новыми сервисами

При расследовании учитывать все активные компоненты проекта:

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

**Последнее обновление**: 2026-06-15  
**Ведёт**: Security & Platform Teams
