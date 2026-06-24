// Command paper-inator is the unified executable for the publication aggregator.
// A single binary runs the background ingestion worker, serves the REST API, and
// serves the embedded web frontend, so deployment is "copy the binary and run".
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"paperinator/src/api"
	"paperinator/src/serviceWorker"
	"paperinator/src/shared/config"
	"paperinator/src/shared/store"
)

func main() {
	cfg := config.Load()

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer st.Close()
	log.Printf("database ready at %s", cfg.DBPath)

	// Cancel everything on SIGINT/SIGTERM for a clean shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Background ingestion worker.
	worker := serviceWorker.New(st, cfg.FetchInterval)
	go worker.Run(ctx)

	// HTTP server (API + embedded frontend).
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           api.NewServer(st, worker),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
		os.Exit(1)
	}
}
