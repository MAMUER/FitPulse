# FitPulse ELK Stack Configuration

## Overview

Конфигурация Elasticsearch, Logstash и Kibana для централизованного логирования FitPulse.

- **Retention**: 3 года (hot/warm/cold/frozen tiers) для соответствия 152-ФЗ.
- **Index**: `fitpulse-logs-YYYY.MM.dd` с ILM policy.
- **Template**: оптимизирован для поиска и агрегаций.

## Files

```
configs/elk/
├── ilm-policy/
│   └── fitpulse-logs-3y.json       # ILM policy: 3-year retention
├── templates/
│   └── fitpulse-logs-template.json # Index template
├── logstash/
│   └── fitpulse.conf               # Logstash pipeline
└── README.md                       # This file
```

## Setup

### 1. Elasticsearch

```bash
# Load ILM policy
curl -X PUT "http://localhost:9200/_ilm/policy/fitpulse-logs-policy" \
  -H "Content-Type: application/json" \
  -d @configs/elk/ilm-policy/fitpulse-logs-3y.json

# Load index template
curl -X PUT "http://localhost:9200/_index_template/fitpulse-logs-template" \
  -H "Content-Type: application/json" \
  -d @configs/elk/templates/fitpulse-logs-template.json
```

### 2. Logstash

```bash
# Set environment variables
export ELASTICSEARCH_HOSTS="elasticsearch:9200"
export ELASTICSEARCH_USER="elastic"
export ELASTICSEARCH_PASSWORD="changeme"

# Start Logstash
docker run --rm -it \
  -v ${PWD}/configs/elk/logstash/fitpulse.conf:/usr/share/logstash/pipeline/fitpulse.conf \
  -e ELASTICSEARCH_HOSTS=$ELASTICSEARCH_HOSTS \
  -e ELASTICSEARCH_USER=$ELASTICSEARCH_USER \
  -e ELASTICSEARCH_PASSWORD=$ELASTICSEARCH_PASSWORD \
  docker.elastic.co/logstash/logstash:8.12.0
```

### 3. Kibana

```bash
# Start Kibana
docker run --rm -it \
  -p 5601:5601 \
  -e ELASTICSEARCH_HOSTS="http://elasticsearch:9200" \
  -e ELASTICSEARCH_USER="elastic" \
  -e ELASTICSEARCH_PASSWORD="changeme" \
  docker.elastic.co/kibana/kibana:8.12.0
```

## Retention Policy

| Tier | Age | Actions |
|---|---|---|
| **Hot** | 0-7 days | Read/write, rollover at 50GB or 7 days |
| **Warm** | 7-30 days | Read-only, shrink to 1 shard, force merge |
| **Cold** | 30-90 days | Frozen (searchable snapshot) |
| **Frozen** | 90 days - 3 years | Searchable snapshot in S3 |
| **Delete** | 3 years | Automatic deletion |

## Compliance

- **152-ФЗ**: Retention audit logs 3 года.
- **GDPR Art. 5(1)(e)**: Storage limitation.
- **Роскомнадзор**: Audit log доступен для запросов.

## Security

- RBAC: роли `admin`, `platform-team`, `auditor`.
- TLS 1.3 для всех соединений.
- Audit logging всех действий в Kibana.

## Monitoring

- Alertmanager алерты на:
  - Падение indexing throughput.
  - Ошибки в pipeline.
  - Заполнение hot tier (> 80% disk usage).
