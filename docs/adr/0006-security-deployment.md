# ADR 0006: Security Deployment and Update Management

## Context

The system handles sensitive biometric and personal data, requiring robust security measures for network segmentation, access control, encryption, and compliance with regulations like 152-ФЗ.

## Decision

Implement comprehensive security measures:

1. **Network Segmentation**: Kubernetes Network Policies dividing into zones:
   - dmz: External traffic (Ingress, WAF)
   - app-zone: Application microservices
   - data-zone: Databases, cache, queues
   - monitoring-zone: ELK, Prometheus, Grafana

2. **RBAC and Privileges**: Kubernetes RBAC with ServiceAccount per service, principle of least privilege, separate accounts for CI/CD and runtime.

3. **Encryption**:
   - At rest: TDE for PostgreSQL, KMS for volumes, Vault for secrets
   - In transit: TLS 1.3, mTLS for gRPC, certificate pinning

4. **Dependency Management**: Dependabot and Snyk for vulnerability scanning, policies for CVE remediation.

5. **Admin Audit**: Audit logging for sensitive actions, 1-year retention for 152-ФЗ compliance.

6. **WAF**: Nginx + ModSecurity or managed WAF with SQL injection, XSS protection, rate limiting.

7. **Secrets Rotation**: 90-day automatic rotation via Vault, dynamic DB credentials.

## Consequences

- Ensures data protection and regulatory compliance.
- Reduces attack surface through segmentation and encryption.
- Provides audit trail for security incidents.

## Implementation

- Configure Network Policies in Kubernetes.
- Set up RBAC roles and ServiceAccounts.
- Implement encryption at storage and transport layers.
- Integrate security scanning in CI/CD pipelines.
- Deploy WAF and configure rules.

## Alternatives Considered

- Less restrictive network policies: Higher security risk.
- Manual secret management: More prone to breaches.
- No WAF: Vulnerable to common web attacks.