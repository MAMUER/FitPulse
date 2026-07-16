# ADR 0006: Безопасное развёртывание и управление обновлениями

## Контекст

Система обрабатывает чувствительные биометрические и персональные данные, что требует надёжных мер безопасности для сетевой сегментации, контроля доступа, шифрования и соответствия регуляторным требованиям, включая 152-ФЗ.

## Решение

Реализовать комплексные меры безопасности:

1. **Сетевая сегментация**: Kubernetes Network Policies, делящие кластер на зоны:
   - dmz: внешний трафик (Ingress, WAF)
   - app-zone: микросервисы приложения
   - data-zone: базы данных, кэш, очереди
   - monitoring-zone: ELK, Prometheus, Grafana

2. **RBAC и привилегии**: Kubernetes RBAC с отдельным ServiceAccount на сервис, принцип минимальных привилегий, отдельные аккаунты для CI/CD и runtime.

3. **Шифрование**:
   - At rest: pgsodium для PostgreSQL (детерминированное AEAD для PII, envelope encryption для токенов).
   - In transit: TLS 1.3, mTLS для gRPC, certificate pinning.

4. **Управление зависимостями**: Dependabot и Snyk для сканирования уязвимостей, политики remediaton CVE.

5. **Аудит администратора**: аудит критически важных действий через ModSecurity WAF-логи (SecAuditLog). Централизованное application-level audit trail с длительным retention запланировано на Phase 2. Текущие audit-логи соответствуют требованиям 152-ФЗ только на уровне WAF/Ingress.

6. **WAF**: Ingress NGINX Controller с ModSecurity + OWASP CRS v4. WAF развёрнут в кластере с `hostNetwork: true`, управляется через Kubernetes ConfigMap. Правила автоматически обновляются через CronJob.

7. **Ротация секретов**: статические учётные данные БД через Kubernetes Secrets. Динамическая ротация секретов (Vault, CSI driver) запланирована на Phase 2.

## Последствия

- обеспечивает защиту данных и регуляторное соответствие;
- уменьшает attack surface через сегментацию и шифрование;
- предоставляет аудит-трейл для инцидентов безопасности.

## Реализация

- настройка Network Policies в Kubernetes (`configs/k8s/base/security-zones.yaml`);
- создание RBAC-ролей и ServiceAccounts (`configs/k8s/base/rbac/rbac.yaml`);
- реализация шифрования на уровне хранилища (pgsodium) и транспорта (TLS 1.3);
- интеграция security-сканирования в CI/CD пайплайны (Trivy, govulncheck, gosec);
- развёртывание WAF (Ingress NGINX + ModSecurity + OWASP CRS v4 с audit-логами) в кластере.
- Automated CRS updates через Kubernetes CronJob.
- cert-manager для управления TLS-сертификатами (Let's Encrypt).
- Статические секреты БД через Kubernetes Secrets; динамическая ротация через Vault запланирована на Phase 2.

## Рассмотренные альтернативы

- Менее restrictive network policies: повышенный риск безопасности.
- Ручное управление секретами: больше рисков утечек.
- Отсутствие WAF: уязвимость к распространённым веб-атакам.
