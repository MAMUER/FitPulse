# Runbook: FitPulse Platform Operations

## Table of Contents
1. [Emergency Response](#emergency-response)
2. [Deployment Procedures](#deployment-procedures)
3. [Incident Response](#incident-response)
4. [Monitoring and Alerts](#monitoring-and-alerts)
5. [Data Recovery](#data-recovery)

---

## Emergency Response

### SEV-1: Service Down (< 5 minutes recovery SLA)

**Symptoms**: Service responds with 503, health check fails, high error rate

**Steps**:
1. **Acknowledge Alert** (Slack/PagerDuty)
   ```bash
   # Slack reaction :ack: or PagerDuty acknowledge
   ```

2. **Check Service Status**
   ```bash
   kubectl get pods -n fitness-platform
   kubectl describe pod <pod-name> -n fitness-platform
   ```

3. **Check Logs**
   ```bash
   kubectl logs <pod-name> -n fitness-platform --tail=100
   # Or via Kibana: index "fitness-logs-*", filter service="gateway"
   ```

4. **Restart Pod (if OOMKilled or CrashLoopBackOff)**
   ```bash
   kubectl delete pod <pod-name> -n fitness-platform
   # Pod will be recreated by deployment controller
   ```

5. **Rollback if Recent Deployment**
   ```bash
   kubectl rollout undo deployment/gateway -n fitness-platform
   ```

6. **Escalate to Tech Lead** if issue persists after 5 minutes

---

### SEV-2: High Error Rate (15-30 minutes investigation)

**Symptoms**: Error rate > 5%, latency p95 > 5s

**Steps**:
1. Check Grafana dashboard "FitPulse Service Overview"
2. Review logs in Kibana for patterns
   ```json
   {
     "level": "ERROR",
     "service": "biometric-service",
     "timestamp": "2026-05-06T*"
   }
   ```

3. Scale up if DB connection pool exhausted
   ```bash
   kubectl scale deployment biometric-service --replicas=3 -n fitness-platform
   ```

4. Check database replication lag
   ```bash
   psql -h postgres -U postgres -d fitness -c "SELECT slot_name, restart_lsn FROM pg_replication_slots;"
   ```

---

## Deployment Procedures

### Canary Deployment (9-Stage Pipeline)

```bash
# Stage 1-3: Development, Code Review, CI Build (automated)

# Stage 4: Deploy to Test
kubectl set image deployment/gateway-test \
  gateway=fitness-gateway:sha256:abc123 \
  -n fitness-platform

# Wait 5 minutes, verify health
kubectl get pods -n fitness-platform -l deployment=gateway-test

# Stage 5: Deploy to Staging
kubectl set image deployment/gateway-staging \
  gateway=fitness-gateway:sha256:abc123 \
  -n fitness-platform

# Run UAT and performance tests: k6, OWASP ZAP, chaos tests
make k6-load-test
make owasp-scan

# Stage 6: Create Release Candidate
git tag v2.1.0-rc1
git push origin v2.1.0-rc1

# Stage 7: Canary Deploy to Production
kubectl patch service gateway-canary -p \
  '{"spec":{"selector":{"version":"canary"}}}'

# Monitor for 1 hour
kubectl get hpa -n fitness-platform -w

# Success criteria:
# - Error rate < 1%
# - p95 latency < 3s
# - No critical logs

# Stage 7b: Rolling Deploy
kubectl set image deployment/gateway \
  gateway=fitness-gateway:sha256:abc123 \
  -n fitness-platform

# Stage 8: Post-Deploy Monitoring (24 hours)
# Dashboards: Error Rate, Latency, ML Confidence, DB Pool, Backup Status

# Stage 9: Auto-Rollback Trigger
# Automatically triggered if:
# - error_rate > 5% for 15 minutes
# - latency p95 > 10s for 15 minutes
# - CRITICAL security issue detected
kubectl rollout undo deployment/gateway -n fitness-platform
```

### Manual Rollback

```bash
# View rollout history
kubectl rollout history deployment/gateway -n fitness-platform

# Rollback to previous version
kubectl rollout undo deployment/gateway -n fitness-platform

# Rollback to specific revision
kubectl rollout undo deployment/gateway --to-revision=5 -n fitness-platform

# Verify rollback
kubectl get pods -n fitness-platform -l app=gateway
kubectl logs -f deployment/gateway -n fitness-platform
```

---

## Incident Response

### Database Connection Pool Exhausted (SEV-1)

**Alert**: `db_connection_pool_usage > 0.9`

**Steps**:
1. **Check active connections**
   ```sql
   SELECT datname, count(*) FROM pg_stat_activity GROUP BY datname;
   ```

2. **Identify long-running queries**
   ```sql
   SELECT query, duration FROM pg_stat_statements 
   ORDER BY duration DESC LIMIT 5;
   ```

3. **Scale up service replicas**
   ```bash
   kubectl scale deployment user-service --replicas=5 -n fitness-platform
   ```

4. **Monitor pool recovery**
   ```bash
   # Watch Grafana: db_connection_pool_usage metric
   ```

### Backup Failed (SEV-1)

**Alert**: `backup_success{type='full'} == 0`

**Steps**:
1. **Check backup job logs**
   ```bash
   kubectl get pods -n fitness-platform -l job-name=backup
   kubectl logs -f backup-job -n fitness-platform
   ```

2. **Verify disk space**
   ```bash
   df -h /var/lib/postgresql/data
   ```

3. **Manually trigger backup**
   ```bash
   # Via backup script
   scripts/backup-db.sh --encrypted --s3-upload

   # Verify
   aws s3 ls s3://fitness-backups/
   ```

### Low ML Model Confidence (SEV-4)

**Alert**: `classification_confidence < 0.7`

**Steps**:
1. **Check model versions**
   ```bash
   curl http://ml-classifier:8001/model-info
   ```

2. **Review recent predictions**
   ```bash
   # Kibana query: service="ml-classifier" AND action="CLASSIFY" AND confidence < 0.7
   ```

3. **Trigger model retraining**
   ```bash
   # Via ML service admin endpoint
   curl -X POST http://ml-classifier:8001/retrain \
     -H "Authorization: Bearer $ML_ADMIN_TOKEN"
   ```

4. **Create incident ticket** for ML team to investigate drift

---

## Monitoring and Alerts

### Key Metrics to Watch

| Metric | Threshold | Check Frequency |
|--------|-----------|-----------------|
| Error Rate | < 5% | Continuous (1m) |
| p95 Latency | < 5s | Continuous (1m) |
| Uptime | > 99.9% | Daily |
| DB Pool Usage | < 80% | Every 5m |
| Backup Success | 100% | Every 6h |
| ML Confidence | > 0.7 | Every 15m |

### Grafana Dashboard Access

```
URL: https://grafana.fitpulse.app:3000
Username: admin
Password: ${GRAFANA_ADMIN_PASSWORD}
```

**Default Dashboards**:
- `FitPulse Service Overview`: Request rate, error rate, latency, ML metrics
- `Database Performance`: Connections, query time, replication lag
- `ELK Stack Health`: Elasticsearch indices, Logstash throughput

### Elasticsearch Snapshot for 90-day Rotation

```bash
# Create snapshot repository (one-time setup)
curl -X PUT "elasticsearch:9200/_snapshot/s3-backup" \
  -H "Content-Type: application/json" \
  -d '{
    "type": "s3",
    "settings": {
      "bucket": "fitness-logs-backup",
      "base_path": "snapshots"
    }
  }'

# Create daily snapshot
curl -X PUT "elasticsearch:9200/_snapshot/s3-backup/snapshot-$(date +%Y-%m-%d)" \
  -H "Content-Type: application/json" \
  -d '{"indices": "fitness-logs-*"}'

# Archive old indices to cold storage (> 90 days)
curl -X POST "elasticsearch:9200/fitness-logs-2026.01.01/_close"
aws s3 cp elasticsearch-snapshot-2026.01.01.tar.gz s3://fitness-logs-archive/
```

---

## Data Recovery

### PostgreSQL Point-in-Time Recovery (PITR)

```bash
# 1. Stop current PostgreSQL instance
kubectl scale deployment postgres --replicas=0 -n fitness-platform

# 2. Restore from backup
scripts/restore-db.sh --backup-file=fitness-backup-2026-05-05.sql.enc \
  --target-time="2026-05-05T14:30:00Z" \
  --use-wal-archive

# 3. Verify data integrity
psql -h localhost -U postgres -d fitness \
  -c "SELECT COUNT(*) FROM users; SELECT MAX(created_at) FROM biometric_data;"

# 4. Restart PostgreSQL
kubectl scale deployment postgres --replicas=1 -n fitness-platform

# 5. Monitor replication to replicas
kubectl logs -f postgres-replica-0 -n fitness-platform
```

### Elasticsearch Data Recovery

```bash
# 1. List available snapshots
curl "elasticsearch:9200/_snapshot/s3-backup/_all"

# 2. Restore specific indices
curl -X POST "elasticsearch:9200/_snapshot/s3-backup/snapshot-2026-05-05/_restore" \
  -H "Content-Type: application/json" \
  -d '{
    "indices": "fitness-logs-2026.05.05",
    "rename_pattern": "(.+)",
    "rename_replacement": "$1-restored"
  }'

# 3. Verify restored indices
curl "elasticsearch:9200/_cat/indices?v" | grep restored

# 4. Merge back into live indices (if needed)
```

---

## Contact & Escalation

- **On-Call Engineer**: See PagerDuty schedule
- **Tech Lead**: tech-lead@fitpulse.app
- **CTO**: cto@fitpulse.app (SEV-1 only, escalate after 15 min)

---

**Last Updated**: 2026-05-06  
**Maintained By**: Platform Team