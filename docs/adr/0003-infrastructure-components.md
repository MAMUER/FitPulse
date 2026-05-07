# ADR 0003: Infrastructure Components Selection - RabbitMQ, ELK Stack, and Prometheus

## Context

The project requires reliable asynchronous messaging, centralized logging, and metrics collection to support microservices architecture with high observability and resilience. We need to choose components that are production-ready, scalable, and integrate well with Kubernetes.

## Decision

We will use:

1. **RabbitMQ** as the message broker for asynchronous communication between microservices.
   - Purpose: Handle queues for notifications, background biometric data processing, and inter-service synchronization.
   - Requirements: Persistent queues, mirrored queues for durability, dead letter queues for failed messages, and monitoring of queue depth, consumer lag, and message rates.

2. **ELK Stack (Elasticsearch, Logstash, Kibana)** for centralized logging and analysis.
   - Purpose: Store and analyze logs from all services.
   - Retention: 90 days hot storage + archiving to S3.
   - Requirements: Structured JSON logging with mandatory fields (timestamp, level, correlationId, userId, action), indexing by service, action, error_code, and role-based access in Kibana (dev: read-only, security: full access).

3. **Prometheus + Grafana** for metrics collection, storage, and visualization.
   - Purpose: Monitor service health, performance, and business metrics.
   - Requirements: Service discovery via Kubernetes annotations, recording rules for pre-aggregated metrics, and Alertmanager integration with Slack/PagerDuty.

## Consequences

- **Positive**: Provides robust, scalable infrastructure for messaging, logging, and monitoring that supports the microservices architecture.
- **Negative**: Adds complexity in deployment and maintenance, requires expertise in these tools.
- **Risks**: Potential vendor lock-in, but all chosen tools are open-source and widely adopted.

## Implementation

- RabbitMQ: Deploy as Kubernetes StatefulSet with persistent volumes and mirroring policies.
- ELK: Deploy Elasticsearch cluster, Logstash for ingestion, Kibana for visualization, with appropriate security configurations.
- Prometheus: Deploy with service discovery, configure recording rules and alerting rules as specified.

## Alternatives Considered

- Kafka instead of RabbitMQ: More scalable for high-throughput, but RabbitMQ is simpler for our use case.
- Loki + Grafana instead of ELK: Lighter for logs, but ELK provides better search and analysis capabilities.
- Other monitoring stacks: Datadog, New Relic - but Prometheus is open-source and integrates well with Kubernetes.