# FitPulse — Интеллектуальная платформа персонализированных тренировок

[![Build Status](https://github.com/MAMUER/fitpulse/actions/workflows/ci.yml/badge.svg)](https://github.com/MAMUER/fitpulse/actions)
[![Docker Pulls](https://img.shields.io/docker/pulls/fitpulse/gateway.svg)](https://hub.docker.com/u/fitpulse)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.28+-326CE5.svg)](https://kubernetes.io/)
[![Security](https://img.shields.io/badge/Security-Hardened-green.svg)](docs/SECURITY.md)
[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8.svg)](https://go.dev/)
[![Python Version](https://img.shields.io/badge/Python-3.12+-3776AB.svg)](https://www.python.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Coverage](https://codecov.io/gh/MAMUER/fitpulse/branch/main/graph/badge.svg)](https://codecov.io/gh/MAMUER/fitpulse)

**FitPulse** — микросервисная платформа для персонализированных тренировок, ML-анализа биометрии и интеграции с носимыми устройствами (Apple Watch, Samsung, Huawei, Amazfit).

---

## Документация

Полная документация разделена по назначению:

| Документ | Описание |
| ---------- | ---------- |
| [Техническое задание](docs/TECHNICAL_SPECIFICATION.md) | Полное ТЗ с требованиями, стадиями разработки и критериями приемки |
| [Архитектура](docs/ARCHITECTURE.md) | Инфраструктура, наблюдаемость, безопасность, релизный процесс |
| [API Reference](docs/API.md) | Полная спецификация REST/gRPC endpoints |
| [Security Policy](docs/SECURITY.md) | Меры безопасности, compliance, аудит |
| [Architecture Decision Records](docs/adr/) | Обоснование архитектурных решений |
| [UI Specification](docs/UI_SPECIFICATION.md) | Спецификация мобильного веб-интерфейса |
| [Runbooks](docs/runbooks/) | Операционные инструкции и response playbooks |
| [Contributing Guide](CONTRIBUTING.md) | Как внести вклад, стандарты кода, тестирование |

---

## Возможности платформы

**Для пользователей:**

- Персонализированные тренировочные планы (GAN)
- Автоматическая генерация диеты
- Интеграция с носимыми устройствами
- ML-классификация состояния (6 классов)
- Мониторинг биометрии в реальном времени

**Для администраторов:**

- Регистрация через invite-коды
- Управление пользователями
- Мониторинг и аудит

**Подробные таблицы**: [Возможности](docs/FEATURES.md) • [API Endpoints](docs/API.md) • [ML/GAN логика](docs/ML_SPECIFICATION.md)

---

## Безопасность

FitPulse реализует комплексные меры безопасности:

- JWT (ES256) + Refresh Token rotation
- HMAC-SHA256 подпись критических ответов
- Argon2id хеширование паролей
- Content Security Policy (nonce-based)
- Rate limiting (sliding window)
- Сетевая сегментация (Kubernetes Network Policies)
- mTLS для внутренних коммуникаций
- Соответствие 152-ФЗ

**Полный список мер**: [Security Policy](SECURITY.md) • [ADR-0006](docs/adr/0006-security-deployment.md)

---

## API Endpoints

### Публичные (без auth)

| Метод | Путь | Описание |
| ------- | ------ | ---------- |
| POST | `/api/v1/register` | Регистрация пользователя |
| POST | `/api/v1/register/invite` | Регистрация через invite-код |
| POST | `/api/v1/invite/validate` | Проверка invite-кода |
| POST | `/api/v1/login` | Вход |
| POST | `/api/v1/auth/confirm` | Подтверждение email |
| POST | `/api/v1/devices/register` | Регистрация устройства |
| POST | `/api/v1/devices/{id}/ingest` | Приём данных с устройства |
| GET | `/health` | Health check |

### Защищённые (JWT required)

| Метод | Путь | Описание |
| ------- | ------ | ---------- |
| POST | `/logout` | Выход с инвалидацией сессии |
| GET | `/profile` | Получить профиль |
| PUT | `/profile` | Обновить профиль |
| DELETE | `/profile` | Удалить профиль |
| POST | `/biometrics` | Добавить биометрию |
| GET | `/biometrics` | Получить биометрию |
| POST | `/training/generate` | Сгенерировать план |
| GET | `/training/plans` | Список планов |
| POST | `/training/complete` | Завершить тренировку |
| GET | `/training/progress` | Прогресс |
| POST | `/ml/classify` | Классификация состояния |
| POST | `/ml/generate-plan` | Генерация плана (GAN) |

### Админ (JWT + role=admin)

| Метод | Путь | Описание |
| ------- | ------ | ---------- |
| GET | `/admin/users` | Список пользователей |

**Полная спецификация**: [docs/API.md](docs/API.md)

---

## Быстрый старт (Quick Start)

```bash
# 1. Клонировать репозиторий
git clone https://github.com/MAMUER/fitpulse.git && cd fitpulse

# 2. Создать namespace и применить манифесты
kubectl create namespace fitness-platform
kubectl apply -k configs/k8s/base/ -n fitness-platform

# 3. Применить миграции БД
kubectl apply -f configs/k8s/base/jobs/init-db.yaml -n fitness-platform

# 4. Проверить статус подов
kubectl get pods -n fitness-platform

# 5. Health check
kubectl port-forward svc/gateway 8443:8443 -n fitness-platform
curl -k https://localhost:8443/health
```

**Требования**: Kubernetes 1.28+, 4+ ядер CPU, 8+ ГБ RAM, 40+ ГБ SSD.

**Подробная инструкция**: [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)

---

## Как внести вклад

См. [CONTRIBUTING.md](CONTRIBUTING.md) — ветвление, код-стайл, тесты, PR процесс.

---

## Лицензия

MIT
