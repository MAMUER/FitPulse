# ADR 0025: Выбор provisioner для локального хранилища k3s

## Статус

Принято

## Контекст

k3s по умолчанию использует local-path-provisioner для динамического Provisioning PersistentVolumeClaim. На VPS с одним диском требуется простое решение, которое не зависит от внешних хранилищ (NFS, Ceph, etc.).

Требования:

- автоматическое создание PV на локальном диске;
- поддержка ReadWriteOnce;
- совместимость с StatefulSet (PostgreSQL);
- простота развёртывания.

## Решение

Использовать local-path-provisioner как provisioner по умолчанию:

- **local-path-provisioner**: легковесный provisioner, создающий PV в `/opt/local-path-provisioner`.
- **HostPath**: для single-node кластера достаточно; при росте до multi-node потребуется замена на распределённое хранилище.
- **ResourceQuota + LimitRange**: ограничение размера PVC в namespace `database`.

## Последствия

- **Плюсы**: нет внешних зависимостей, простота отладки.
- **Плюсы**: данные остаются на узле, что ускоряет I/O.
- **Нейтрально**: local-path не обеспечивает replication; при отказе узда данные теряются.
- **Риски**: при миграции на multi-node потребуется миграция данных PostgreSQL.

## Рассмотренные альтернативы

- **NFS-provisioner**: требует отдельный NFS-сервер, избыточно для single-node VPS.
- **Rook/Ceph**: слишком тяжёлый для 2 vCPU / 4 ГБ RAM.
- **Longhorn**: требует минимум 3 узда для replication, не подходит для текущего железа.

## Реализация

- `configs/k8s/base/local-path-provisioner.yaml` — манифест provisioner.
- `configs/k8s/base/local-path-provisioner/resource-quota.yaml` — ограничения для database namespace.
- `terraform/modules/k3s/main.tf` — настройка k3s с local-path.
