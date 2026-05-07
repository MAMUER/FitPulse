# ADR 0002: Canary Deployment, Encrypted Backups, and Observability

## Context

The project must support canary deployment and rollback, secure encrypted backups, and clear operational guidance for observability.

## Decision

- add a dedicated canary gateway deployment and service
- add a canary ingress resource with NGINX ingress annotations for traffic splitting
- provide encrypted backup and restore scripts for PostgreSQL dumps
- document the strategy for production-grade rollout and restore testing

## Consequences

- production rollout becomes incremental and safer
- backup data is preserved encrypted with AES-256
- restore can be tested regularly using the provided scripts
- the architecture is aligned with release gates and monitoring strategy

## Implementation

- added `configs/k8s/deployments/gateway-canary.yaml`
- added `configs/k8s/services/gateway-canary-service.yaml`
- added `configs/k8s/ingress-canary.yaml`
- added `scripts/backup-db.sh` and `scripts/restore-db.sh`

