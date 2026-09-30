# SLO Monthly Error Budget Report

## Report Metadata

| Field | Value |
| --- | --- |
| **Month** | YYYY-MM |
| **Generated** | YYYY-MM-DD HH:MM UTC |
| **SLO Target** | 99.9% availability |
| **Error Budget** | 0.1% per month |

## Error Budget Consumption

| Service | Error Budget Remaining | Burn Rate | Status |
| --- | --- | --- | --- |
| gateway | X% | X× | ✅/❌ |
| user-service | X% | X× | ✅/❌ |
| biometric-service | X% | X× | ✅/❌ |
| training-service | X% | X× | ✅/❌ |
| classifier | X% | X× | ✅/❌ |
| ml-generator | X% | X× | ✅/❌ |
| data-processor | X% | X× | ✅/❌ |
| device-aggregator | X% | X× | ✅/❌ |
| security-events (audit log delivery) | X% | X× | ✅/❌ |
| vault (secrets availability) | X% | X× | ✅/❌ |

## Top-5 Error Sources

| Rank | Service | Error Type | Count | % of Total |
| --- | --- | --- | --- | --- |
| 1 | gateway | Rate limit exceeded / Valkey connection pool exhausted | X | X% |
| 2 | biometric-service | pgsodium/aegis256 decryption error / invalid ciphertext | X | X% |
| 3 | user-service | Argon2id password verification timeout / audit log write failure | X | X% |
| 4 | data-processor | RabbitMQ message processing failure / DLQ overflow | X | X% |
| 5 | classifier | ML inference timeout / model loading failure | X | X% |

## Latency Analysis

| Service | P50 | P95 | P99 | Target | Status |
| --- | --- | --- | --- | --- | --- |
| gateway | | | | < 2s | ✅/❌ |
| user-service | | | | < 2s | ✅/❌ |
| biometric-service | | | | < 2s | ✅/❌ |
| training-service | | | | < 2s | ✅/❌ |
| classifier | | | | < 500ms | ✅/❌ |
| ml-generator | | | | < 1s | ✅/❌ |
| data-processor | | | | < 2s | ✅/❌ |
| device-aggregator | | | | < 2s | ✅/❌ |
| security-events (audit log ingestion) | | | | < 5s | ✅/❌ |
| vault (secret read latency) | | | | < 200ms | ✅/❌ |

## Action Items

| # | Action | Priority | Owner | Due Date |
| --- | --- | --- | --- | --- |
| 1 | | P0/P1/P2 | | |
| 2 | | P0/P1/P2 | | |
| 3 | | P0/P1/P2 | | |

## Security & Reliability SLO

| Category | SLO | Target | Status |
| --- | --- | --- | --- |
| Security events | Время обнаружения инцидента (MTTD) | < 5 мин | ✅/❌ |
| Security events | Время реагирования (MTTR) для SEV-1 | < 1 час | ✅/❌ |
| Security events | Audit log delivery lag (event → ELK) | < 5 сек | ✅/❌ |
| ML inference | Classifier P99 latency | < 500ms | ✅/❌ |
| ML inference | ML generator P99 latency | < 1s | ✅/❌ |
| Vault | Secret read availability | 99.9% | ✅/❌ |
| Vault | Secret read P99 latency | < 200ms | ✅/❌ |
| Vault | Vault unseal time (after restart) | < 30 сек | ✅/❌ |

## Sign-offs

- [ ] Platform Team
- [ ] Security/Compliance
- [ ] Engineering Lead

---

**Report saved to:** `docs/compliance/slo-reports/YYYY-MM-error-budget-report.md`
