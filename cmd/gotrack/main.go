package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shortontech/gotrack/internal/event"
	httpx "github.com/shortontech/gotrack/internal/http"
	"github.com/shortontech/gotrack/internal/metrics"
	"github.com/shortontech/gotrack/internal/sink"
	"github.com/shortontech/gotrack/pkg/config"
)

func main() {
	// Parse command line flags
	var (
		healthCheck = flag.Bool("healthcheck", false, "Perform health check and exit")
		healthHost  = flag.String("health-host", "localhost", "Host for health check")
		healthPort  = flag.String("health-port", "19890", "Port for health check")
	)
	flag.Parse()

	// Handle health check mode
	if *healthCheck {
		if err := performHealthCheck(*healthHost, *healthPort); err != nil {
			log.Printf("Health check failed: %v", err)
			os.Exit(1)
		}
		log.Println("Health check passed")
		os.Exit(0)
	}

	cfg := config.Load()

	// Initialize metrics
	appMetrics := metrics.InitMetrics()
	metricsConfig := metrics.Config{
		Enabled:     cfg.MetricsEnabled,
		Addr:        cfg.MetricsAddr,
		TLSCert:     cfg.MetricsTLSCert,
		TLSKey:      cfg.MetricsTLSKey,
		ClientCA:    cfg.MetricsClientCA,
		RequireTLS:  cfg.MetricsRequireTLS,
		RequireAuth: false, // Not implemented yet
	}
	metricsServer := metrics.NewServer(metricsConfig)

	// start sinks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sinks := initializeSinks(ctx, cfg.Outputs)
	if len(sinks) == 0 {
		log.Fatal("no valid sinks configured")
	}

	hmacAuth := initializeHMACAuth(cfg)

	env := httpx.Env{
		Cfg:      cfg,
		HMACAuth: hmacAuth,
		Metrics:  appMetrics,
		Emit:     createEmitFunc(sinks, appMetrics),
	}

	// Start metrics server
	if err := metricsServer.Start(ctx); err != nil {
		log.Printf("failed to start metrics server: %v", err)
	}

	// Run test mode if enabled (generate test events)
	if cfg.TestMode {
		go func() {
			// Wait a moment for sinks to be fully initialized
			time.Sleep(2 * time.Second)
			runTestMode(env.Emit)
		}()
	}

	srv := startHTTPServer(cfg, env)
	waitForShutdown(srv, metricsServer, sinks)
}

func initializeSinks(ctx context.Context, outputs []string) []sink.Sink {
	var sinks []sink.Sink

	for _, output := range outputs {
		switch output {
		case "log":
			logSink := sink.NewLogSink()
			if err := logSink.Start(ctx); err != nil {
				log.Fatalf("failed to start log sink: %v", err)
			}
			sinks = append(sinks, logSink)
			log.Println("log sink started")

		case "kafka":
			kafkaSink := sink.NewKafkaSinkFromEnv()
			if err := kafkaSink.Start(ctx); err != nil {
				log.Fatalf("failed to start kafka sink: %v", err)
			}
			sinks = append(sinks, kafkaSink)
			log.Println("kafka sink started")

		case "postgres":
			pgSink := sink.NewPGSinkFromEnv()
			if err := pgSink.Start(ctx); err != nil {
				log.Fatalf("failed to start postgres sink: %v", err)
			}
			sinks = append(sinks, pgSink)
			log.Println("postgres sink started")

		default:
			log.Printf("unknown output type: %s, skipping", output)
		}
	}

	return sinks
}

func initializeHMACAuth(cfg config.Config) *httpx.HMACAuth {
	var hmacAuth *httpx.HMACAuth
	if cfg.HMACSecret != "" {
		hmacAuth = httpx.NewHMACAuth(cfg.HMACSecret, cfg.HMACPublicKey, cfg.RequireHMAC)
		if cfg.RequireHMAC {
			log.Printf("HMAC authentication enabled and required for /collect endpoint")
		} else {
			log.Printf("HMAC authentication configured but not required")
		}
		log.Printf("HMAC client script available at /hmac.js")
		log.Printf("HMAC public key available at /hmac/public-key")
	}
	return hmacAuth
}

func createEmitFunc(sinks []sink.Sink, appMetrics *metrics.Metrics) func(event.Event) {
	return func(ev event.Event) {
		// Send event to all configured sinks
		for _, s := range sinks {
			if err := s.Enqueue(ev); err != nil {
				log.Printf("failed to enqueue event to sink: %v", err)
				// Track sink errors in metrics
				appMetrics.IncrementSinkErrors(s.Name(), "enqueue_error")
			} else {
				// Track successful ingestion
				appMetrics.IncrementEventsIngested(s.Name())
			}
		}
	}
}

func startHTTPServer(cfg config.Config, env httpx.Env) *http.Server {
	srv := &http.Server{
		Addr:              cfg.ServerAddr,
		Handler:           httpx.NewMux(env),
		ReadHeaderTimeout: 10 * time.Second, // Prevent Slowloris attacks
	}

	go func() {
		if cfg.EnableHTTPS {
			log.Printf("gotrack listening on %s (HTTPS)", cfg.ServerAddr)
			if err := srv.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTPS server error: %v", err)
			}
		} else {
			log.Printf("gotrack listening on %s (HTTP)", cfg.ServerAddr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("HTTP server error: %v", err)
			}
		}
	}()

	return srv
}

func waitForShutdown(srv *http.Server, metricsServer *metrics.Server, sinks []sink.Sink) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)

	// Shutdown metrics server
	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("error shutting down metrics server: %v", err)
	}

	// Close all sinks
	for _, s := range sinks {
		if err := s.Close(); err != nil {
			log.Printf("error closing sink: %v", err)
		}
	}

	log.Println("shutdown complete")
}

// performHealthCheck performs a health check against the specified host and port
func performHealthCheck(host, port string) error {
	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	// Construct health check URL
	scheme := "http"
	url := fmt.Sprintf("%s://%s/healthz", scheme, net.JoinHostPort(host, port))

	// Perform health check request
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to health endpoint: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	// Read and verify response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read health check response: %w", err)
	}

	// Verify expected response
	if string(body) != "ok" {
		return fmt.Errorf("unexpected health check response: %s", string(body))
	}

	return nil
}
