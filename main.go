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
	store  Store
	engine Engine
}

func newServer(addr string, store Store) *Server {
	return &Server{
		Addr:   addr,
		store:  store,
		engine: NewEngine(store),
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type assignResponse struct {
	Experiment string `json:"experiment"`
	Variant    string `json:"variant"`
	UserID     string `json:"user_id"`
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

	writeJSON(w, http.StatusOK, assignResponse{
		Experiment: assignment.Experiment,
		Variant:    assignment.Variant,
		UserID:     assignment.UserID,
	})
}

type bulkAssignRequest struct {
	UserID      string   `json:"user_id"`
	Experiments []string `json:"experiments,omitempty"`
}

type bulkAssignResponse struct {
	UserID      string       `json:"user_id"`
	Assignments []Assignment `json:"assignments"`
}

func (s *Server) bulkAssignHandler(w http.ResponseWriter, r *http.Request) {
	var req bulkAssignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json body"})
		return
	}

	if req.UserID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing required field: user_id"})
		return
	}

	assignments, err := s.engine.BulkAssign(req.UserID, req.Experiments)
	if err != nil {
		slog.Error("bulk assignment failed", "user_id", req.UserID, "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, bulkAssignResponse{UserID: req.UserID, Assignments: assignments})
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	addr := fmt.Sprintf(":%s", os.Getenv("PORT"))
	if addr == ":" {
		addr = ":8080"
		slog.Warn("⚠️ PORT not set, using default", "port", "8080")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "choixpeau.db"
	}

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		slog.Error("❌ failed to open database", "path", dbPath, "error", err)
		os.Exit(1)
	}
	slog.Info("✅ connected to database", "path", dbPath)

	cached, err := NewCachedStore(store)
	if err != nil {
		slog.Error("❌ failed to initialize cache", "error", err)
		os.Exit(1)
	}

	server := newServer(addr, cached)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("GET /api/v1/assign", server.assignHandler)
	mux.HandleFunc("POST /api/v1/assign/bulk", server.bulkAssignHandler)

	mux.HandleFunc("GET /admin/v1/experiments", server.listExperiments)
	mux.HandleFunc("GET /admin/v1/experiments/{slug}", server.getExperiment)
	mux.HandleFunc("POST /admin/v1/experiments", server.createExperiment)
	mux.HandleFunc("PUT /admin/v1/experiments/{slug}", server.updateExperiment)
	mux.HandleFunc("DELETE /admin/v1/experiments/{slug}", server.deleteExperiment)

	slog.Info("🚀 starting server", "addr", server.Addr)
	if err := http.ListenAndServe(server.Addr, mux); err != nil {
		slog.Error("❌ server failed", "error", err)
		os.Exit(1)
	}
}
