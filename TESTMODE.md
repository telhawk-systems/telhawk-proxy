# GoTrack Test Mode Guide

## Overview

Test Mode automatically generates realistic sample events when GoTrack starts up. This is essential for:

- **Sink Verification**: Ensure all configured sinks (log, Kafka, PostgreSQL) receive events
- **Configuration Testing**: Validate connection strings, authentication, and settings
- **Development**: Quick iteration without manual HTTP requests
- **CI/CD**: Automated testing of the complete pipeline

## Usage

Enable test mode by setting `TEST_MODE=true`:

```bash
# Basic usage
TEST_MODE=true ./gotrack

# With specific sinks
TEST_MODE=true OUTPUTS=log,kafka ./gotrack

# Full configuration test
TEST_MODE=true \
OUTPUTS=log,kafka,postgres \
KAFKA_BROKERS=localhost:9092 \
PG_DSN="postgres://user:pass@localhost/db" \
./gotrack
```

## Generated Test Events

Test mode creates 5 diverse events to thoroughly test your pipeline:

### Event 1: Organic Search Pageview
```json
{
  "event_id": "uuid-1",
  "type": "pageview",
  "url": {
    "referrer": "https://google.com",
    "utm": {
      "source": "google",
      "medium": "organic", 
      "campaign": "search"
    }
  },
  "device": {
    "browser": "Chrome",
    "os": "Windows",
    "viewport_w": 1920,
    "viewport_h": 1080
  }
}
```

### Event 2: Mobile Click Event  
```json
{
  "event_id": "uuid-2",
  "type": "click",
  "device": {
    "browser": "Safari",
    "os": "iOS",
    "ua_mobile": true,
    "viewport_w": 375,
    "viewport_h": 812
  }
}
```

### Event 3: Conversion (Desktop Firefox)
```json
{
  "event_id": "uuid-3", 
  "type": "conversion",
  "device": {
    "browser": "Firefox",
    "os": "Linux"
  }
}
```

### Event 4: Social Media Campaign
```json
{
  "event_id": "uuid-4",
  "type": "pageview", 
  "url": {
    "referrer": "https://facebook.com",
    "utm": {
      "source": "facebook",
      "medium": "social",
      "campaign": "spring_sale"
    },
    "meta": {
      "fbclid": "fb_click_123",
      "campaign_id": "camp_456"
    }
  }
}
```

### Event 5: Custom Event
```json
{
  "event_id": "uuid-5",
  "type": "custom_event",
  "device": {
    "browser": "Firefox",
    "os": "Windows"
  }
}
```

## Verification Methods

### 1. Log Files
```bash
# View generated events
tail -f ./events.ndjson | jq .

# Count events
wc -l ./events.ndjson

# Check specific event types
jq 'select(.type == "pageview")' ./events.ndjson
```

### 2. Kafka Topics
```bash
# Console consumer
kafka-console-consumer --bootstrap-server localhost:9092 \
  --topic gotrack.events --from-beginning

# With Docker Compose
./deploy/manage.sh kafka-console
```

### 3. PostgreSQL Queries
```sql
-- Event count by type
SELECT 
    payload->>'type' as event_type,
    COUNT(*) as count
FROM events_json 
GROUP BY payload->>'type';

-- Recent test events
SELECT 
    event_id,
    payload->>'type' as type,
    payload->'device'->>'browser' as browser
FROM events_json 
WHERE ts >= NOW() - INTERVAL '5 minutes'
ORDER BY ts;

-- Verify idempotency (should be 5 unique events)
SELECT COUNT(DISTINCT event_id) FROM events_json;
```

## Use Cases

### Local Development
```bash
# Quick smoke test
TEST_MODE=true OUTPUTS=log ./gotrack

# Test Kafka locally  
TEST_MODE=true OUTPUTS=kafka KAFKA_BROKERS=localhost:9092 ./gotrack
```

### Docker Compose Stack
```bash
# Enable in docker-compose.yml
environment:
  - TEST_MODE=true
  
# Or override
docker-compose run -e TEST_MODE=true gotrack
```

### CI/CD Pipeline
```yaml
# Example GitHub Actions
- name: Test GoTrack Pipeline
  run: |
    docker-compose up -d postgres kafka
    TEST_MODE=true OUTPUTS=postgres,kafka ./gotrack &
    sleep 5
    # Verify data in sinks
    docker-compose exec postgres psql -c "SELECT COUNT(*) FROM events_json;"
```

### Integration Testing
```bash
# Test all sinks with realistic data
TEST_MODE=true \
OUTPUTS=log,kafka,postgres \
LOG_PATH=./integration_test.ndjson \
KAFKA_TOPIC=test.events \
PG_TABLE=test_events \
./gotrack
```

## Timing & Behavior

- **Delay**: 2-second wait after startup for sink initialization
- **Interval**: 200ms between each event (total ~1 second for all 5)
- **Shutdown**: Application continues running after test events
- **Idempotency**: Each run generates new UUIDs (safe to run multiple times)

## Troubleshooting

### No Events Generated
```bash
# Check if test mode is enabled
echo $TEST_MODE

# Check logs for errors
TEST_MODE=true ./gotrack 2>&1 | grep -i test
```

### Sink Connection Failures
```bash
# Test sinks individually
TEST_MODE=true OUTPUTS=log ./gotrack     # Should always work
TEST_MODE=true OUTPUTS=kafka ./gotrack   # Check Kafka connection
TEST_MODE=true OUTPUTS=postgres ./gotrack # Check PostgreSQL connection
```

### Event Count Mismatch
```bash
# PostgreSQL: Check for constraint violations
SELECT * FROM events_json WHERE event_id IN (
  SELECT event_id FROM events_json GROUP BY event_id HAVING COUNT(*) > 1
);

# Kafka: Check consumer lag
kafka-consumer-groups --bootstrap-server localhost:9092 --describe --group your-group
```

## Best Practices

1. **Always test locally first**: `TEST_MODE=true OUTPUTS=log`
2. **Verify each sink individually** before testing multiple sinks
3. **Use unique topics/tables** for testing to avoid production data confusion
4. **Check logs** for detailed error messages during sink failures
5. **Run test mode in CI/CD** to catch configuration regressions