# ADR 0004: Environment Matrix Configuration for Dev/Test/Staging/Prod

## Context

The project needs consistent configuration across multiple environments to ensure proper scaling, security, and monitoring as the system moves from development to production. Each environment has different requirements for resources, redundancy, and access controls.

## Decision

Implement a matrix of configurations for all components across environments:

- **Dev**: Minimal setup for development, local access, no backups.
- **Test**: Automated testing environment with basic redundancy and monitoring.
- **Staging**: Pre-production environment with full monitoring and security.
- **Prod**: Production environment with high availability, security, and compliance.

Key parameters include:
- K8s pods per service (1 → 2 → 3 → 5+ with HPA)
- PostgreSQL topology (single instance → primary+replica → primary+2 replicas → primary+3 replicas with sync/async)
- Redis topology (single node → Sentinel → Cluster mode)
- GPU resources (CPU only → T4 → A10 for ML inference)
- Monitoring stack (basic → full ELK+Prometheus → with alerts → on-call rotation)
- Backup strategy (none → daily dumps → WAL archiving → PITR)
- SSL/TLS (self-signed → Let's Encrypt → Corporate CA → CA + HSM)
- Access control (local → VPN → VPN+2FA → 2FA + IP whitelist + hardware token)

## Consequences

- Ensures consistent scaling and security practices across environments.
- Facilitates smooth transitions from dev to prod.
- Provides clear guidelines for infrastructure provisioning.

## Implementation

- Document the matrix in architecture documentation.
- Implement environment-specific configurations in Kubernetes manifests and deployment scripts.
- Use Helm charts or Kustomize for environment-specific overlays.

## Alternatives Considered

- Single configuration with overrides: Less clear separation between environments.
- Manual configuration per environment: Error-prone and inconsistent.