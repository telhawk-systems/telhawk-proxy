# GoTrack Prometheus Metrics

GoTrack includes a built-in Prometheus metrics server that exposes operational metrics on a separate port for monitoring and alerting.

## Configuration

Metrics are disabled by default and must be explicitly enabled. Configure using environment variables:

```bash
# Enable metrics server (required)
METRICS_ENABLED=true

# Bind address (default: 127.0.0.1:9090 - localhost only for security)
METRICS_ADDR=127.0.0.1:9090

# Optional TLS configuration
METRICS_TLS_CERT=/path/to/metrics.crt
METRICS_TLS_KEY=/path/to/metrics.key
METRICS_REQUIRE_TLS=false

# Optional mTLS for client authentication
METRICS_CLIENT_CA=/path/to/client-ca.crt
```

## Security Considerations

**⚠️ Important: Never expose the metrics endpoint publicly without authentication.**

The metrics server:
- Binds to localhost (127.0.0.1) by default
- Should be accessed only by Prometheus/monitoring systems
- Can be secured with TLS and mTLS
- Includes a health check at `/healthz`

## Available Metrics

### Event Processing
- `gotrack_events_ingested_total{sink}` - Total events successfully processed by sink type
- `gotrack_sink_errors_total{sink,error_type}` - Total errors writing to sinks
- `gotrack_queue_depth{sink}` - Current depth of internal event queues
- `gotrack_batch_flush_latency_seconds{sink}` - Batch flush timing to sinks

### HTTP Performance
- `gotrack_http_requests_total{endpoint,method,status}` - HTTP request counts
- `gotrack_http_duration_seconds{endpoint,method}` - HTTP response time distributions

### Standard Metrics
- Go runtime metrics (GC, memory, goroutines)
- Process metrics (CPU, memory, file descriptors)
- Prometheus scrape metrics

## Example Usage

### Start with Metrics
```bash
export METRICS_ENABLED=true
export METRICS_ADDR=127.0.0.1:9090
./gotrack
```

### Check Metrics
```bash
curl http://127.0.0.1:9090/metrics | grep gotrack
```

### Prometheus Configuration
```yaml
scrape_configs:
  - job_name: 'gotrack'
    static_configs:
      - targets: ['gotrack-host:9090']
    scrape_interval: 15s
    metrics_path: /metrics
    scheme: http
```

### Docker Compose Example
```yaml
version: '3.8'
services:
  gotrack:
    image: gotrack:latest
    environment:
      - METRICS_ENABLED=true
      - METRICS_ADDR=0.0.0.0:9090
    ports:
      - "19890:19890"    # Main HTTP server
      - "9090:9090"      # Metrics (internal access only)
    networks:
      - monitoring

  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
    networks:
      - monitoring

networks:
  monitoring:
    driver: bridge
```

## Example Queries

### Event Ingestion Rate
```promql
rate(gotrack_events_ingested_total[5m])
```

### Error Rate by Sink
```promql
rate(gotrack_sink_errors_total[5m]) / rate(gotrack_events_ingested_total[5m])
```

### 95th Percentile Response Time
```promql
histogram_quantile(0.95, rate(gotrack_http_duration_seconds_bucket[5m]))
```

### Request Rate by Endpoint
```promql
sum(rate(gotrack_http_requests_total[5m])) by (endpoint)
```

## Alerting Examples

### High Error Rate
```yaml
groups:
  - name: gotrack
    rules:
      - alert: GoTrackHighErrorRate
        expr: rate(gotrack_sink_errors_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "GoTrack experiencing high sink error rate"
```

### No Events Ingested (Potential Outage)
```yaml
      - alert: GoTrackNoEvents
        expr: rate(gotrack_events_ingested_total[5m]) == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "GoTrack not ingesting any events"
```

## Production Deployment

For production environments:

1. **Network Security**: Use Docker networks or Kubernetes NetworkPolicies to restrict access
2. **TLS**: Enable TLS for metrics endpoint
3. **Authentication**: Consider mTLS or proxy authentication
4. **Monitoring**: Set up alerts for critical metrics
5. **Backup**: Monitor both event ingestion and system health