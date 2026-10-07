# Runbook: Quarterly Access Review (QUAR)

> **Назначение:** Данный runbook описывает процедуру ежеквартального пересмотра доступов к персональным данным (ПДн) и критической инфраструктуре FitPulse. Процедура является обязательной в соответствии с требованиями 152-ФЗ, GDPR и внутренней Политикой обработки ПДн.
>
> **Частота:** Ежеквартально (каждые 3 месяца)  
> **Ведёт:** Ответственный за обработку ПДн + CTO / Security Lead  
> **Длительность:** 1–2 дня  
> **Статус:** Обязательный

---

## 1. Подготовка (за 1 неделю до QUAR)

### 1.1 Сбор данных

- [ ] Получить список всех пользователей (сотрудников, администраторов) с доступом к:
  - Kubernetes cluster (`kubectl get rolebindings, clusterrolebindings -A`)
  - Базам данных (PostgreSQL roles: `\du`)
  - Vault (политики, токены)
  - GitHub репозиторию (collaborators, teams)
  - CI/CD pipeline (GitHub Actions secrets, deploy keys)
  - Мониторингу (Grafana, Alertmanager, Prometheus)
  - Valkey (ключи, конфигурация)
  - RabbitMQ (пользователи, очереди, permissions)
  - Elasticsearch/Kibana (если развёрнут; в Phase 1 audit-logger пишет в stdout, ES не развёрнут)
  - Telegram Bot API (токены, chat IDs, permissions)
- [ ] Получить audit-логи за квартал (доступы, изменения ролей, действия с ПДн)
- [ ] Подготовить шаблон отчёта: используйте структуру из раздела 4 ниже (шаблон `docs/compliance/QUARTERLY_ACCESS_REVIEW_REPORT_TEMPLATE.md` может быть создан при необходимости)

### 1.2 Уведомление

- [ ] Уведомить участников QUAR за 5 рабочих дней
- [ ] Назначить:
  - **QUAR Lead** (обычно CTO или Security Lead)
  - **Recorder** (ведёт протокол)
  - **Reviewers** (руководители команд: Platform, Security, Backend, Frontend)

---

## 2. Выполнение QUAR

### 2.1 Критерии доступа

Для каждого аккаунта проверяется:

| Критерий | Вопрос | Действие |
| --- | --- | --- |
| **Назначение** | Зачем нужен доступ? | Документировать назначение |
| **Least privilege** | Является ли доступ минимально необходимым? | Удалить лишние права |
| **Активность** | Используется ли доступ? | Отозвать неиспользуемые доступы |
| **MFA** | Включён ли MFA? | Обязать включить MFA |
| **Срок действия** | Есть ли срок действия доступа? | Установить срок для временных доступов |
| **NDA / onboarding** | Подписан ли NDA / пройден ли onboarding? | Запросить документы |

### 2.2 Проверка по компонентам

#### Kubernetes RBAC

```bash
# Список всех ServiceAccounts, Roles, RoleBindings
kubectl get sa,roles,rolebindings,clusterroles,clusterrolebindings -A -o wide

# Проверить, кто может читать секреты
kubectl get rolebindings -A -o json | jq -r '.items[] | select(.roleRef.name == "secret-reader") | .subjects[]?.name'

# Проверить, кто имеет доступ к data-zone (БД, Vault)
kubectl get networkpolicies -n data-zone -o wide
```

#### PostgreSQL

```bash
# Список ролей и их привилегий
psql -U postgres -c "\du"

# Проверить, кто может подключиться к production БД
psql -U postgres -c "SELECT rolname, rolcreaterole, rolcreatedb, rolsuper FROM pg_roles;"
```

#### Vault

```bash
# Список политик
vault policy list

# Список токенов (с истечением срока)
vault list auth/token/lookup-self

# Проверить, нет ли токенов без TTL
vault list auth/token/accessors
```

#### GitHub

```bash
# Список collaborators с правами admin/maintain
gh api repos/:owner/:repo/collaborators --jq '.[] | select(.permissions.admin == true or .permissions.maintain == true) | .login'

# Список команд и их участников
gh api orgs/:org/teams --jq '.[].slug'
```

#### CI/CD Secrets

```bash
# Список секретов в GitHub репозитории (только названия, без значений)
gh secret list

# Проверить, нет ли устаревших секретов (дата последнего обновления)
gh secret list --json name,createdAt,updatedAt
```

#### Мониторинг

```bash
# Grafana: список пользователей с ролью Admin
curl -s -H "Authorization: Bearer $GRAFANA_API_KEY" "$GRAFANA_URL/api/admin/users?perpage=100" | jq '.[] | select(.role == "Admin")'

# Alertmanager: список получателей алертов
kubectl get secret alertmanager-main -n monitoring -o jsonpath='{.data.alertmanager\.yaml}' | base64 -d
```

#### Valkey

```bash
# Список ключей и TTL
kubectl exec -it deployment/valkey -n fitness-platform-production -- redis-cli KEYS '*'

# Проверить, нет ли ключей без TTL (вечные сессии)
kubectl exec -it deployment/valkey -n fitness-platform-production -- redis-cli TTL <key>
```

#### RabbitMQ

```bash
# Список пользователей RabbitMQ
kubectl exec -it statefulset/rabbitmq -n fitness-platform-production -- rabbitmqctl list_users

# Список очередей и их владельцев
kubectl exec -it statefulset/rabbitmq -n fitness-platform-production -- rabbitmqctl list_queues name owner
```

#### Telegram Bot API

```bash
# Проверить токен бота
curl -s "https://api.telegram.org/bot$TELEGRAM_BOT_TOKEN/getMe" | jq '.result.username'

# Список chat IDs, которым бот отправляет алерты
grep -r "chat_id" configs/k8s/base/ | grep -v secret
```

#### Device Aggregator & Admin CLI

```bash
# Device Aggregator: проверить, что сервис работает и имеет корректные credentials
kubectl get pods -n fitness-platform-production -l app=device-aggregator
kubectl logs -f deployment=device-aggregator -n fitness-platform-production | grep -i "error\|webhook"

# Admin CLI: проверить, что бинарник имеет корректный config и доступ к Vault/PostgreSQL
admin-cli --check-config
admin-cli --verify-vault-connection
```

### 2.3 Процедура пересмотра

Для каждого аккаунта:

1. **Верификация назначения:** Подтвердить, что доступ необходим для выполнения job responsibilities.
2. **Проверка least privilege:** Убедиться, что доступ ограничен минимально необходимыми правами.
3. **Проверка MFA:** Все административные аккаунты должны иметь включённый MFA (TOTP).
4. **Анализ активности:** Проверить audit-логи — использовался ли доступ в течение квартала.
5. **Решение:**
   - **Оставить** — доступ необходим, права минимальны, MFA включён
   - **Уменьшить** — лишние права, требуется сужение scope
   - **Отозвать** — доступ не используется, сотрудник ушёл, проект завершён
   - **Установить срок** — временный доступ, требует автоматического отзыва

---

## 3. Действия по итогам QUAR

### 3.1 Немедленные действия (в течение 24 часов)

- [ ] Отозвать доступы для уволенных сотрудников / завершённых проектов
- [ ] Отозвать лишние права у активных аккаунтов
- [ ] Установить TTL для временных доступов
- [ ] Обязать включить MFA для всех административных аккаунтов

### 3.2 Краткосрочные действия (в течение 1 недели)

- [ ] Внедрить automated access review (cronjob / GitHub Action)
- [ ] Настроить алерты на неиспользуемые доступы
- [ ] Обновить runbook с изменениями

### 3.3 Долгосрочные действия (в течение 1 месяца)

- [ ] Внедрить Vault для динамической выдачи временных доступов
- [ ] Автоматизировать проверку MFA через GitHub API / K8s API
- [ ] Добавить QUAR checklist в CI/CD pipeline

---

## 4. Отчётность

### 4.1 Шаблон отчёта

Используйте шаблон: `docs/compliance/QUARTERLY_ACCESS_REVIEW_REPORT_TEMPLATE.md`

### 4.2 Содержание отчёта

- Общее количество проверенных аккаунтов
- Количество аккаунтов с отозванным доступом
- Количество аккаунтов с уменьшенными правами
- Количество аккаунтов без MFA
- Количество новых доступов, выданных за квартал
- Рекомендации по улучшению процесса
- Подписи QUAR Lead и CTO

### 4.3 Хранение отчётов

- Отчёты хранятся в `docs/compliance/quarterly-access-reviews/YYYY-QX-review.md`
- Отчёты хранятся в течение 3 лет (требование 152-ФЗ)
- Отчёты предоставляются по запросу Роскомнадзора

---

## 5. Плановые даты QUAR

| Квартал | Плановый период | Ответственный | Отчёт |
| --- | --- | --- | --- |
| Q1 | Январь – Март | CTO / Security Lead | `docs/compliance/quarterly-access-reviews/YYYY-Q1-review.md` |
| Q2 | Апрель – Июнь | CTO / Security Lead | `docs/compliance/quarterly-access-reviews/YYYY-Q2-review.md` |
| Q3 | Июль – Сентябрь | CTO / Security Lead | `docs/compliance/quarterly-access-reviews/YYYY-Q3-review.md` |
| Q4 | Октябрь – Декабрь | CTO / Security Lead | `docs/compliance/quarterly-access-reviews/YYYY-Q4-review.md` |

---

## 6. Ссылки

- [Политика обработки ПДн FitPulse](ПОЛИТИКА_ОБРАБОТКИ_ПДН.md)
- [Реестр обработки ПДн](РЕЕСТР_ОБРАБОТКИ_ПДН.md)
- [DPA с Yandex ID](DPA_YANDEX_ID.md)
- [Runbook: Ответ на инциденты](../runbooks/INCIDENT_RESPONSE.md)
- [ADR 0005: Безопасное развёртывание](../adr/0005-security-deployment.md)
- [152-ФЗ «О персональных данных»](https://www.consultant.ru/document/cons_doc_LAW_61801/)
- [GDPR Art. 28 — Processor](https://gdpr-info.eu/art-28-gdpr/)
