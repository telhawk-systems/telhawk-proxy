# GoTrack Pixel — streaming tracking pixel with pluggable sinks

A production‑minded, privacy‑aware tracking pixel written in Go that ingests pageview/interaction events and fans them out to multiple streaming sinks:

* **NDJSON log** (newline‑delimited JSON, 1 event per line)
* **Kafka** topic (configurable key/value)
* **Postgres** JSONB table (batched inserts / COPY)

It’s designed for **low-latency, high‑throughput** ingestion with backpressure, batching, and idempotency to support modern analytics and fraud‑detection pipelines.

---

## Highlights

* ⚡ **Fast path**: zero‑alloc hot loop where practical, pooled buffers, batched flushes
* 🔌 **Pluggable sinks**: enable any combo of `log`, `kafka`, `postgres`
* 🔁 **At‑least‑once** delivery with **idempotent event_id** to dedupe downstream
* 🧪 **Table‑driven tests** + integration tests via Docker Compose
* 🔒 **Privacy first**: optional IP hashing / GeoIP coarse resolution / DNT respect
* 📈 **Ops hooks**: `/healthz`, `/readyz`, Prometheus `/metrics`

---

## Quick start

### Requirements

* Go **1.22+**
* Docker & Docker Compose (for local Kafka/Postgres)

### 1) Clone & build

```bash
make build   # or: go build -o bin/gotrack ./cmd/gotrack
```

### 2) Bring up infra (local)

```bash
docker compose -f deploy/local/docker-compose.yml up -d
```

Services:

* **Kafka**: `localhost:9092`
* **Postgres**: `localhost:5432` (db `analytics`, user `analytics`, pw `analytics`)

### 3) Run the server

```bash
OUTPUTS=log,kafka,postgres \
LOG_PATH=./data/events.ndjson \
KAFKA_BROKERS=localhost:9092 \
KAFKA_TOPIC=gotrack.events \
PG_DSN="postgres://analytics:analytics@localhost:5432/analytics?sslmode=disable" \
SERVER_ADDR=":19890" \
TRUST_PROXY=true \
DNT_RESPECT=true \
./bin/gotrack
```

### 4) Fire test events

**Pixel GET**

```bash
curl -I "http://localhost:19890/px.gif?e=pageview&url=https%3A%2F%2Fexample.com%2F&uid=abc123&ref=https%3A%2F%2Fgoogle.com%2F"
```

**JSON POST**

```bash
curl -s http://localhost:19890/collect \
  -H 'content-type: application/json' \
  -d '{"e":"signup","uid":"abc123","url":"https://example.com/register","props":{"plan":"pro"}}'
```

---

## Event model

Each ingested event is normalized to the following JSON structure (keys present when known):

```json
{
  "event_id": "1b2e0f8a-e1e9-4a1e-9f77-6a5c8f99b0df",
  "ts": "2025-09-28T13:37:42.420Z",
  "e": "pageview",            // event name / type
  "uid": "abc123",            // stable user id if provided
  "sid": "s_...",             // anonymous session id (cookie)
  "url": "https://example.com/",
  "ref": "https://google.com/",
  "ua": "Mozilla/5.0 ...",    // user agent
  "ip": "203.0.113.42",       // optionally hashed / truncated
  "utm": {"source":"...","medium":"...","campaign":"..."},
  "geo": {"country":"US","region":"CA","city":"Los Angeles"}, // coarse
  "props": {"key":"value"}   // arbitrary custom payload
}
```

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

* `GET /healthz` → liveness
* `GET /readyz` → readiness (verifies sink connectivity)
* `GET /metrics` → Prometheus

---

## Configuration

All configuration is via environment variables (12‑factor). Common flags:

### General

* `SERVER_ADDR` (default `:19890`)
* `OUTPUTS` → comma list of enabled sinks: `log`, `kafka`, `postgres`
* `BATCH_SIZE` (default `100`), `FLUSH_INTERVAL_MS` (default `250`)
* `WORKER_CONCURRENCY` (default `4`)
* `TRUST_PROXY` (default `false`): honor `X-Forwarded-For`
* `GEOIP_DB` (optional path): enables coarse geo lookup
* `DNT_RESPECT` (default `true`): drop or anonymize on `DNT: 1`

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

---

## FAQ

**Q: Exactly‑once?**
A: Practically **at‑least‑once**. Use `event_id` for dedupe. Postgres `UNIQUE(event_id)` + Kafka compaction or consumer‑side dedupe recommended.

**Q: How big can `props` be?**
A: Keep it small (< 4KB). Enforce via `MAX_BODY_BYTES`.

**Q: Will ad blockers kill it?**
A: Some will. Host on your own subdomain and avoid obvious paths; provide `/collect` JSON fallback.

---

## License

MIT (see `LICENSE`).

---

## Credits

Built with ♥ in Go. Inspired by years of shipping analytics/fraud pipelines in fintech & e‑commerce.
