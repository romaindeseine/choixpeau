package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var body jsonBody
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %s", body["status"])
	}
}

type jsonBody map[string]string

type mockEngine struct {
	assignment Assignment
	err        error
}

func (m *mockEngine) Assign(experimentSlug string, userID string) (Assignment, error) {
	return m.assignment, m.err
}

func TestAssignHandler(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		engine     *mockEngine
		wantStatus int
		wantBody   jsonBody
	}{
		{
			name:       "success",
			query:      "experiment=checkout-redesign&user_id=u_123",
			engine:     &mockEngine{assignment: Assignment{Experiment: "checkout-redesign", Variant: "new_checkout", UserID: "u_123"}},
			wantStatus: http.StatusOK,
			wantBody:   jsonBody{"experiment": "checkout-redesign", "variant": "new_checkout", "user_id": "u_123"},
		},
		{
			name:       "missing experiment",
			query:      "user_id=u_123",
			engine:     &mockEngine{},
			wantStatus: http.StatusBadRequest,
			wantBody:   jsonBody{"error": "missing required parameter: experiment"},
		},
		{
			name:       "missing user_id",
			query:      "experiment=checkout-redesign",
			engine:     &mockEngine{},
			wantStatus: http.StatusBadRequest,
			wantBody:   jsonBody{"error": "missing required parameter: user_id"},
		},
		{
			name:       "experiment not found",
			query:      "experiment=unknown&user_id=u_123",
			engine:     &mockEngine{err: ErrExperimentNotFound},
			wantStatus: http.StatusNotFound,
			wantBody:   jsonBody{"error": "experiment not found"},
		},
		{
			name:       "experiment not running",
			query:      "experiment=checkout-redesign&user_id=u_123",
			engine:     &mockEngine{err: ErrExperimentNotRunning},
			wantStatus: http.StatusConflict,
			wantBody:   jsonBody{"error": "experiment not running"},
		},
		{
			name:       "engine error",
			query:      "experiment=checkout-redesign&user_id=u_123",
			engine:     &mockEngine{err: errors.New("something went wrong")},
			wantStatus: http.StatusInternalServerError,
			wantBody:   jsonBody{"error": "internal server error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &Server{engine: tt.engine}
			req := httptest.NewRequest(http.MethodGet, "/api/v1/assign?"+tt.query, nil)
			w := httptest.NewRecorder()

			srv.assignHandler(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			var body jsonBody
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatalf("invalid json: %v", err)
			}
			for key, want := range tt.wantBody {
				if body[key] != want {
					t.Errorf("expected %s=%s, got %s", key, want, body[key])
				}
			}
		})
	}
}
