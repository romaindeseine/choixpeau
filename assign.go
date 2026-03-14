package pearcut

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

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
		UserID:     userID,
	})
}

type bulkAssignRequest struct {
	UserID      string   `json:"user_id"`
	Experiments []string `json:"experiments,omitempty"`
}

type bulkAssignResponse struct {
	UserID      string            `json:"user_id"`
	Assignments map[string]string `json:"assignments"`
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

	result := make(map[string]string, len(assignments))
	for _, a := range assignments {
		result[a.Experiment] = a.Variant
	}

	writeJSON(w, http.StatusOK, bulkAssignResponse{UserID: req.UserID, Assignments: result})
}
