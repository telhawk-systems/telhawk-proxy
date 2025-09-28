# Project Structure — GoTrack Pixel

This document explains the folder and file layout of the GoTrack Pixel project. It’s designed to be idiomatic Go, modular, and production‑ready, while keeping the codebase easy to navigate.

---

## Top‑level

```
.
├── cmd/                # Entrypoints (binaries)
├── internal/           # Core application logic (not imported externally)
├── pkg/                # Reusable libraries/utilities
├── deploy/             # Deployment manifests (local, k8s, etc.)
├── test/               # Integration/system tests
├── README.md           # Overview & usage docs
├── PROJECT_STRUCTURE.md# (this file)
├── go.mod / go.sum     # Go modules
└── .gitignore
```

---

## `cmd/`

Holds the main entrypoint(s) of the application. For now, only one binary:

```
cmd/gotrack/
└── main.go   # bootstraps config, HTTP server, sinks
```

---

## `internal/`

Private packages that make up the core of the tracking pixel.

### `internal/http/`

HTTP server and request handlers.

* `server.go` → starts the HTTP server, routing, lifecycle.
* `handlers.go` → `/px.gif`, `/collect`, `/healthz`, `/readyz`, `/metrics`.
* `middleware.go` → request logging, recovery, CORS, DNT enforcement.

### `internal/sink/`

Implements pluggable data sinks.

* `sink.go` → defines the `Sink` interface and sink registry.
* `logsink.go` → NDJSON log sink.
* `kafkasink.go` → Kafka producer sink.
* `pgsink.go` → Postgres JSONB sink.

### `internal/event/`

Event model and enrichment logic.

* `event.go` → event struct, validation, JSON marshalling.
* `enrich.go` → adds metadata (event_id, IP hash, UA parsing, GeoIP).

---

## `pkg/`

Reusable helpers that could stand alone in other projects.

### `pkg/config/`

* `config.go` → loads environment variables into a typed config struct.

---

## `deploy/`

Deployment manifests, scripts, and infra config.

### `deploy/local/`

* `docker-compose.yml` → local dev environment with Kafka, Postgres, Zookeeper.
* `init.sql` → initializes Postgres schema (`events_json` table, indexes).

Future expansion:

* `deploy/k8s/` for Helm charts or raw manifests.
* `deploy/terraform/` for cloud infra provisioning.

---

## `test/`

High‑level tests that exercise real sinks & network.

* `integration_test.go` → verifies end‑to‑end ingestion across sinks.

Unit tests for individual packages live next to their Go files.

---

## Planned evolution

* Add more sinks (Redis, S3/Parquet, RabbitMQ).
* Add observability (`pkg/otel`, dashboards).
* Add Helm chart under `deploy/k8s/`.
* Expand test suite with load tests and fuzzers.

---

## Summary

* **`cmd/`** → entrypoints
* **`internal/`** → application logic (HTTP, sinks, events)
* **`pkg/`** → reusable utilities (config, logging, etc.)
* **`deploy/`** → infra + manifests
* **`test/`** → integration/system tests

This structure balances Go’s simplicity with the needs of a production data pipeline.
