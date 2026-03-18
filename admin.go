package pearcut

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
)

func isValidStatus(s ExperimentStatus) bool {
	switch s {
	case StatusDraft, StatusRunning, StatusPaused, StatusStopped:
		return true
	}
	return false
}

type ListExperimentsResponse struct {
	Data       []Experiment `json:"data"`
	Page       int          `json:"page"`
	PerPage    int          `json:"per_page"`
	Total      int          `json:"total"`
	TotalPages int          `json:"total_pages"`
}

func (s *Server) listExperiments(w http.ResponseWriter, r *http.Request) {
	var filter ExperimentFilter
	var opts ListOptions
	q := r.URL.Query()

	if raw := q.Get("status"); raw != "" {
		status := ExperimentStatus(raw)
		if !isValidStatus(status) {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid status"})
			return
		}
		filter.Status = &status
	}

	filter.Tags = q["tags"]

	page, perPage := 1, 20
	if raw := q.Get("page"); raw != "" {
		p, err := strconv.Atoi(raw)
		if err != nil || p < 1 {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid page parameter"})
			return
		}
		page = p
	}
	if raw := q.Get("per_page"); raw != "" {
		pp, err := strconv.Atoi(raw)
		if err != nil || pp < 1 || pp > 100 {
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: "invalid per_page parameter (1-100)"})
			return
		}
		perPage = pp
	}

	opts.Page = page
	opts.PerPage = perPage

	result, err := s.experimentStore.List(filter, opts)
	if err != nil {
		slog.Error("failed to list experiments", "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}

	totalPages := 0
	if result.Total > 0 {
		totalPages = (result.Total + perPage - 1) / perPage
	}

	writeJSON(w, http.StatusOK, ListExperimentsResponse{
		Data:       result.Experiments,
		Page:       page,
		PerPage:    perPage,
		Total:      result.Total,
		TotalPages: totalPages,
	})
}

func (s *Server) getExperiment(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	exp, err := s.experimentStore.Get(slug)
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

	if err := s.experimentStore.Create(exp); err != nil {
		switch {
		case errors.Is(err, ErrExperimentExists):
			writeJSON(w, http.StatusConflict, ErrorResponse{Error: "experiment already exists"})
		default:
			slog.Error("failed to create experiment", "slug", exp.Slug, "error", err)
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		}
		return
	}

	created, err := s.experimentStore.Get(exp.Slug)
	if err != nil {
		slog.Error("failed to read back experiment", "slug", exp.Slug, "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}

	s.assignStore.Set(created)
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

	if err := s.experimentStore.Update(exp); err != nil {
		switch {
		case errors.Is(err, ErrExperimentNotFound):
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "experiment not found"})
		default:
			slog.Error("failed to update experiment", "slug", slug, "error", err)
			writeJSON(w, http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		}
		return
	}

	updated, err := s.experimentStore.Get(slug)
	if err != nil {
		slog.Error("failed to read back experiment", "slug", slug, "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}

	s.assignStore.Set(updated)
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteExperiment(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	if err := s.experimentStore.Delete(slug); err != nil {
		if errors.Is(err, ErrExperimentNotFound) {
			writeJSON(w, http.StatusNotFound, ErrorResponse{Error: "experiment not found"})
			return
		}
		slog.Error("failed to delete experiment", "slug", slug, "error", err)
		writeJSON(w, http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
		return
	}

	s.assignStore.Delete(slug)
	w.WriteHeader(http.StatusNoContent)
}
