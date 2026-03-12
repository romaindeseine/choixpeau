package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

type HealthResponse struct {
	Status string `json:"status"`
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

type Server struct {
	Addr   string
	store  ReadStore
	engine Engine
}

func newServer(addr string, store ReadStore) *Server {
	return &Server{
		Addr:   addr,
		store:  store,
		engine: NewEngine(store),
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (s *Server) assignHandler(w http.ResponseWriter, r *http.Request) {
	experimentSlug := r.URL.Query().Get("experiment")
	if experimentSlug == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing required parameter: experiment"})
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing required parameter: user_id"})
		return
	}

	assignment, err := s.engine.Assign(experimentSlug, userID)
	if err != nil {
		switch {
		case errors.Is(err, ErrExperimentNotFound):
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "experiment not found"})
		case errors.Is(err, ErrExperimentNotRunning):
			writeJSON(w, http.StatusConflict, ErrorResponse{Error: "experiment not running"})
		default:
			slog.Error("assignment failed", "experiment", experimentSlug, "user_id", userID, "error", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, assignment)
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	addr := fmt.Sprintf(":%s", os.Getenv("PORT"))
	if addr == ":" {
		addr = ":8080"
		slog.Warn("⚠️ PORT not set, using default", "port", "8080")
	}

	server := newServer(addr, newMemStore())

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/v1/assign", server.assignHandler)

	slog.Info("🚀 starting server", "addr", server.Addr)
	if err := http.ListenAndServe(server.Addr, mux); err != nil {
		slog.Error("❌ server failed", "error", err)
		os.Exit(1)
	}
}
