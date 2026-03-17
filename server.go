package pearcut

import (
	"encoding/json"
	"net/http"
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

func NewServer(addr string, store Store, publisher EventPublisher) *Server {
	return &Server{
		Addr:   addr,
		store:  store,
		engine: NewEngine(store, publisher),
	}
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", healthHandler)
	mux.HandleFunc("POST /api/v1/assign", s.assignHandler)
	mux.HandleFunc("POST /api/v1/assign/bulk", s.bulkAssignHandler)

	mux.HandleFunc("GET /admin/v1/experiments", s.listExperiments)
	mux.HandleFunc("GET /admin/v1/experiments/{slug}", s.getExperiment)
	mux.HandleFunc("POST /admin/v1/experiments", s.createExperiment)
	mux.HandleFunc("PUT /admin/v1/experiments/{slug}", s.updateExperiment)
	mux.HandleFunc("DELETE /admin/v1/experiments/{slug}", s.deleteExperiment)
}
