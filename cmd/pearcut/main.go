package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/pearcut"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	addr := fmt.Sprintf(":%s", os.Getenv("PORT"))
	if addr == ":" {
		addr = ":8080"
		slog.Warn("⚠️ PORT not set, using default", "port", "8080")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "pearcut.db"
	}

	store, err := pearcut.NewSQLiteStore(dbPath)
	if err != nil {
		slog.Error("❌ failed to open database", "path", dbPath, "error", err)
		os.Exit(1)
	}
	slog.Info("✅ connected to database", "path", dbPath)

	cached, err := pearcut.NewCachedStore(store)
	if err != nil {
		slog.Error("❌ failed to initialize cache", "error", err)
		os.Exit(1)
	}

	srv := pearcut.NewServer(addr, cached)

	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	slog.Info("🚀 starting server", "addr", srv.Addr)
	if err := http.ListenAndServe(srv.Addr, mux); err != nil {
		slog.Error("❌ server failed", "error", err)
		os.Exit(1)
	}
}
