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
	logSink := sink.NewLogSink()
	_ = logSink.Start(ctx)

	env := httpx.Env{
		Cfg: cfg,
		Emit: func(ev event.Event) {
			_ = logSink.Enqueue(ev)
		},
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

	shutdownCtx, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()
	_ = srv.Shutdown(shutdownCtx)
	_ = logSink.Close()
}
