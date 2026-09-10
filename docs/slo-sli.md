# FitPulse — SLO/SLI Definition

## 1. Обозначения

| Термин | Определение |
| --- | --- |
| **SLI** | Service Level Indicator — метрика, показывающая уровень обслуживания |
| **SLO** | Service Level Objective — целевое значение SLI |
| **SLA** | Service Level Agreement — контракт с пользователем |
| **Error budget** | Допустимое количество ошибок = 1 − SLO |

## 2. Доступность (Availability)

### 2.1 SLI

Доля успешных HTTP/gRPC запросов за 28 дней.

```text
availability = successful_requests / total_requests
```

### 2.2 SLO

- **Production**: 99.9% availability
- **Staging**: 99.0% availability

### 2.3 Error budget

- **Production**: 0.1% ≈ 43.2 минуты даунтайма в месяц
- **Staging**: 1.0% ≈ 7.2 часа даунтайма в месяц

### 2.4 Измерение

- Prometheus: `rate(http_requests_total{status=~"2.."}[5m]) / rate(http_requests_total[5m])`
- Alert: `availability < 0.999` за 5 минут → SEV-1

## 3. Задержка (Latency)

### 3.1 SLI

p95 и p99 задержка HTTP-запросов.

### 3.2 SLO

| Endpoint | p95 SLO | p99 SLO |
| --- | --- | --- |
| `/api/v1/login` | < 1s | < 3s |
| `/api/v1/register` | < 1s | < 3s |
| `/api/v1/profile` | < 500ms | < 1.5s |
| `/api/v1/biometrics` | < 1s | < 3s |
| `/api/v1/training/plans` | < 1s | < 3s |
| `/api/v1/ml/classify` | < 3s | < 8s |
| `/api/v1/ml/generate-plan` | < 5s | < 15s |
| `/health` | < 100ms | < 200ms |

### 3.3 Измерение

- Prometheus: `histogram_quantile(0.95, sum(rate(request_duration_seconds_bucket[5m])) by (le, endpoint))`
- Alert: `p95 > slo` за 10 минут → SEV-2

## 4. Частота ошибок (Error Rate)

### 4.1 SLI

Доля запросов с ошибкой (5xx + 4xx для аутентифицированных запросов).

### 4.2 SLO

- **Production**: < 0.1% ошибок
- **Staging**: < 1% ошибок

### 4.3 Измерение

- Prometheus: `rate(error_total[5m]) / rate(request_total[5m])`
- Alert: `error_rate > 0.001` за 5 минут → SEV-1

## 5. Пропускная способность (Throughput)

### 5.1 SLI

Количество успешных запросов в секунду (RPS).

### 5.2 SLO

- **Production**: поддерживать 100 RPS с headroom 2× (т.е. до 200 RPS без деградации)
- **Staging**: поддерживать 20 RPS

### 5.3 Измерение

- Prometheus: `rate(http_requests_total[1m])`
- k6: пиковая нагрузка 200 RPS, p95 < 3s

## 6. Свежесть данных (Data Freshness)

### 6.1 SLI

Время между последним обновлением данных пользователя и текущим временем.

### 6.2 SLO

- Биометрические данные: < 5 минут
- Тренировочные планы: < 1 час

### 6.3 Измерение

- Prometheus gauge: `biometric_sync_lag_seconds`
- Alert: `biometric_sync_lag_seconds > 300` за 10 минут → SEV-3

## 7. Длительность бэкапов (Backup Freshness)

### 7.1 SLI

Время с момента последнего успешного бэкапа.

### 7.2 SLO

- Полный бэкап (pg_dump): < 24 часов
- WAL архив: непрерывный

### 7.3 Измерение

- CronJob prometheus metric: `backup_timestamp_seconds`
- Alert: `time() - backup_timestamp_seconds > 86400` → SEV-2

## 8. Восстановление (Recovery)

### 8.1 SLO

- MTTR (Mean Time To Recovery): < 5 минут
- RPO (Recovery Point Objective): < 5 минут (с WAL archiving)
- RTO (Recovery Time Objective): < 15 минут

### 8.2 Измерение

- Grafana dashboard: `mttr_seconds`
- Плановые учения по восстановлению: ежемесячно

## 9. Бюджет ошибок (Error Budget Policy)

| Остаток бюджета | Действие |
| --- | --- |
| > 50% | Разрешаются обычные релизы |
| 20–50% | Требуется дополнительное тестирование, approval Tech Lead |
| < 20% | Только критические фиксы (security, data loss) |
| < 5% | Разрешён freeze на новые фичи |

### 9.1 Расчёт

```text
error_budget_remaining = (current_availability − 1 + slo_target) / slo_target
```

## 10. Дашборды и алерты

### 10.1 Grafana дашборды

- **SLO Overview**: availability, latency, error rate, error budget remaining
- **Service Health**: per-service RED metrics
- **Infrastructure**: CPU, memory, disk, network

### 10.2 Prometheus alerts

- `SLOServiceDown` — `up{job=~'fitness-.*'} == 0` за 2 мин
- `SLOHighLatency` — p95 > SLO за 10 мин
- `SLOHighErrorRate` — error_rate > SLO за 5 мин
- `SLOErrorBudgetBurnRate` — burn rate > 14.4× (потратим бюджет за 1 день вместо 28)

## 11. Ссылки

- Google SRE Book: <https://sre.google/sre-book/service-level-objectives/>
- CNCF Observability Whitepaper
