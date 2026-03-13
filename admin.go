package choixpeau

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
)

func isValidStatus(s ExperimentStatus) bool {
	switch s {
	case StatusDraft, StatusRunning, StatusPaused, StatusStopped:
		return true
	}
	return false
}

func (s *Server) listExperiments(w http.ResponseWriter, r *http.Request) {
	var filter ExperimentFilter

	if raw := r.URL.Query().Get("status"); raw != "" {
		status := ExperimentStatus(raw)
		if !isValidStatus(status) {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid status"})
			return
		}
		filter.Status = &status
	}

	experiments, err := s.store.List(filter)
	if err != nil {
		slog.Error("failed to list experiments", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, experiments)
}

func (s *Server) getExperiment(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	exp, err := s.store.Get(slug)
	if err != nil {
		if errors.Is(err, ErrExperimentNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "experiment not found"})
			return
		}
		slog.Error("failed to get experiment", "slug", slug, "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, exp)
}

func (s *Server) createExperiment(w http.ResponseWriter, r *http.Request) {
	var exp Experiment
	if err := json.NewDecoder(r.Body).Decode(&exp); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json"})
		return
	}

	if err := s.store.Create(exp); err != nil {
		switch {
		case errors.Is(err, ErrExperimentExists):
			writeJSON(w, http.StatusConflict, ErrorResponse{Error: "experiment already exists"})
		default:
			slog.Error("failed to create experiment", "slug", exp.Slug, "error", err)
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		}
		return
	}

	created, err := s.store.Get(exp.Slug)
	if err != nil {
		slog.Error("failed to read back experiment", "slug", exp.Slug, "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) updateExperiment(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	var exp Experiment
	if err := json.NewDecoder(r.Body).Decode(&exp); err != nil {
		writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid json"})
		return
	}
	exp.Slug = slug

	if err := s.store.Update(exp); err != nil {
		switch {
		case errors.Is(err, ErrExperimentNotFound):
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "experiment not found"})
		default:
			slog.Error("failed to update experiment", "slug", slug, "error", err)
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		}
		return
	}

	updated, err := s.store.Get(slug)
	if err != nil {
		slog.Error("failed to read back experiment", "slug", slug, "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteExperiment(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	if err := s.store.Delete(slug); err != nil {
		if errors.Is(err, ErrExperimentNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "experiment not found"})
			return
		}
		slog.Error("failed to delete experiment", "slug", slug, "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
