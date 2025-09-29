package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"revinar.io/go.track/internal/event"
	httpx "revinar.io/go.track/internal/http"
	"revinar.io/go.track/internal/sink"
	"revinar.io/go.track/pkg/config"
)

func main() {
	cfg := config.Load()

	// start sinks
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	var sinks []sink.Sink
	
	// Initialize sinks based on configuration
	for _, output := range cfg.Outputs {
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
	
	if len(sinks) == 0 {
		log.Fatal("no valid sinks configured")
	}

	env := httpx.Env{
		Cfg: cfg,
		Emit: func(ev event.Event) {
			// Send event to all configured sinks
			for _, s := range sinks {
				if err := s.Enqueue(ev); err != nil {
					log.Printf("failed to enqueue event to sink: %v", err)
				}
			}
		},
	}

	// Run test mode if enabled (generate test events)
	if cfg.TestMode {
		go func() {
			// Wait a moment for sinks to be fully initialized
			time.Sleep(2 * time.Second)
			runTestMode(env.Emit)
		}()
	}

	srv := &http.Server{
		Addr:    cfg.ServerAddr,
		Handler: httpx.NewMux(env),
	}

	go func() {
		log.Printf("gotrack listening on %s", cfg.ServerAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("shutting down...")
	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	_ = srv.Shutdown(shutdownCtx)
	
	// Close all sinks
	for _, s := range sinks {
		if err := s.Close(); err != nil {
			log.Printf("error closing sink: %v", err)
		}
	}
	
	log.Println("shutdown complete")
}
