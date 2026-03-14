package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/pearcut"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	httpAddr := flag.String("http", "0.0.0.0:8080", "listen address (host:port)")
	dbPath := flag.String("db", "pearcut.db", "path to SQLite database")
	events := flag.String("events", "noop", "event publisher (noop)")
	flag.Parse()

	var publisher pearcut.EventPublisher
	switch *events {
	case "noop":
		publisher = pearcut.NoopPublisher{}
	default:
		slog.Error("❌ unknown events publisher", "events", *events)
		fmt.Fprintf(os.Stderr, "unknown --events value: %q\n", *events)
		os.Exit(1)
	}

	store, err := pearcut.NewSQLiteStore(*dbPath)
	if err != nil {
		slog.Error("❌ failed to open database", "path", *dbPath, "error", err)
		os.Exit(1)
	}
	slog.Info("✅ connected to database", "path", *dbPath)

	cached, err := pearcut.NewCachedStore(store)
	if err != nil {
		slog.Error("❌ failed to initialize cache", "error", err)
		os.Exit(1)
	}

	srv := pearcut.NewServer(*httpAddr, cached, publisher)

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	slog.Info("🚀 starting server", "addr", srv.Addr)
	if err := http.ListenAndServe(srv.Addr, mux); err != nil {
		slog.Error("❌ server failed", "error", err)
		os.Exit(1)
	}
}
