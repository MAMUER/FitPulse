# FitPulse v11 — Бэкенд-хендофт и актуальное состояние

## Актуальное состояние

- Фронтенд: React + Vite в `web/`, 13 экранов, маршрутизация в `web/src/App.jsx`, состояние в `web/src/contexts/AppContext.jsx`.
- Бэкенд: Gateway + сервисы на Go/Python, API эндпоинты описаны в `cmd/gateway/main.go`.
- БД: миграции в `db/migrations/`, схема покрывает auth, profiles, biometrics, training, health, devices, invites.
- Инфраструктура: docker-compose, Makefile, GitHub Actions, Terraform, K8s манифесты.

## Что уже реализовано на бэкенде

### 1. Аутентификация

- POST `/api/v1/register`
- POST `/api/v1/register/invite`
- POST `/api/v1/invite/validate`
- POST `/api/v1/login`
- POST `/api/v1/auth/confirm`
- GET `/api/v1/auth/verify-status`
- GET `/api/v1/auth/google`
- GET `/api/v1/auth/google/callback`
- POST `/api/v1/auth/refresh`
- POST `/api/v1/auth/2fa/verify`
- POST `/api/v1/auth/2fa/setup`
- POST `/api/v1/auth/2fa/confirm`
- GET `/api/v1/auth/2fa/status`
- POST `/api/v1/auth/2fa/disable`
- POST `/api/v1/auth/critical-session`
- POST `/api/v1/logout`

### 2. Профиль пользователя

- GET/PUT/DELETE `/api/v1/profile`
- GET/POST/DELETE `/api/v1/health/conditions`
- GET/POST `/api/v1/health/body-composition`
- GET/POST/PUT/DELETE `/api/v1/health/menstrual-cycles`

### 3. Биометрия и устройства

- POST `/api/v1/biometrics`
- GET `/api/v1/biometrics`
- GET `/api/v1/integrations/providers`
- POST `/api/v1/integrations/{source}/disconnect`
- Webhook: `/api/integrations/open-wearables/webhook`

### 4. Тренировки

- GET `/api/v1/training/plans`
- POST `/api/v1/training/generate`
- GET `/api/v1/training/plans/{plan_id}`
- GET `/api/v1/training/progress`
- POST `/api/v1/training/complete`
- GET `/api/v1/achievements`

### 5. ML

- POST `/api/v1/ml/chat` — классификация состояния + генерация плана тренировок + генерация диеты

### 6. Админка

- GET `/api/v1/admin/users`
- GET/POST `/api/v1/admin/invites`
- POST `/api/v1/admin/invites/{code}/revoke`

## Требуемые доработки

### Бэкенд

- [ ] Выпустить стабильные JSON-ошибки по формату:

  ```json
  {
    "error": {
      "code": "VALIDATION_ERROR",
      "message": "Human-readable message",
      "details": {}
    }
  }
  ```

- [ ] Реaltime-транспорт для чата: WebSocket/SSE.
- [ ] Прокси/хостинг для видеоконтента, если выходим за демо-данные.
- [ ] Картографический провайдер и проксирование геокодеров.
- [ ] Production секреты: вынести из `external_secrets` в реальный External Secrets Operator / vault.

### Фронтенд

- [ ] Подключить React-экранны к API через `web/src/utils/apiClient.js` и сервисы в `web/src/utils/api/`.
- [ ] hydrateProfile на старте, 401→refresh→logout при провале.
- [ ] Экраны: Splash → Survey → Verify → ChatDetail.

### Инфраструктура

- [ ] Проверить актуальность `terraform/`, `configs/k8s/`, `.github/workflows/ci.yml`, `docker-compose.yml`.
- [ ] Проверить Dependabot, Alertmanager, мониторинг.

## Критерии приема релиза

1. Все демо-fallback'и заменены реальными API-ответами.
2. Жизненный цикл auth/session работает через refresh/reopen.
3. Удаление конфиденциальных данных/аккаунта выполняется на стороне сервера.
4. AI/чат/календарь/питание/тренировочные данные сохраняются в рамках аккаунта.
5. Production-медиа/карты работают без разглашения приватных credentials.
6. Полный набор регрессионных тестов проходит с нулевым количеством критических ошибок.
