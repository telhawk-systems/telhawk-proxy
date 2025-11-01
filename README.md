# TelHawk Proxy

[![Tests](https://github.com/telhawk-systems/telhawk-proxy/actions/workflows/test-coverage.yml/badge.svg)](https://github.com/telhawk-systems/telhawk-proxy/actions/workflows/test-coverage.yml)
[![Coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/shortontech/06522d3b723a877fce2c749350f6dc83/raw/telhawk-proxy-coverage.json)](https://github.com/telhawk-systems/telhawk-proxy/actions/workflows/test-coverage.yml)
[![JS Tests](https://github.com/telhawk-systems/telhawk-proxy/actions/workflows/js-test-coverage.yml/badge.svg)](https://github.com/telhawk-systems/telhawk-proxy/actions/workflows/js-test-coverage.yml)
[![JS Coverage](https://img.shields.io/endpoint?url=https://gist.githubusercontent.com/shortontech/06522d3b723a877fce2c749350f6dc83/raw/telhawk-proxy-js-coverage.json)](https://github.com/telhawk-systems/telhawk-proxy/actions/workflows/js-test-coverage.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/telhawk-systems/telhawk-proxy)](https://goreportcard.com/report/github.com/telhawk-systems/telhawk-proxy)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

TelHawk Proxy is a **transparent reverse proxy with built-in telemetry collection** built in Go.  
It sits between your users and your application, automatically instrumenting web traffic by injecting tracking code into HTML responses. This enables **real-time visibility** into user behavior, bot detection, fraud monitoring, and operational telemetry — all without requiring code changes to your application.  
The platform is **privacy-aware, compliance-minded**, and ships with a hardened container image designed for security-focused teams who need visibility without compromising user privacy.

---

## What TelHawk Proxy Does

TelHawk Proxy acts as a **reverse proxy** that:

1. **Intercepts all web traffic** between users and your application
2. **Automatically injects tracking code** into every HTML response (no code changes needed)
3. **Collects telemetry data** from user interactions, device fingerprints, and behavior patterns
4. **Routes data to your analytics pipeline** via pluggable outputs (logs, Kafka, PostgreSQL)
5. **Transparently forwards** all other requests to your backend application

**Use Cases:**

- **Operational visibility**: Real-time monitoring of user journeys and application usage
- **Security monitoring**: Bot detection, fraud prevention, and anomaly detection
- **Compliance**: Privacy-aware telemetry that avoids GDPR/HIPAA violations
- **Performance analytics**: Track page load times, user flows, and conversion funnels

**What TelHawk Proxy is NOT:**

- Not an adtech/tracking platform for profiling or targeting
- Not a replacement for application monitoring (APM) tools
- Not a general-purpose load balancer or API gateway

---

## ✨ Features

- **Transparent Proxy with Auto-Injection**  
  Operates as a reverse proxy that automatically injects tracking JavaScript and pixels into all HTML responses. Supports gzip compression and maintains full transparency for non-HTML content.

- **Stealth Tracking Mode**  
  Posts tracking data to the same URLs as regular page requests using HMAC headers for identification. Resistant to ad-blockers and script-blocking extensions.

- **HMAC-Authenticated Collection**  
  IP-specific HMAC-SHA256 authentication ensures data integrity and prevents forged tracking data. Automatic key derivation per client.

- **Pluggable outputs**  
  - Log sink → NDJSON lines for SIEM/SOC ingestion  
  - Kafka sink → scalable pipeline integration  
  - PostgreSQL sink → COPY-based high-throughput ingestion with JSONB schema + GIN indexes

- **Privacy & compliance aware**  
  - Explicitly designed to **avoid HIPAA, AML/KYC, FinCEN, GDPR violations**  
  - Collects identifiers strictly for **security and fraud detection (legitimate interest)**

- **Operational readiness**  
  - Health checks: `/healthz`, `/readyz`  
  - **Prometheus metrics server** on separate port with 20+ metrics  
  - Bounded queues with backpressure  
  - At-least-once delivery semantics

- **Production shipping**  
  - Multi-stage Docker build with Go 1.24+  
  - Final stage runs as **distroless Debian 12 nonroot**  
  - Minimal attack surface, non-root runtime

---

## 🚀 Quick Start

### Understanding the Architecture

TelHawk Proxy sits **between your users and your application** as a reverse proxy:

```
[Users] → [TelHawk Proxy :8080] → [Your Application :3000]
            ↓
    [Telemetry Pipeline]
    (Logs/Kafka/PostgreSQL)
```

When a user requests a page:
1. TelHawk Proxy receives the request and forwards it to your application
2. Your application returns an HTML response
3. TelHawk Proxy automatically injects tracking JavaScript and a 1x1 pixel
4. The instrumented HTML is sent to the user
5. User interactions generate telemetry events collected by TelHawk Proxy
6. Events are routed to your configured outputs (logs, Kafka, PostgreSQL)

### Basic proxy setup

```bash
go build -o ./telhawk-proxy ./cmd/telhawk-proxy

HMAC_SECRET=your-secret-key \
FORWARD_DESTINATION=http://your-site.com \
SERVER_ADDR=":19899" \
OUTPUTS=log \
LOG_PATH=./events.ndjson \
./telhawk-proxy
```

### Run with test events (for testing sinks)

```bash
go build -o ./telhawk-proxy ./cmd/telhawk-proxy

TEST_MODE=true \
HMAC_SECRET=your-secret-key \
FORWARD_DESTINATION=http://example.com \
OUTPUTS=log \
LOG_PATH=./events.ndjson \
SERVER_ADDR=":19890" \
./telhawk-proxy
```

This will automatically generate 5 sample events after startup to test your sink configuration.

### Run with metrics enabled

```bash
# Enable Prometheus metrics on separate port (secure localhost binding)
METRICS_ENABLED=true \
METRICS_ADDR=127.0.0.1:9090 \
HMAC_SECRET=your-secret-key \
FORWARD_DESTINATION=http://your-site.com \
OUTPUTS=log \
./telhawk-proxy

# Check metrics
curl http://127.0.0.1:9090/metrics | grep telhawk
```

See [METRICS.md](METRICS.md) for full monitoring and alerting documentation.

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

### `POST /`

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
./telhawk-proxy
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

### Transparent Proxy Mode (Always Enabled)

TelHawk Proxy operates exclusively as a **reverse proxy**, automatically injecting tracking code into all HTML responses. All non-tracking requests are transparently forwarded to the destination server.

* `FORWARD_DESTINATION` (required): destination URL to proxy all requests to
* `HMAC_SECRET` (required): secret key for HMAC authentication and tracking security

**Basic Setup:**

```bash
# Run TelHawk Proxy as a transparent tracking proxy
HMAC_SECRET=your-secret-key \
FORWARD_DESTINATION=http://localhost:3000 \
OUTPUTS=log \
SERVER_ADDR=:8080 \
./telhawk-proxy
```

**How It Works:**

- **Tracking endpoints** (`/px.gif`, `/collect`, `/healthz`, `/readyz`, `/metrics`, `/hmac.js`) are handled by TelHawk Proxy
- **All other requests** are proxied to the `FORWARD_DESTINATION` server  
- **HTML responses** automatically get tracking JavaScript and pixel injected
- **POST requests with HMAC header** are routed to collection handler (stealth mode)
- **Regular POST requests** (no HMAC) are proxied normally to destination
- Headers, query parameters, and request bodies are preserved during proxy

**Automatic Tracking Injection:**

TelHawk Proxy automatically injects into every HTML response:
- ✅ **Full JavaScript tracking library** (43KB inlined) - ad-blocker resistant
- ✅ **1x1 transparent pixel** as fallback
- ✅ **HMAC authentication script** (when HMAC_SECRET is set)
- ✅ **Only modifies HTML** - never touches JSON, CSS, JS, images, etc.
- ✅ **Injects before `</body>`** tag or before `</html>` as fallback
- ✅ **Handles gzip compression** - decompresses, injects, recompresses
- ✅ **Updates Content-Length** header automatically

**Injected Content:**
```html
<script src="/hmac.js"></script>
<script>(full 43KB tracking library inlined here)</script>
<img src="/px.gif?e=pageview&auto=1&url=%2F" width="1" height="1" style="display:none" alt="">
```

**Stealth Mode:**
- Tracking data POSTs to the **same URL** as page requests (not `/collect`)
- HMAC header identifies tracking requests server-side
- Ad-blockers can't detect suspicious endpoints
- Works even when external script loading is blocked

**Example Architecture:**
```
[Client] → [TelHawk Proxy :8080] → [Your App :3000]
           ↓ (injects tracking + collects data)
         [Analytics Pipeline]
```

### HMAC Authentication (Required)

TelHawk Proxy requires HMAC-SHA256 authentication to identify tracking requests and prevent forged data:

* `HMAC_SECRET` (required): Master secret key for HMAC generation/verification
* `HMAC_PUBLIC_KEY` (optional): Override the derived public key with a custom base64-encoded key

**HMAC Security Model:**
- Uses **IP-derived keys**: Each client IP gets a unique HMAC key derived from `HMAC_SECRET + IP`
- **SHA-256 based**: Uses HMAC-SHA256 for cryptographic integrity
- **Header-based**: HMAC signature sent via `X-TelHawk Proxy-HMAC` header
- **Replay protection**: Different IPs cannot reuse each other's signatures

**Setup HMAC Authentication:**

```bash
# Generate a strong secret key
HMAC_SECRET="$(openssl rand -base64 32)"

# Enable HMAC authentication
HMAC_SECRET="$HMAC_SECRET" \
OUTPUTS=log \
./telhawk-proxy
```

**Client Integration:**

```javascript
// Automatic integration - include the HMAC script
<script src="/hmac.js"></script>

// Manual integration - get public key and generate HMAC
fetch('/hmac/public-key')
  .then(r => r.json())
  .then(data => {
    // Use data.public_key for HMAC generation
    // Send HMAC in X-TelHawk Proxy-HMAC header
  });
```

**HMAC + Auto-Injection:**
Auto-injected HTML includes both the tracking pixel AND the HMAC script:

```html
<script src="/hmac.js"></script>
<img src="/px.gif?e=pageview&auto=1&url=http%3A%2F%2Fwww.example.com%2Flander" width="1" height="1" style="display:none" alt="">
```

**Endpoints:**
- `GET /hmac.js` - JavaScript client for automatic HMAC generation
- `GET /hmac/public-key` - Public key and configuration for manual integration

### NDJSON log sink

* `LOG_PATH` (default `./events.ndjson`)
* `LOG_ROTATE_MB`, `LOG_BACKUPS`, `LOG_MAX_AGE_DAYS`

**Format**: newline‑delimited JSON, exactly the Event model per line.

### Kafka sink

* `KAFKA_BROKERS` (e.g., `localhost:9092,localhost:9093`)
* `KAFKA_TOPIC` (default `telhawk-proxy.events`)
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

TelHawk Proxy includes a built-in test mode that generates sample events for testing your sink configurations:

```bash
# Test locally with log sink only
TEST_MODE=true OUTPUTS=log ./telhawk-proxy

# Test all sinks (requires running Kafka/PostgreSQL)
TEST_MODE=true OUTPUTS=log,kafka,postgres ./telhawk-proxy

# Test specific configuration
TEST_MODE=true \
OUTPUTS=kafka \
KAFKA_BROKERS=localhost:9092 \
KAFKA_TOPIC=test.events \
./telhawk-proxy
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

### JavaScript/TypeScript Testing

The client-side tracking library (`js/`) has comprehensive test coverage:

```bash
cd js

# Install dependencies
npm ci

# Run tests with coverage
npm test -- --coverage

# Run tests in watch mode
npm test -- --watch

# Type checking
npx tsc --noEmit

# Linting
npm run lint

# Build the library
npm run build
```

**Test Coverage:**
- Unit tests: `js/test/unit/`
- Current coverage: ~84% on core modules
- Coverage reports: `js/coverage/lcov-report/index.html`

**CI/CD Integration:**
- Automated tests run on all PRs touching `js/` code
- TypeScript type checking enforced
- Coverage tracked and badged
- Build verification ensures library compiles

**Key test files:**
- `rand.test.ts` - Random number generation
- `batch.test.ts` - Event batching and queuing
- `sign.test.ts` - HMAC signing for authenticated requests
- `session.test.ts` - Session ID management with localStorage
- `webdriver.test.ts` - Bot/automation detection
- `plugins.test.ts` - Browser plugin detection

### Quick Testing

```bash
# Build and test locally with generated events
go build -o ./telhawk-proxy ./cmd/telhawk-proxy
TEST_MODE=true OUTPUTS=log ./telhawk-proxy

# Test with Docker Compose stack
./deploy/manage.sh up
./deploy/manage.sh test-mode

# Verify events in each sink
tail -f out/events.ndjson              # Log files
./deploy/manage.sh kafka-console       # Kafka messages  
./deploy/manage.sh psql                # PostgreSQL: SELECT * FROM events_json;
```

---

## 🔄 CI/CD & Quality Gates

TelHawk Proxy has comprehensive automated testing and quality checks via GitHub Actions:

### Go Backend Workflows

**Tests & Coverage** (`.github/workflows/test-coverage.yml`)
- Runs on all PRs and pushes to main
- Executes full test suite with race detector
- Tracks code coverage (82-100% across packages)
- Generates coverage badges and PR comments
- Uploads coverage artifacts

**Code Quality** (`.github/workflows/code-quality.yml`)
- Enforces cyclomatic complexity ≤15 for non-test code
- Fails build if complexity thresholds exceeded
- Posts detailed complexity reports on PRs
- Tracks top 10 most complex functions

### JavaScript/TypeScript Workflows

**JS Tests & Coverage** (`.github/workflows/js-test-coverage.yml`)
- Runs when `js/` code changes
- TypeScript type checking with `tsc --noEmit`
- ESLint linting (non-blocking)
- Jest unit tests with coverage reporting
- Coverage threshold: 60% (warning)
- Build verification with Rollup
- Generates JS coverage badges

### Security Scans

**SAST** - Static Application Security Testing (Semgrep)
**DAST** - Dynamic Application Security Testing (OWASP ZAP)
**Secret Scanning** - gitleaks for committed secrets
**Dependency Scanning** - Trivy for vulnerable dependencies
**Container Scanning** - SBOM generation and vulnerability scanning

### Quality Metrics

- **Go Coverage:** 82-100% across most packages
- **JS Coverage:** ~84% on tested modules (growing)
- **Complexity:** All non-test functions ≤15 cyclomatic complexity
- **Security:** Automated scanning on every commit

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
A: No, it would be nearly impossible to block with adblock.

---

## Credits

Built with ❤️ in Go. Inspired by years of shipping analytics/fraud pipelines in fintech & e‑commerce.

## License

This project is licensed under the Apache 2.0 License - see the [LICENSE.md](LICENSE.md) file for details.

---

## 🧑‍💻 Author

**Steven Horton**  
Software Engineer | Red Teaming | DevSecOps | Cloud Security  

- 💼 [LinkedIn](https://www.linkedin.com/in/steven-horton-66325520)  
- 🐙 [GitHub](https://github.com/shortontech)  