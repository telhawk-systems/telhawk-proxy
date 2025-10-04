BURN AFTER USING/READING

Prom metrics = Prometheus metrics.

Basically: your service exposes a /metrics HTTP endpoint (usually plaintext, Content-Type: text/plain; version=0.0.4) with a bunch of counters, gauges, and histograms in the Prometheus exposition format. Prometheus (or any scraper like VictoriaMetrics, Grafana Agent, etc.) hits that endpoint, ingests the numbers, and lets you graph/alert on them.

Concrete examples that would fit GoTrack:

# HELP gotrack_events_ingested_total Total events ingested
# TYPE gotrack_events_ingested_total counter
gotrack_events_ingested_total{sink="postgres"} 12345
gotrack_events_ingested_total{sink="kafka"} 234

# HELP gotrack_sink_errors_total Total errors writing to a sink
# TYPE gotrack_sink_errors_total counter
gotrack_sink_errors_total{sink="ndjson"} 5

# HELP gotrack_queue_depth Current depth of the internal event queue
# TYPE gotrack_queue_depth gauge
gotrack_queue_depth 42

# HELP gotrack_batch_flush_latency_seconds Latency of flushing a batch to sinks
# TYPE gotrack_batch_flush_latency_seconds histogram
gotrack_batch_flush_latency_seconds_bucket{le="0.05"} 120
gotrack_batch_flush_latency_seconds_bucket{le="0.1"} 250
...

Why it matters

Ops: You see when queues are backing up or sinks are failing.

Security telemetry: Lets you alert if events suddenly drop to zero (pixel blocked, DNS poisoned, proxy misbehaving).

Performance: Histograms show how long batch flushes are taking.

How to wire in Go

You’d add something like:

import "github.com/prometheus/client_golang/prometheus/promhttp"

http.Handle("/metrics", promhttp.Handler())


Then run Prometheus/Grafana, scrape http://gotrack:8080/metrics, and you’ve got dashboards.

Short answer: don’t expose /metrics to the world. Treat it like an internal admin surface and gate it at the network and/or auth layer.

Here are sane, concrete options—pick 1–2 that fit your deploy:

0) Ship “off by default” + separate listener

Feature flag: METRICS_ENABLED=false

Separate bind: METRICS_ADDR=127.0.0.1:9090 (or a dedicated port)

Optional TLS/mTLS knobs: METRICS_TLS_CERT, METRICS_TLS_KEY, METRICS_CLIENT_CA

In Go:

mux := http.NewServeMux()
mux.Handle("/metrics", promhttp.Handler())

srv := &http.Server{
    Addr:    os.Getenv("METRICS_ADDR"), // e.g. 127.0.0.1:9090
    Handler: mux,
}
log.Fatal(srv.ListenAndServe())

1) Network-only (simple + robust)

Docker: don’t publish the port; let Prometheus scrape over the bridge network.

Kubernetes: ClusterIP Service scoped to the namespace; no Ingress. Use a ServiceMonitor so only in-cluster Prom scrapes it.

Lock it down with NetworkPolicies (allow from Prometheus Pod/Namespace only).

2) Reverse proxy auth in front of /metrics

Good when you must cross trust boundaries.

Basic auth (fastest):

Nginx/Traefik in front of GoTrack’s /metrics

Prometheus uses basic_auth in scrape_config.

Bearer/OAuth2:

Sidecar oauth2-proxy / IdP in front; Prometheus uses a static bearer_token secret.

mTLS (most stringent):

Terminate TLS at the app or proxy; require client cert signed by your CA.

Minimal examples

Nginx (basic auth)

location /metrics {
    satisfy all;
    auth_basic "metrics";
    auth_basic_user_file /etc/nginx/htpasswd; # create with `htpasswd`
    proxy_pass http://gotrack-metrics:9090/metrics;
    proxy_set_header Host $host;
}


Prometheus scrape_config (basic auth)

scrape_configs:
  - job_name: gotrack
    metrics_path: /metrics
    basic_auth:
      username: prom
      password: ${GOTRACK_METRICS_PASSWORD}
    static_configs:
      - targets: ["gotrack-metrics.local:443"]
    scheme: https


mTLS at the app (Go)

tlsCfg := &tls.Config{
    ClientCAs:  mustLoadCertPool(os.Getenv("METRICS_CLIENT_CA")),
    ClientAuth: tls.RequireAndVerifyClientCert,
    MinVersion: tls.VersionTLS12,
}
srv := &http.Server{Addr: ":9443", Handler: mux, TLSConfig: tlsCfg}
log.Fatal(srv.ListenAndServeTLS(os.Getenv("METRICS_TLS_CERT"), os.Getenv("METRICS_TLS_KEY")))


Kubernetes (Operator)

Expose metrics on :9090 via ClusterIP Service.

Add a ServiceMonitor selecting that service (no Ingress).

Add a NetworkPolicy allowing ingress only from the Prometheus namespace/labels.

If you want RBAC-style auth, put kube-rbac-proxy in front of the metrics port and grant Prometheus’s ServiceAccount the needed Role.

3) Hardening notes (worth doing)

Never include secrets or PII in metric names/labels.

Keep label cardinality bounded (e.g., don’t label by event_id).

Rate-limit /metrics if it’s ever reachable across zones.

If using TLS, automate cert rotation (cert-manager on K8s).

Log and alert on scrape failures (a sudden drop to zero events or scrape errors is itself a signal).

If you want, I’ll spec env vars + a tiny metrics.go that:

spins a separate listener,

supports plain/TLS/mTLS,

registers default Go + process collectors,

and adds a couple of custom counters/histograms to get you started.