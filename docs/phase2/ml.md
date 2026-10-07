# ML

## 14. Ежедневная адаптивная модификация плана

### 14.1 Текущий статус Phase 1

- **Classifier**: Rule-based (не ML), детектирует `recovery`, `endurance_basic`, `endurance_threshold`, `power_hiit`, `overtraining`, `illness`, `unknown`
- **ML Generator**: Conditional Diffusion Model (DDPM) с 3-tier fallback, inference-ready в production
- **DVC**: Инициализирован (`dvc.yaml`, `params.yaml`)
- **Ограничение**: Текущий сервер (2 vCPU / 4 ГБ RAM / 60 ГБ Storage) не позволяет запускать ежедневное ML-переобучение без влияния на отзывчивость приложения

### 14.2 Acceptance Criteria Phase 2

- Планы пользователей автоматически пересматриваются раз в 24 часа на основе последних биометрических данных.
- Переобучение не влияет на p95 латентности API (< 2s).
- DVC-tracked модели версионируются и откатываются при деградации качества.
- План переобучения завершается за < 10 минут на выделенном воркере (2+ vCPU, 4+ ГБ RAM).

---

## 15. Model Versioning & Production Update

### 15.1 Контекст

Phase 1 развёрнута статическая ONNX-модель. Phase 2 требует версионирования и безопасного rollout моделей.

### 15.2 Acceptance Criteria

- Каждая модель имеет DVC-хеш и semantic version
- Rollback модели при деградации качества < 5 минут
- Canary rollout: 10% трафика → 50% → 100% при успешных метриках
- Fallback на rule-based планы при недоступности ML-воркера

---

## 16. Оценка стоимости (руб/мес)

### 16.1 Контекст

Все оценки указаны для российского хостинга (Yandex Cloud / Selectel / Timeweb VPS) и approximated. Точные цифры зависят от провайдера и региона.

### 16.2 Инфраструктура

| Компонент | Текущая стоимость | Новая стоимость | Δ |
| --- | --- | --- | --- |
| VPS (2 vCPU / 4 ГБ / 60 ГБ) | ~2 500 ₽/мес | ~3 500 ₽/мес (4 vCPU / 8 ГБ / 80 ГБ SSD) | +1 000 ₽/мес |
| Managed PostgreSQL (Yandex Managed) | — | ~2 500–4 000 ₽/мес | +2 500–4 000 ₽/мес |
| Vault (self-hosted на отдельном VPS) | — | ~1 500 ₽/мес (2 vCPU / 4 ГБ) | +1 500 ₽/мес |
| Backup storage (S3-compatible, 100 ГБ) | — | ~300 ₽/мес | +300 ₽/мес |
| Domain fittpulse.duckdns.org (первый год) | — | ~1 500 ₽/год | +200 ₽/мес |
| SSL-сертификат (Let's Encrypt) | — | 0 ₽/мес | 0 ₽/мес |
| **Итого инфраструктура** | **~1 500 ₽/мес** | **~8 000–10 500 ₽/мес** | **+6 500–9 000 ₽/мес** |

### 16.3 ML-сервисы

| Компонент | Стоимость | Примечание |
| --- | --- | --- |
| MLflow (self-hosted) | 0 ₽/мес | Запускается на существующем VPS |
| DVC remote storage | 0 ₽/мес | Локальный диск / S3-compatible |
| GPU-воркер (опционально, для ускорения переобучения) | ~5 000–15 000 ₽/мес | Yandex Cloud GPU / Lambda Labs; не требуется для inference |
| **Итого ML** | **0–15 000 ₽/мес** | Зависит от необходимости GPU для переобучения |

### 16.4 Security / Compliance

| Компонент | Стоимость | Примечание |
| --- | --- | --- |
| Corp email (Yandex 360 / Google Workspace) | ~300–600 ₽/мес за пользователя | 1–2 пользователя |
| PGP ключ / WKD | 0 ₽/мес | Self-hosted |
| Bug Bounty вознаграждения (опционально) | 0–10 000 ₽/мес | Зависит от бюджета |
| **Итого Security** | **300–10 600 ₽/мес** | — |

### 16.5 Итого Phase 2

| Сценарий | Стоимость/мес | Годовая стоимость |
| --- | --- | --- |
| Минимум (без GPU, без bug bounty) | ~8 500 ₽/мес | ~102 000 ₽/год |
| Рекомендуемый (с observability, без GPU) | ~12 000 ₽/мес | ~144 000 ₽/год |
| Максимальный (с GPU, bug bounty) | ~25 000–35 000 ₽/мес | ~300 000–420 000 ₽/год |

**Trade-off**: На текущем 2-vCPU / 4 ГБ сервере невозможно запустить Vault + Istio + PostgreSQL HA одновременно. Требуется апгрейд VPS до минимум 4 vCPU / 16 ГБ RAM или разделение на 2 VPS.

---

## 17. Resource Plan: FTE

### 17.1 Контекст

Phase 2 требует специализации, которой нет у единственного разработчика. Ниже — оценка человеко-часов и необходимых ролей.

### 17.2 Роли и ответственность

| Роль | Занятость | Ответственность |
| --- | --- | --- |
| **DevOps/Platform** | 0.8 FTE | VPS provisioning, k8s, Vault, PostgreSQL HA, Service Mesh, CI/CD |
| **Backend (Go)** | 0.6 FTE | Secrets integration, mTLS migration, admin panel, compliance endpoints |
| **ML/Data Engineer** | 0.4 FTE | DVC pipeline, adaptive retrain, model versioning |
| **Frontend** | 0.3 FTE | Achievements, Diet, Devices views из UI_SPECIFICATION |
| **Legal/Compliance** | 0.2 FTE | 152-ФЗ, медицинская регистрация, политики |
| **Security** | 0.2 FTE | Bug bounty, PGP, WAF rules, penetration testing |
| **Product/Design** | 0.1 FTE | Приоритизация фич, UI/UX approval |

### 17.3 Общие затраты

| Сценарий | FTE | Срок | Человеко-часы |
| --- | --- | --- | --- |
| Агрессивный (все параллельно) | 1.6 FTE | 8 недель | ~2 560 ч |
| Рекомендуемый (последовательный) | 0.8 FTE | 16 недель | ~2 560 ч |
| Консервативный (1 человек, 0.5 FTE) | 0.5 FTE | 32 недели | ~2 560 ч |

**Важно**: В текущем состоянии проект поддерживается 1 человеком (`@MAMUER`). Phase 2 **невозможна** без привлечения хотя бы одного дополнительного DevOps/Backend разработчика.
