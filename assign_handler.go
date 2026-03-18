package pearcut

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

type assignRequest struct {
	Experiment string            `json:"experiment"`
	UserID     string            `json:"user_id"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

type assignResponse struct {
	Experiment string `json:"experiment"`
	Variant    string `json:"variant"`
	UserID     string `json:"user_id"`
}

func (s *Server) assignHandler(w http.ResponseWriter, r *http.Request) {
	var req assignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json body"})
		return
	}

	if req.Experiment == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing required field: experiment"})
		return
	}

	if req.UserID == "" {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "missing required field: user_id"})
		return
	}

	assignment, err := s.engine.Assign(r.Context(), req.UserID, req.Experiment, req.Attributes)
	if err != nil {
		switch {
		case errors.Is(err, ErrExperimentNotFound):
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "experiment not found"})
		case errors.Is(err, ErrExperimentNotRunning):
			writeJSON(w, http.StatusConflict, ErrorResponse{Error: "experiment not running"})
		case errors.Is(err, ErrUserNotTargeted), errors.Is(err, ErrUserExcludedByTraffic):
			w.WriteHeader(http.StatusNoContent)
		default:
			slog.Error("assignment failed", "experiment", req.Experiment, "user_id", req.UserID, "error", err)
			writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		}
		return
	}

	writeJSON(w, http.StatusOK, assignResponse{
		Experiment: assignment.Experiment,
		Variant:    assignment.Variant,
		UserID:     req.UserID,
	})
}

type bulkAssignRequest struct {
	UserID      string            `json:"user_id"`
	Experiments []string          `json:"experiments,omitempty"`
	Attributes  map[string]string `json:"attributes,omitempty"`
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

	assignments, err := s.engine.BulkAssign(r.Context(), req.UserID, req.Experiments, req.Attributes)
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
