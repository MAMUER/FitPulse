# Шаблон отчёта о Recovery Drill

## Метаданные

| Поле | Значение |
| --- | --- |
| **Дата drill** | YYYY-MM-DD |
| **Время начала** | HH:MM UTC |
| **Время окончания** | HH:MM UTC |
| **Длительность** | X часов Y минут |
| **Tester** | [Имя] |
| **Reviewer** | [Имя] |
| **Статус** | ✅ PASS / ❌ FAIL |

## Сценарий drill

### Подготовка

- [x] Уведомление команды отправлено
- [x] Проверен последний бэкап (< 24 часов)
- [x] Проверена доступность MinIO/S3
- [x] Подготовлено clone-окружение
- [x] Резервное копирование текущего состояния (перед drill)

### Выполнение

#### Шаг 1: Остановка PostgreSQL

```bash
kubectl scale deployment/postgres --replicas=0 -n fitness-platform-production
```

**Время остановки:** HH:MM:SS  
**Результат:** ✅ / ❌  
**Заметки:**

#### Шаг 2: Восстановление из бэкапа

```bash
bash scripts/restore-to-clone.sh s3://postgres-backups/backup-fitness-LATEST.dump.enc
```

**Время начала восстановления:** HH:MM:SS  
**Время завершения восстановления:** HH:MM:SS  
**RTO (Recovery Time Objective):** X минут  
**Результат:** ✅ / ❌  
**Заметки:**

#### Шаг 3: Проверка целостности данных

```bash
psql -h clone-postgres -U postgres -d fitness -c "SELECT COUNT(*) FROM users;"
```

**Количество пользователей:** X  
**Последняя запись в biometric_data:** YYYY-MM-DD  
**RPO (Recovery Point Objective):** X часов  
**Результат:** ✅ / ❌  
**Заметки:**

#### Шаг 4: Тестирование сервисов

```bash
# Проверить доступность API
curl -f https://fittpulse.ru/health

# Проверить работу auth
curl -f https://fittpulse.ru/api/v1/auth/login
```

**Результат:** ✅ / ❌  
**Заметки:**

## Chaos Tests

### PostgreSQL Failure

**Скрипт:** `scripts/chaos-test-postgres.sh`  
**Время восстановления:** X секунд  
**Результат:** ✅ PASS / ❌ FAIL  
**Заметки:**

### Valkey Failure

**Скрипт:** `scripts/chaos-test-valkey.sh`  
**Время восстановления:** X секунд  
**Результат:** ✅ PASS / ❌ FAIL  
**Заметки:**

### Vault Failure

**Скрипт:** `scripts/chaos-test-vault.sh`  
**Время восстановления:** X секунд  
**Результат:** ✅ PASS / ❌ FAIL  
**Заметки:**

## Критерии успеха

| Критерий | Цель | Фактический | Статус |
| --- | --- | --- | --- |
| RTO | < 1 часа | X часов Y минут | ✅ / ❌ |
| RPO | < 24 часа | X часов | ✅ / ❌ |
| Целостность данных | 100% | X% | ✅ / ❌ |
| Доступность сервисов | 100% | X% | ✅ / ❌ |

## Итоги

### Что прошло хорошо

1. [Пункт 1]
2. [Пункт 2]

### Что нужно улучшить

1. [Пункт 1] - Приоритет: P0/P1/P2
2. [Пункт 2] - Приоритет: P0/P1/P2

### Action Items

| #  | Действие    | Приоритет | Ответственный | Срок        |
|----|-------------|-----------|---------------|-------------|
| 1  | [Описание]  | P0        | [Имя]         | YYYY-MM-DD  |
| 2  | [Описание]  | P1        | [Имя]         | YYYY-MM-DD  |

## Подписи

**Tester:** _________________ Дата: _______  
**Reviewer:** _________________ Дата: _______

---

**Отчёт сохранён в:** `docs/compliance/recovery-drills/YYYY-MM-DD-recovery-drill.md`
