package pearcut

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockEngine struct {
	assignment      Assignment
	err             error
	bulkAssignments []Assignment
	bulkErr         error
}

func (m *mockEngine) Assign(_ context.Context, userID string, experimentSlug string, attributes map[string]string) (Assignment, error) {
	return m.assignment, m.err
}

func (m *mockEngine) BulkAssign(_ context.Context, userID string, experimentSlugs []string, attributes map[string]string) ([]Assignment, error) {
	return m.bulkAssignments, m.bulkErr
}

func TestAssignHandler(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		engine     *mockEngine
		wantStatus int
		wantBody   jsonBody
	}{
		{
			name:       "success",
			body:       `{"experiment":"checkout-redesign","user_id":"u_123"}`,
			engine:     &mockEngine{assignment: Assignment{Experiment: "checkout-redesign", Variant: "new_checkout"}},
			wantStatus: http.StatusOK,
			wantBody:   jsonBody{"experiment": "checkout-redesign", "variant": "new_checkout", "user_id": "u_123"},
		},
		{
			name:       "success with attributes",
			body:       `{"experiment":"checkout-redesign","user_id":"u_123","attributes":{"country":"FR"}}`,
			engine:     &mockEngine{assignment: Assignment{Experiment: "checkout-redesign", Variant: "new_checkout"}},
			wantStatus: http.StatusOK,
			wantBody:   jsonBody{"experiment": "checkout-redesign", "variant": "new_checkout", "user_id": "u_123"},
		},
		{
			name:       "missing experiment",
			body:       `{"user_id":"u_123"}`,
			engine:     &mockEngine{},
			wantStatus: http.StatusBadRequest,
			wantBody:   jsonBody{"error": "missing required field: experiment"},
		},
		{
			name:       "missing user_id",
			body:       `{"experiment":"checkout-redesign"}`,
			engine:     &mockEngine{},
			wantStatus: http.StatusBadRequest,
			wantBody:   jsonBody{"error": "missing required field: user_id"},
		},
		{
			name:       "invalid json",
			body:       `not json`,
			engine:     &mockEngine{},
			wantStatus: http.StatusBadRequest,
			wantBody:   jsonBody{"error": "invalid json body"},
		},
		{
			name:       "experiment not found",
			body:       `{"experiment":"unknown","user_id":"u_123"}`,
			engine:     &mockEngine{err: ErrExperimentNotFound},
			wantStatus: http.StatusNotFound,
			wantBody:   jsonBody{"error": "experiment not found"},
		},
		{
			name:       "experiment not running",
			body:       `{"experiment":"checkout-redesign","user_id":"u_123"}`,
			engine:     &mockEngine{err: ErrExperimentNotRunning},
			wantStatus: http.StatusConflict,
			wantBody:   jsonBody{"error": "experiment not running"},
		},
		{
			name:       "user not targeted",
			body:       `{"experiment":"checkout-redesign","user_id":"u_123","attributes":{"country":"US"}}`,
			engine:     &mockEngine{err: ErrUserNotTargeted},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "user excluded by traffic",
			body:       `{"experiment":"checkout-redesign","user_id":"u_123"}`,
			engine:     &mockEngine{err: ErrUserExcludedByTraffic},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "engine error",
			body:       `{"experiment":"checkout-redesign","user_id":"u_123"}`,
			engine:     &mockEngine{err: errors.New("something went wrong")},
			wantStatus: http.StatusInternalServerError,
			wantBody:   jsonBody{"error": "internal server error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &Server{engine: tt.engine}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/assign", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			srv.assignHandler(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantBody != nil {
				var body jsonBody
				if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
					t.Fatalf("invalid json: %v", err)
				}
				for key, want := range tt.wantBody {
					if body[key] != want {
						t.Errorf("expected %s=%s, got %s", key, want, body[key])
					}
				}
			}
		})
	}
}

func TestBulkAssignHandler(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		engine     *mockEngine
		wantStatus int
		wantCheck  func(t *testing.T, body map[string]any)
	}{
		{
			name: "success with specific experiments",
			body: `{"user_id":"u_123","experiments":["exp-1","exp-2"]}`,
			engine: &mockEngine{bulkAssignments: []Assignment{
				{Experiment: "exp-1", Variant: "control"},
				{Experiment: "exp-2", Variant: "treatment"},
			}},
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body map[string]any) {
				if body["user_id"] != "u_123" {
					t.Errorf("expected user_id u_123, got %v", body["user_id"])
				}
				assignments := body["assignments"].(map[string]any)
				if len(assignments) != 2 {
					t.Fatalf("expected 2 assignments, got %d", len(assignments))
				}
				if assignments["exp-1"] != "control" {
					t.Errorf("expected exp-1=control, got %v", assignments["exp-1"])
				}
				if assignments["exp-2"] != "treatment" {
					t.Errorf("expected exp-2=treatment, got %v", assignments["exp-2"])
				}
			},
		},
		{
			name:       "success with all running",
			body:       `{"user_id":"u_123"}`,
			engine:     &mockEngine{bulkAssignments: []Assignment{{Experiment: "exp-1", Variant: "control"}}},
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body map[string]any) {
				assignments := body["assignments"].(map[string]any)
				if len(assignments) != 1 {
					t.Fatalf("expected 1 assignment, got %d", len(assignments))
				}
			},
		},
		{
			name:       "success with attributes",
			body:       `{"user_id":"u_123","attributes":{"country":"FR"}}`,
			engine:     &mockEngine{bulkAssignments: []Assignment{{Experiment: "exp-1", Variant: "control"}}},
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body map[string]any) {
				assignments := body["assignments"].(map[string]any)
				if len(assignments) != 1 {
					t.Fatalf("expected 1 assignment, got %d", len(assignments))
				}
			},
		},
		{
			name:       "missing user_id",
			body:       `{"experiments":["exp-1"]}`,
			engine:     &mockEngine{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid json",
			body:       `not json`,
			engine:     &mockEngine{},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "engine error",
			body:       `{"user_id":"u_123"}`,
			engine:     &mockEngine{bulkErr: errors.New("something went wrong")},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "empty result",
			body:       `{"user_id":"u_123"}`,
			engine:     &mockEngine{bulkAssignments: []Assignment{}},
			wantStatus: http.StatusOK,
			wantCheck: func(t *testing.T, body map[string]any) {
				assignments := body["assignments"].(map[string]any)
				if len(assignments) != 0 {
					t.Fatalf("expected 0 assignments, got %d", len(assignments))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := &Server{engine: tt.engine}
			req := httptest.NewRequest(http.MethodPost, "/api/v1/assign/bulk", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			srv.bulkAssignHandler(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantCheck != nil {
				var body map[string]any
				if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
					t.Fatalf("invalid json: %v", err)
				}
				tt.wantCheck(t, body)
			}
		})
	}
}
