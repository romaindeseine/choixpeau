package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pearcut"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	httpAddr := flag.String("http", "0.0.0.0:8080", "listen address (host:port)")
	dbPath := flag.String("db", "pearcut.db", "path to SQLite database")
	events := flag.String("events", "noop", "event publisher (noop, stdout)")
	flag.Parse()

	var publisher pearcut.EventPublisher
	switch *events {
	case "noop":
		publisher = pearcut.NoopPublisher{}
	case "stdout":
		publisher = pearcut.NewStdoutPublisher(os.Stdout)
	default:
		slog.Error("❌ unknown events publisher", "events", *events)
		fmt.Fprintf(os.Stderr, "unknown --events value: %q\n", *events)
		os.Exit(1)
	}

	experimentStore, err := pearcut.NewSQLiteStore(*dbPath)
	if err != nil {
		slog.Error("❌ failed to open database", "path", *dbPath, "error", err)
		os.Exit(1)
	}
	slog.Info("✅ connected to database", "path", *dbPath)

	result, err := experimentStore.List(pearcut.ExperimentFilter{}, pearcut.ListOptions{})
	if err != nil {
		slog.Error("❌ failed to load experiments into memory", "error", err)
		os.Exit(1)
	}
	assignStore := pearcut.NewMemStore(result.Experiments)

	async := pearcut.NewAsyncPublisher(publisher)
	engine := pearcut.NewEngine(assignStore, async)

	srv := pearcut.NewServer(*httpAddr, experimentStore, assignStore, engine)

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	httpServer := &http.Server{Addr: srv.Addr, Handler: mux}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		slog.Info("🚀 starting server", "addr", srv.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("❌ server failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("⚠️ shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("❌ http shutdown failed", "error", err)
	}

	if err := async.Close(); err != nil {
		slog.Error("❌ publisher close failed", "error", err)
	}
}
