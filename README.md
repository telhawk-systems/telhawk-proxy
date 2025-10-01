# GoTrack

GoTrack is a **high-security tracking pixel and collection service** built in Go.  
It’s designed for **bot and hacker detection**, fraud monitoring, and operational telemetry — not adtech profiling.  
The platform is **privacy-aware, compliance-minded**, and ships with a hardened container image.

---

## ✨ Features

- **Security-focused event pipeline**  
  Collects requests via `/px.gif` (pixel) or `/collect` (JSON API), normalizes into structured events, and routes to pluggable sinks.

- **Pluggable outputs**  
  - Log sink → NDJSON lines for SIEM/SOC ingestion  
  - Kafka sink → scalable pipeline integration  
  - PostgreSQL sink → COPY-based high-throughput ingestion with JSONB schema + GIN indexes

- **Privacy & compliance aware**  
  - Honors **Do-Not-Track** headers  
  - Configurable sanitization of query params (`SANITIZE_PARAMS`)  
  - Explicitly designed to **avoid HIPAA, AML/KYC, FinCEN, GDPR violations**  
  - Collects identifiers strictly for **security and fraud detection (legitimate interest)**

- **Operational readiness**  
  - Health checks: `/healthz`, `/readyz`  
  - Prometheus metrics at `/metrics`  
  - Bounded queues with backpressure  
  - At-least-once delivery semantics

- **Production shipping**  
  - Multi-stage Docker build with Go 1.24+  
  - Final stage runs as **distroless Debian 12 nonroot**  
  - Minimal attack surface, non-root runtime

---

## 🚀 Quick Start

### Run locally (log sink only)

```bash
go build -o ./gotrack ./cmd/gotrack

OUTPUTS=log \
LOG_PATH=./events.ndjson \
SERVER_ADDR=":19890" \
./gotrack
```

### Run with test events (for testing sinks)

```bash
go build -o ./gotrack ./cmd/gotrack

TEST_MODE=true \
OUTPUTS=log \
LOG_PATH=./events.ndjson \
SERVER_ADDR=":19890" \
./gotrack
```

This will automatically generate 5 sample events after startup to test your sink configuration.

---

🗄 Event Model

All events are JSON with fixed top-level fields:
```json
{
  "event_id": "uuid",
  "timestamp": "2025-09-28T23:59:59Z",
  "ip": "203.0.113.42",
  "ua": "Mozilla/5.0 ...",
  "url": "https://example.com/?e=pageview",
  "payload": {
    "e": "pageview"
  }
}
```


* event_id ensures idempotency (downstream dedupe possible).
* Payload (payload) is JSON-typed for flexible attributes.
---

### Idempotency

* A UUID **event_id** is assigned per request when absent.
* Sinks should dedupe on `event_id` (Kafka key = `event_id`; Postgres unique index on `event_id`).

---

## HTTP interface

### `GET /px.gif`

Returns a 1×1 transparent GIF. Accepts query params (any unknowns go to `props`):

* `e` (string): event type (e.g., `pageview`)
* `uid` (string): user id
* `sid` (string): session id (auto-issued cookie if absent)
* `url`, `ref`, `utm_source`, `utm_medium`, `utm_campaign`, etc.

**Response**: `200` with `image/gif`, cache headers disabled. CORS allowlist optional.

### `POST /collect`

`Content-Type: application/json` with an event object or array of objects using the **Event model**.

### Health & metrics

* `GET /healthz` ➡️ liveness
* `GET /readyz` ➡️ readiness (verifies sink connectivity)
* `GET /metrics` ➡️ Prometheus

---

## Configuration

All configuration is via environment variables (12‑factor). Common flags:

### General

* `SERVER_ADDR` (default `:19890`)
* `OUTPUTS` ➡️ comma list of enabled sinks: `log`, `kafka`, `postgres`
* `BATCH_SIZE` (default `100`), `FLUSH_INTERVAL_MS` (default `250`)
* `WORKER_CONCURRENCY` (default `4`)
* `TRUST_PROXY` (default `false`): honor `X-Forwarded-For`
* `GEOIP_DB` (optional path): enables coarse geo lookup
* `DNT_RESPECT` (default `true`): drop or anonymize on `DNT: 1`
* `TEST_MODE` (default `false`): generate test events on startup for testing sinks

### HTTPS/TLS Configuration

* `ENABLE_HTTPS` (default `false`): enable HTTPS server instead of HTTP
* `SSL_CERT_FILE` (default `server.crt`): path to SSL certificate file
* `SSL_KEY_FILE` (default `server.key`): path to SSL private key file

**HTTPS Setup Example:**

```bash
# Generate self-signed certificates for testing (run once)
./generate-certs.sh

# Run with HTTPS enabled
ENABLE_HTTPS=true \
SSL_CERT_FILE=./server.crt \
SSL_KEY_FILE=./server.key \
OUTPUTS=log \
./gotrack
```

**Docker HTTPS Setup:**

```bash
# Create certificate directory
mkdir -p ./certs

# Copy your certificates to the certs directory
cp server.crt server.key ./certs/

# Update docker-compose.yml to enable HTTPS:
# Uncomment the HTTPS environment variables and volume mount
# Then run:
docker-compose up
```

**Production Notes:**
- Use certificates from a trusted Certificate Authority in production
- The included `generate-certs.sh` script creates self-signed certificates for testing only
- Mount certificates as read-only volumes in Docker containers
- Consider using Let's Encrypt or your organization's PKI for production certificates

### NDJSON log sink

* `LOG_PATH` (default `./events.ndjson`)
* `LOG_ROTATE_MB`, `LOG_BACKUPS`, `LOG_MAX_AGE_DAYS`

**Format**: newline‑delimited JSON, exactly the Event model per line.

### Kafka sink

* `KAFKA_BROKERS` (e.g., `localhost:9092,localhost:9093`)
* `KAFKA_TOPIC` (default `gotrack.events`)
* `KAFKA_ACKS` (default `all`), `KAFKA_COMPRESSION` (e.g., `snappy`)
* TLS/SASL: `KAFKA_SASL_MECHANISM`, `KAFKA_SASL_USER`, `KAFKA_SASL_PASSWORD`, `KAFKA_TLS_CA` (path), `KAFKA_TLS_SKIP_VERIFY`

**Record**: key = `event_id`, value = full JSON event. Headers include `event_type`, `schema=v1`.

### Postgres sink

* `PG_DSN` (e.g., `postgres://user:pass@host:5432/db?sslmode=disable`)
* `PG_TABLE` (default `events_json`)
* `PG_BATCH_SIZE` (default `500`), `PG_FLUSH_MS` (default `500`)
* `PG_COPY` (default `true`): prefer `COPY` over multi‑VALUES

Schema (baseline):

```sql
CREATE TABLE IF NOT EXISTS events_json (
  id BIGSERIAL PRIMARY KEY,
  event_id UUID UNIQUE NOT NULL,
  ts TIMESTAMPTZ NOT NULL DEFAULT now(),
  payload JSONB NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_json_ts ON events_json (ts);
CREATE INDEX IF NOT EXISTS idx_events_json_gin ON events_json USING GIN (payload);
```

Upsert example (idempotent):

```sql
INSERT INTO events_json (event_id, ts, payload)
VALUES ($1, $2, $3)
ON CONFLICT (event_id) DO NOTHING;
```

---

## Architecture

```
        ┌──────────────┐
        │  Browser/JS  │
        └──────┬───────┘
               │  GET /px.gif  |  POST /collect
               ▼
        ┌──────────────┐      batching / backpressure     ┌────────────┐
        │  HTTP Ingest │ ───────────────────────────────▶ │  Queue     │
        └──────┬───────┘                                   └────┬───────┘
               │  normalize + enrich (UA, IP, Geo, UTM)        │ workers
               ├───────────────────────────────────────────────▶│ log sink
               │                                               │
               ├───────────────────────────────────────────────▶│ kafka sink
               │                                               │
               └───────────────────────────────────────────────▶│ postgres sink
                                                               └────────────┘
```

**Delivery semantics**: at‑least‑once to each enabled sink. Use `event_id` for downstream dedupe.

**Backpressure**: bounded channels; if sinks stall, in‑memory queue slows intake; optional 429 on overflow.

---

## 🧪 Testing & Development

### Test Mode

GoTrack includes a built-in test mode that generates sample events for testing your sink configurations:

```bash
# Test locally with log sink only
TEST_MODE=true OUTPUTS=log ./gotrack

# Test all sinks (requires running Kafka/PostgreSQL)
TEST_MODE=true OUTPUTS=log,kafka,postgres ./gotrack

# Test specific configuration
TEST_MODE=true \
OUTPUTS=kafka \
KAFKA_BROKERS=localhost:9092 \
KAFKA_TOPIC=test.events \
./gotrack
```

**Test Events Generated:**
- `pageview` with UTM parameters and device info
- `click` with mobile device simulation  
- `conversion` event
- `pageview` with social media attribution (Facebook)
- `custom_event` with desktop browser info

Each event includes realistic data for:
- Unique `event_id` (UUID) for idempotency testing
- Timestamps with proper sequencing
- Device information (browser, OS, viewport)
- Session data (visitor_id, session_id)
- URL/UTM attribution data
- Geo information

### Management Scripts

Use the included management script for easy testing:

```bash
# Test locally (log sink only)
./deploy/manage.sh test-local

# Test with full Docker stack  
./deploy/manage.sh up
./deploy/manage.sh test-mode

# Manual HTTP tests
./deploy/manage.sh test-pixel
./deploy/manage.sh test-json
```

> 📖 **Detailed Testing Guide**: See [TESTMODE.md](TESTMODE.md) for comprehensive test mode documentation, event structure details, and verification methods.

---

## JS snippet (pixel)

Use a 1×1 GIF so ad/script blockers are less likely to interfere (still not guaranteed):

```html
<img src="https://collect.example.com/px.gif?e=pageview&url=" + encodeURIComponent(location.href) +
     "&ref=" + encodeURIComponent(document.referrer)
     width="1" height="1" style="display:none" alt="">
```

Or an async loader:

```html
<script>
(function(){
  var img = new Image(1,1);
  var q = new URLSearchParams({
    e: 'pageview',
    url: location.href,
    ref: document.referrer
  });
  img.src = 'https://collect.example.com/px.gif?' + q.toString();
})();
</script>
```

---

## Local development

### Docker Compose (snippet)

`deploy/local/docker-compose.yml`

```yaml
services:
  zookeeper:
    image: confluentinc/cp-zookeeper:7.6.0
    environment:
      ZOOKEEPER_CLIENT_PORT: 2181
  kafka:
    image: confluentinc/cp-kafka:7.6.0
    ports: ["9092:9092"]
    environment:
      KAFKA_ZOOKEEPER_CONNECT: zookeeper:2181
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092,PLAINTEXT_HOST://localhost:9092
      KAFKA_LISTENER_SECURITY_PROTOCOL_MAP: PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT
      KAFKA_INTER_BROKER_LISTENER_NAME: PLAINTEXT
      KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR: 1
    depends_on: [zookeeper]
  postgres:
    image: postgres:16
    ports: ["5432:5432"]
    environment:
      POSTGRES_DB: analytics
      POSTGRES_USER: analytics
      POSTGRES_PASSWORD: analytics
    volumes:
      - ./init.sql:/docker-entrypoint-initdb.d/init.sql
```

### Init Postgres

`deploy/local/init.sql` is applied automatically; it creates `events_json` with indexes.

### Run tests

```bash
make test   # or: go test ./...
```

### Quick Testing

```bash
# Build and test locally with generated events
go build -o ./gotrack ./cmd/gotrack
TEST_MODE=true OUTPUTS=log ./gotrack

# Test with Docker Compose stack
./deploy/manage.sh up
./deploy/manage.sh test-mode

# Verify events in each sink
tail -f out/events.ndjson              # Log files
./deploy/manage.sh kafka-console       # Kafka messages  
./deploy/manage.sh psql                # PostgreSQL: SELECT * FROM events_json;
```

---

## Observability

* **Logs**: structured JSON logs to stdout; per‑sink error counters
* **Metrics** (Prometheus): `requests_total`, `ingest_latency_seconds`, `queue_depth`, `sink_failures_total`, `batch_flush_seconds`
* **Tracing** (optional): OTEL export via `OTEL_EXPORTER_OTLP_ENDPOINT`

---

## Performance targets (baseline)

* Single instance on modest hardware: **10–20k req/s** pixel GETs with mixed sinks
* Latency p50 < 10ms (local), p99 < 50ms excluding network/Kafka/Postgres

Tuning knobs: `BATCH_SIZE`, `FLUSH_INTERVAL_MS`, `WORKER_CONCURRENCY`, Kafka compression, Postgres `COPY`.

---

## Security & privacy

* Respect **DNT** (drop or anonymize events)
* **PII minimization**: don’t collect emails/names; hash IPs with per‑day salt if you need uniqueness
* **Cookie**: httpOnly, SameSite=Lax; optional domain scoping
* **CORS**: origin allowlist for `/collect`; `px.gif` is cache‑busted, no‑store
* **TLS**: terminate at LB or enable built‑in TLS for dev

---

## Roadmap

* Redis/RabbitMQ sinks
* S3/GCS parquet writes via buffered rollups
* Schema registry for Kafka (Avro/Proto/JSON‑Schema)
* SQL matviews & example dashboards (Grafana/Metabase)
* Enhanced test mode with custom event templates
* Real-time event validation and alerting

---

## FAQ

**Q: Exactly‑once?**
A: Practically **at‑least‑once**. Use `event_id` for dedupe. Postgres `UNIQUE(event_id)` + Kafka compaction or consumer‑side dedupe recommended.

**Q: How big can `props` be?**
A: Keep it small (< 4KB). Enforce via `MAX_BODY_BYTES`.

**Q: Will ad blockers kill it?**
A: Some will. Host on your own subdomain and avoid obvious paths; provide `/collect` JSON fallback.

---

## Credits

Built with ❤️ in Go. Inspired by years of shipping analytics/fraud pipelines in fintech & e‑commerce.

# Licensing
Source code provided for demonstration and educational purposes only.

Not licensed for commercial deployment.