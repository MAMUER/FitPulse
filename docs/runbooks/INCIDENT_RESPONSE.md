# Incident Response Playbook

## Overview

This document defines the process for responding to security incidents, service outages, and data breaches.

---

## Incident Classification

### Severity Levels

| Level | Impact | Resolution Time | Escalation |
|-------|--------|-----------------|------------|
| **SEV-1** | Service completely down, data loss risk | 15 min | Immediate PagerDuty → Tech Lead → CTO |
| **SEV-2** | Service degraded, partial functionality loss | 1 hour | Slack → On-Call Engineer → Tech Lead |
| **SEV-3** | Minor issues, user experience impact | 4 hours | Slack → On-Call Engineer |
| **SEV-4** | No immediate user impact | 24 hours | Ticket queue |

---

## SEV-1 Response: Critical Incident

### Phase 1: Triage (0-5 minutes)

1. **Alert Acknowledgment**
   - PagerDuty: Click "Acknowledge" immediately
   - Slack: React with `:ack:` to #alerts channel

2. **Declare Incident**
   ```bash
   # Start incident in Slack
   /incident declare
   # Auto-creates channel: #incident-2026-05-06-1
   ```

3. **Identify Incident Commander (IC)**
   - Usually: On-call engineer
   - If unavailable: Next in rotation

4. **Initial Assessment**
   - What is affected? (Service, region, users?)
   - How many users impacted?
   - Is there data loss?

### Phase 2: Stabilization (5-15 minutes)

**IC Coordinates**:
- Assign **Responder** (fix the issue)
- Assign **Communications** (update customers)
- Assign **Doc Writer** (keep timeline updated)

**Responder Actions** (use [Operations Runbook](./OPERATIONS_RUNBOOK.md)):
- Check service health: `kubectl get pods -n fitness-platform`
- Review recent deployments: `kubectl rollout history deployment/gateway`
- Check logs: Kibana, `kubectl logs`
- Consider rollback if recent deployment

**Communications**:
- Update status page: [status.fitpulse.app](https://status.fitpulse.app)
- Notify customers in #general-incidents
- Prepare statement: "We are investigating a service issue..."

### Phase 3: Resolution (15+ minutes)

- **Implement fix** (code patch, scale-up, rollback, etc.)
- **Verify fix**: Run smoke tests, check metrics
- **Monitor for regression**: Watch error rate, latency for 15 minutes

### Phase 4: Recovery (Post-Incident)

1. **Restore Normal Operations**
   - Confirm service stable for 24 hours
   - Remove incident label from status page

2. **Communication**
   - Public post-mortem: [status.fitpulse.app](https://status.fitpulse.app)
   - Team debrief: Schedule within 48 hours

3. **Root Cause Analysis (RCA)**
   - Document what happened
   - Identify why it happened
   - Propose preventive measures

---

## SEV-2 Response: Service Degradation

### Timeline

- **0-15 min**: Alert → Acknowledge → Triage
- **15-60 min**: Investigation → Mitigation (scale-up, optimization)
- **60+ min**: Resolution → Communication

### Example: High Error Rate

```bash
# 1. Check error rates by endpoint
curl "prometheus:9090/api/v1/query" \
  --data-urlencode 'query=rate(error_total[5m])' | jq .

# 2. Identify service with highest errors
kubectl get pods -n fitness-platform -l app=biometric-service
kubectl logs -f deployment/biometric-service -n fitness-platform | grep ERROR

# 3. Check database connections
psql -h postgres -U postgres -c "SELECT count(*) FROM pg_stat_activity;"

# 4. Scale up if needed
kubectl scale deployment biometric-service --replicas=5 -n fitness-platform

# 5. Monitor recovery
kubectl top pod -n fitness-platform --containers

# 6. Alert team in #incidents channel
```

---

## Security Incident Response

### Data Breach (SEV-1+)

1. **Immediate Actions** (0-1 hour)
   - Rotate compromised credentials immediately
   - Revoke user tokens if authentication compromised
   - Isolate affected service from network if needed
   - Take forensic snapshots: `kubectl cp`, `kubectl exec ... tar`

2. **Investigation** (1-4 hours)
   - Review audit logs: Kibana query `level="AUDIT_*"`
   - Check for lateral movement: Network policies in effect?
   - Identify scope: Which data was accessed?

3. **Notification** (4-24 hours)
   - Notify affected users (email template in security team)
   - File incident report with legal
   - Notify Roskomnadzor if required (152-ФЗ)

4. **Remediation** (24+ hours)
   - Implement security patch
   - Re-encrypt potentially exposed data
   - Deploy WAF rules if attack detected

### Code Vulnerability (SEV-1 if critical)

1. **Immediate**: Patch code, build new container image
2. **Test**: Run security scans (Snyk, trivy)
3. **Deploy**: Use canary deployment (Stage 7 of pipeline)
4. **Monitor**: Watch for exploitation attempts in logs

---

## Communication Templates

### Initial Status

```
INCIDENT: Service API Degradation
Started: 2026-05-06T14:30Z
Impact: ~10% of users experiencing timeouts
Status: Investigating
Updates: https://status.fitpulse.app
```

### Resolution

```
RESOLVED: Service API Degradation
Duration: 45 minutes
Cause: Database connection pool exhausted due to memory leak in biometric-service
Fix: Patched and redeployed biometric-service v2.1.1
Monitoring: All metrics normal, no data loss
```

### Post-Mortem

```
Post-Mortem: Service API Degradation (2026-05-06)

**Timeline**:
- 14:30 UTC: Alert triggered (error rate > 5%)
- 14:32 UTC: IC engaged, investigation started
- 14:40 UTC: Root cause identified (memory leak)
- 14:50 UTC: Patched version deployed
- 15:00 UTC: Service recovered

**Root Cause**: Memory leak in connection pooling logic

**Action Items**:
1. [DONE] Deploy memory leak fix v2.1.1
2. [TODO] Implement memory profiling in CI/CD
3. [TODO] Add memory threshold alerting (> 80% usage)
4. [TODO] Review connection pooling configuration

**Attendees**: Platform Team
**Date**: 2026-05-08 10:00 UTC
```

---

## Tools & Access

| Tool | URL | Purpose |
|------|-----|---------|
| PagerDuty | https://fitpulse.pagerduty.com | Incident tracking & on-call |
| Slack | #incidents, #alerts | Communication |
| Grafana | https://grafana.fitpulse.app:3000 | Dashboards & metrics |
| Kibana | https://kibana.fitpulse.app:5601 | Logs & analysis |
| Kubernetes | `kubectl` | Container orchestration |

---

## Incident Checklist

- [ ] Acknowledge alert in PagerDuty/Slack
- [ ] Declare incident, assign IC
- [ ] Gather initial information (What? When? Impact?)
- [ ] Assign Responder, Communications, Doc Writer
- [ ] Implement fix (rollback, scale, patch, etc.)
- [ ] Verify resolution (smoke tests, metrics)
- [ ] Update status page
- [ ] Schedule RCA within 48 hours
- [ ] Document lessons learned

---

**Last Updated**: 2026-05-06  
**Maintained By**: Security & Platform Teams