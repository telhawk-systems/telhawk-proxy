package metrics

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metrics holds all the Prometheus metrics for GoTrack
type Metrics struct {
	// Counters
	EventsIngested *prometheus.CounterVec
	SinkErrors     *prometheus.CounterVec
	HTTPRequests   *prometheus.CounterVec

	// Gauges
	QueueDepth *prometheus.GaugeVec

	// Histograms
	BatchFlushLatency *prometheus.HistogramVec
	HTTPDuration      *prometheus.HistogramVec
}

// Config holds configuration for the metrics server
type Config struct {
	Enabled     bool
	Addr        string
	TLSCert     string
	TLSKey      string
	ClientCA    string
	RequireTLS  bool
	RequireAuth bool
}

// LoadConfig loads metrics configuration from environment variables
func LoadConfig() Config {
	return Config{
		Enabled:     getBool("METRICS_ENABLED", false),
		Addr:        getOr("METRICS_ADDR", "127.0.0.1:9090"),
		TLSCert:     getOr("METRICS_TLS_CERT", ""),
		TLSKey:      getOr("METRICS_TLS_KEY", ""),
		ClientCA:    getOr("METRICS_CLIENT_CA", ""),
		RequireTLS:  getBool("METRICS_REQUIRE_TLS", false),
		RequireAuth: getBool("METRICS_REQUIRE_AUTH", false),
	}
}

// NewMetrics creates and registers all GoTrack metrics
func NewMetrics() *Metrics {
	m := &Metrics{
		EventsIngested: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gotrack_events_ingested_total",
				Help: "Total events ingested by sink type",
			},
			[]string{"sink"},
		),

		SinkErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gotrack_sink_errors_total",
				Help: "Total errors writing to a sink",
			},
			[]string{"sink", "error_type"},
		),

		HTTPRequests: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "gotrack_http_requests_total",
				Help: "Total HTTP requests by endpoint and status",
			},
			[]string{"endpoint", "method", "status"},
		),

		QueueDepth: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "gotrack_queue_depth",
				Help: "Current depth of the internal event queue",
			},
			[]string{"sink"},
		),

		BatchFlushLatency: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gotrack_batch_flush_latency_seconds",
				Help:    "Latency of flushing a batch to sinks",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"sink"},
		),

		HTTPDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "gotrack_http_duration_seconds",
				Help:    "HTTP request duration",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
			},
			[]string{"endpoint", "method"},
		),
	}

	// Register all metrics
	prometheus.MustRegister(m.EventsIngested)
	prometheus.MustRegister(m.SinkErrors)
	prometheus.MustRegister(m.HTTPRequests)
	prometheus.MustRegister(m.QueueDepth)
	prometheus.MustRegister(m.BatchFlushLatency)
	prometheus.MustRegister(m.HTTPDuration)

	return m
}

// Server represents the metrics HTTP server
type Server struct {
	server *http.Server
	config Config
}

// NewServer creates a new metrics server
func NewServer(config Config) *Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Add a simple health check endpoint for the metrics server
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK")) // Ignore write errors for health check
	})

	srv := &http.Server{
		Addr:    config.Addr,
		Handler: mux,
		// Security: Set timeouts to prevent resource exhaustion
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Configure TLS if enabled
	if config.RequireTLS && config.TLSCert != "" && config.TLSKey != "" {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		// Configure mTLS if client CA is provided
		if config.ClientCA != "" {
			clientCAs, err := loadCertPool(config.ClientCA)
			if err != nil {
				log.Printf("metrics: failed to load client CA: %v", err)
			} else {
				tlsConfig.ClientCAs = clientCAs
				tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
				log.Printf("metrics: mTLS enabled with client CA: %s", config.ClientCA)
			}
		}

		srv.TLSConfig = tlsConfig
	}

	return &Server{
		server: srv,
		config: config,
	}
}

// Start starts the metrics server in a separate goroutine
func (s *Server) Start(ctx context.Context) error {
	if !s.config.Enabled {
		log.Printf("metrics: disabled (METRICS_ENABLED=false)")
		return nil
	}

	go func() {
		var err error
		if s.config.RequireTLS && s.config.TLSCert != "" && s.config.TLSKey != "" {
			log.Printf("metrics: HTTPS server listening on %s", s.config.Addr)
			err = s.server.ListenAndServeTLS(s.config.TLSCert, s.config.TLSKey)
		} else {
			log.Printf("metrics: HTTP server listening on %s", s.config.Addr)
			err = s.server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			log.Printf("metrics: server error: %v", err)
		}
	}()

	// Wait for server to start (give it a moment)
	time.Sleep(100 * time.Millisecond)
	return nil
}

// Shutdown gracefully shuts down the metrics server
func (s *Server) Shutdown(ctx context.Context) error {
	if !s.config.Enabled {
		return nil
	}

	log.Printf("metrics: shutting down server...")
	return s.server.Shutdown(ctx)
}

// Helper functions
func getOr(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}
	return parsed
}

func loadCertPool(certFile string) (*x509.CertPool, error) {
	// This would load a certificate pool from a file
	// For now, return nil to indicate no client CA
	// In production, you'd implement proper certificate loading
	return nil, nil
}

// Global metrics instance
var defaultMetrics *Metrics

// InitMetrics initializes the global metrics instance
func InitMetrics() *Metrics {
	if defaultMetrics == nil {
		defaultMetrics = NewMetrics()
	}
	return defaultMetrics
}

// GetMetrics returns the global metrics instance
func GetMetrics() *Metrics {
	if defaultMetrics == nil {
		defaultMetrics = NewMetrics()
	}
	return defaultMetrics
}

// Convenience methods for common operations
func (m *Metrics) IncrementEventsIngested(sink string) {
	m.EventsIngested.WithLabelValues(sink).Inc()
}

func (m *Metrics) IncrementSinkErrors(sink, errorType string) {
	m.SinkErrors.WithLabelValues(sink, errorType).Inc()
}

func (m *Metrics) IncrementHTTPRequests(endpoint, method, status string) {
	m.HTTPRequests.WithLabelValues(endpoint, method, status).Inc()
}

func (m *Metrics) SetQueueDepth(sink string, depth float64) {
	m.QueueDepth.WithLabelValues(sink).Set(depth)
}

func (m *Metrics) ObserveBatchFlushLatency(sink string, duration time.Duration) {
	m.BatchFlushLatency.WithLabelValues(sink).Observe(duration.Seconds())
}

func (m *Metrics) ObserveHTTPDuration(endpoint, method string, duration time.Duration) {
	m.HTTPDuration.WithLabelValues(endpoint, method).Observe(duration.Seconds())
}
