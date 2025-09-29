# GoTrack Deployment Guide

## Quick Start with Docker Compose

The complete GoTrack stack includes:
- **GoTrack** application with all three sinks
- **Apache Kafka** for stream processing
- **PostgreSQL** for persistent analytics storage
- **Persistent volumes** for data durability

### 1. Start the Stack

```bash
# Start all services
./deploy/manage.sh up

# Follow logs
./deploy/manage.sh logs
```

### 2. Test the Setup

```bash
# Test pixel tracking
./deploy/manage.sh test-pixel

# Test JSON API
./deploy/manage.sh test-json

# Check PostgreSQL data
./deploy/manage.sh psql
```

### 3. Monitor Data Flow

```bash
# View Kafka messages
./deploy/manage.sh kafka-console

# Query PostgreSQL
./deploy/manage.sh psql
# Then run: SELECT * FROM events_view LIMIT 10;

# Check log files
tail -f out/events.ndjson
```

## Architecture

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Browser   │───▶│   GoTrack    │───▶│ Log Files   │
│  /px.gif    │    │ :19890       │    │ ./out/*.log │
│  /collect   │    │              │    └─────────────┘
└─────────────┘    │              │    ┌─────────────┐
                   │              │───▶│ Kafka       │
                   │              │    │ :9092       │
                   │              │    │ Topic:      │
                   │              │    │ gotrack.*   │
                   │              │    └─────────────┘
                   │              │    ┌─────────────┐
                   │              │───▶│ PostgreSQL  │
                   │              │    │ :5432       │
                   └──────────────┘    │ events_json │
                                       └─────────────┘
```

## Configuration

All sinks are enabled by default with production-ready settings:

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OUTPUTS` | `log,kafka,postgres` | Enabled sinks |
| `SERVER_ADDR` | `:19890` | HTTP server address |

### Kafka Settings
| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BROKERS` | `kafka:29092` | Kafka broker addresses |
| `KAFKA_TOPIC` | `gotrack.events` | Topic for events |
| `KAFKA_ACKS` | `all` | Acknowledgment level |
| `KAFKA_COMPRESSION` | `snappy` | Compression type |

### PostgreSQL Settings
| Variable | Default | Description |
|----------|---------|-------------|
| `PG_DSN` | `postgres://analytics:analytics@postgres:5432/analytics?sslmode=disable` | Connection string |
| `PG_TABLE` | `events_json` | Table name |
| `PG_BATCH_SIZE` | `500` | Batch size for writes |
| `PG_FLUSH_MS` | `500` | Flush interval (ms) |
| `PG_COPY` | `true` | Use COPY for high throughput |

## Data Persistence

All data is persisted in Docker volumes:
- `kafka_data` - Kafka logs and topics
- `postgres_data` - PostgreSQL database
- `zookeeper_data` & `zookeeper_logs` - Zookeeper state
- `./out/` - Log files (host-mounted)

## Production Considerations

### Scaling
- **Kafka**: Add more brokers by scaling the kafka service
- **PostgreSQL**: Use read replicas or sharding for high load
- **GoTrack**: Run multiple instances behind a load balancer

### Monitoring
- Check `/metrics` endpoint for Prometheus metrics
- Use `/healthz` and `/readyz` for health checks
- Monitor Kafka lag and PostgreSQL connection pool

### Security
- Enable TLS for Kafka and PostgreSQL in production
- Use proper authentication and authorization
- Consider network policies for container communication

### Backup
- PostgreSQL: Use pg_dump or streaming replication
- Kafka: Enable topic replication factor > 1
- Log files: Regular rotation and archival

## Troubleshooting

### Common Issues

1. **Port conflicts**: Change ports in docker-compose.yml
2. **Memory issues**: Adjust JVM settings for Kafka/Zookeeper
3. **PostgreSQL connection errors**: Check health check and wait for initialization

### Debug Commands

```bash
# Check service status
./deploy/manage.sh status

# View all logs
docker-compose logs

# Connect to containers
docker-compose exec gotrack /bin/sh
docker-compose exec postgres bash
docker-compose exec kafka bash

# Reset everything (destructive!)
./deploy/manage.sh clean
```

## SQL Analytics Examples

Connect with `./deploy/manage.sh psql` and try these queries:

```sql
-- Event counts by type (last 24h)
SELECT 
    payload->>'type' as event_type, 
    COUNT(*) as count 
FROM events_json 
WHERE ts >= NOW() - INTERVAL '1 day' 
GROUP BY payload->>'type'
ORDER BY count DESC;

-- Top referrers
SELECT 
    payload->'url'->>'referrer' as referrer, 
    COUNT(*) as visits 
FROM events_json 
WHERE payload->>'type' = 'pageview' 
    AND ts >= NOW() - INTERVAL '1 day'
    AND payload->'url'->>'referrer' IS NOT NULL 
GROUP BY payload->'url'->>'referrer' 
ORDER BY visits DESC 
LIMIT 10;

-- Browser distribution
SELECT 
    payload->'device'->>'browser' as browser, 
    COUNT(DISTINCT payload->'session'->>'visitor_id') as unique_visitors
FROM events_json 
WHERE ts >= NOW() - INTERVAL '1 day' 
GROUP BY payload->'device'->>'browser' 
ORDER BY unique_visitors DESC;

-- Hourly event volume
SELECT 
    DATE_TRUNC('hour', ts) as hour,
    COUNT(*) as events
FROM events_json 
WHERE ts >= NOW() - INTERVAL '24 hours'
GROUP BY DATE_TRUNC('hour', ts)
ORDER BY hour;
```