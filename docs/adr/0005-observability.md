# ADR 0005: Observability Implementation - Structured Logging, Prometheus Metrics, and Alerting

## Context

The system requires comprehensive observability to monitor health, performance, and security across microservices. This includes structured logging, metrics collection, and automated alerting to ensure reliability and quick incident response.

## Decision

Implement observability with:

1. **Structured JSON Logging**: All services must log in JSON format with mandatory fields:
   - timestamp (ISO8601 UTC)
   - level (DEBUG/INFO/WARN/ERROR/FATAL)
   - service (microservice name)
   - correlationId (UUID for request tracing)
   - userId (string|null)
   - action (UPPER_SNAKE_CASE semantic name)
   - Additional context fields as needed.

2. **Prometheus Metrics**: Required metrics set including:
   - request_duration_seconds (Histogram)
   - error_total (Counter)
   - classification_confidence (Gauge for ML)
   - db_connection_pool_usage (Gauge)
   - notification_queue_depth (Gauge)
   - biometric_sync_lag_seconds (Gauge)

3. **Alerting Rules**: Critical and warning alerts with escalation policies:
   - SEV-1: ServiceDown, DBConnectionPoolExhausted, BackupFailed
   - SEV-3: HighErrorRate, HighLatency, LowMLConfidence
   - Escalation: Immediate PagerDuty for SEV-1, Slack notifications with delays for others.

## Consequences

- Provides full visibility into system behavior and performance.
- Enables proactive monitoring and quick incident response.
- Supports compliance and operational requirements.

## Implementation

- Integrate structured logging libraries in Go and Python services.
- Configure Prometheus exporters and Grafana dashboards.
- Set up Alertmanager with Slack and PagerDuty integrations.
- Implement correlation ID propagation across services.

## Alternatives Considered

- Unstructured logging: Harder to search and analyze.
- Fewer metrics: Reduced observability.
- Manual alerting: Slower response times.