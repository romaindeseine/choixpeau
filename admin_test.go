package pearcut

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type mockStore struct {
	experiments map[string]Experiment
	createErr   error
	updateErr   error
	deleteErr   error
}

func (m *mockStore) Get(slug string) (Experiment, error) {
	exp, ok := m.experiments[slug]
	if !ok {
		return Experiment{}, ErrExperimentNotFound
	}
	return exp, nil
}

func (m *mockStore) List(filter ExperimentFilter, opts ListOptions) (ExperimentListResult, error) {
	all := make([]Experiment, 0, len(m.experiments))
	for _, exp := range m.experiments {
		all = append(all, exp)
	}
	filtered := filterExperiments(all, filter)
	sortExperiments(filtered, opts)
	page, total := paginateExperiments(filtered, opts)
	return ExperimentListResult{Experiments: page, Total: total}, nil
}

func (m *mockStore) Create(exp Experiment) error {
	if m.createErr != nil {
		return m.createErr
	}
	m.experiments[exp.Slug] = exp
	return nil
}

func (m *mockStore) Update(exp Experiment) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	if _, ok := m.experiments[exp.Slug]; !ok {
		return ErrExperimentNotFound
	}
	m.experiments[exp.Slug] = exp
	return nil
}

func (m *mockStore) Delete(slug string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	if _, ok := m.experiments[slug]; !ok {
		return ErrExperimentNotFound
	}
	delete(m.experiments, slug)
	return nil
}

func newMockStore() *mockStore {
	return &mockStore{experiments: make(map[string]Experiment)}
}

type noopAssignStore struct{}

func (noopAssignStore) Get(slug string) (Experiment, error) {
	return Experiment{}, ErrExperimentNotFound
}
func (noopAssignStore) List(slugs []string, status ExperimentStatus) ([]Experiment, error) {
	return nil, nil
}
func (noopAssignStore) Set(exp Experiment) {}
func (noopAssignStore) Delete(slug string) {}

func newTestServer(store ExperimentStore) *Server {
	return &Server{experimentStore: store, assignStore: noopAssignStore{}}
}

func TestListExperiments(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		store      *mockStore
		wantStatus int
		wantCount  int
		wantTotal  int
	}{
		{
			name: "all experiments",
			store: &mockStore{experiments: map[string]Experiment{
				"exp-a": {Slug: "exp-a", Status: StatusRunning},
				"exp-b": {Slug: "exp-b", Status: StatusDraft},
			}},
			wantStatus: http.StatusOK,
			wantCount:  2,
			wantTotal:  2,
		},
		{
			name:  "filter by status",
			query: "status=running",
			store: &mockStore{experiments: map[string]Experiment{
				"exp-a": {Slug: "exp-a", Status: StatusRunning},
				"exp-b": {Slug: "exp-b", Status: StatusDraft},
			}},
			wantStatus: http.StatusOK,
			wantCount:  1,
			wantTotal:  1,
		},
		{
			name:       "filter returns empty",
			query:      "status=draft",
			store:      &mockStore{experiments: map[string]Experiment{"exp-a": {Slug: "exp-a", Status: StatusRunning}}},
			wantStatus: http.StatusOK,
			wantCount:  0,
			wantTotal:  0,
		},
		{
			name:       "invalid status",
			query:      "status=bogus",
			store:      newMockStore(),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid page",
			query:      "page=-1",
			store:      newMockStore(),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid per_page",
			query:      "per_page=0",
			store:      newMockStore(),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "per_page too large",
			query:      "per_page=999",
			store:      newMockStore(),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid sort",
			query:      "sort=bogus",
			store:      newMockStore(),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid order",
			query:      "order=bogus",
			store:      newMockStore(),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:  "pagination page 1",
			query: "page=1&per_page=1",
			store: &mockStore{experiments: map[string]Experiment{
				"exp-a": {Slug: "exp-a", Status: StatusRunning},
				"exp-b": {Slug: "exp-b", Status: StatusDraft},
			}},
			wantStatus: http.StatusOK,
			wantCount:  1,
			wantTotal:  2,
		},
		{
			name:  "pagination beyond last page",
			query: "page=99&per_page=20",
			store: &mockStore{experiments: map[string]Experiment{
				"exp-a": {Slug: "exp-a", Status: StatusRunning},
			}},
			wantStatus: http.StatusOK,
			wantCount:  0,
			wantTotal:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(tt.store)
			path := "/admin/v1/experiments"
			if tt.query != "" {
				path += "?" + tt.query
			}
			req := httptest.NewRequest(http.MethodGet, path, nil)
			w := httptest.NewRecorder()

			srv.listExperiments(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d; body: %s", tt.wantStatus, w.Code, w.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var body ListExperimentsResponse
				if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
					t.Fatalf("invalid json: %v", err)
				}
				if len(body.Data) != tt.wantCount {
					t.Fatalf("expected %d experiments, got %d", tt.wantCount, len(body.Data))
				}
				if body.Total != tt.wantTotal {
					t.Fatalf("expected total %d, got %d", tt.wantTotal, body.Total)
				}
			}
		})
	}
}

func TestGetExperiment(t *testing.T) {
	tests := []struct {
		name       string
		slug       string
		store      *mockStore
		wantStatus int
	}{
		{
			name: "success",
			slug: "exp-a",
			store: &mockStore{experiments: map[string]Experiment{
				"exp-a": {Slug: "exp-a", Status: StatusRunning},
			}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found",
			slug:       "unknown",
			store:      newMockStore(),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(tt.store)
			mux := http.NewServeMux()
			mux.HandleFunc("GET /admin/v1/experiments/{slug}", srv.getExperiment)

			req := httptest.NewRequest(http.MethodGet, "/admin/v1/experiments/"+tt.slug, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantStatus == http.StatusOK {
				var exp Experiment
				if err := json.NewDecoder(w.Body).Decode(&exp); err != nil {
					t.Fatalf("invalid json: %v", err)
				}
				if exp.Slug != tt.slug {
					t.Fatalf("expected slug %s, got %s", tt.slug, exp.Slug)
				}
			}
		})
	}
}

func TestCreateExperiment(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		store      *mockStore
		wantStatus int
	}{
		{
			name:       "success",
			body:       `{"slug":"new-exp","status":"draft","variants":[{"name":"control","weight":1}],"description":"test desc","tags":["checkout"],"owner":"team-a","hypothesis":"improves ctr"}`,
			store:      newMockStore(),
			wantStatus: http.StatusCreated,
		},
		{
			name:       "already exists",
			body:       `{"slug":"exp-a","status":"draft","variants":[{"name":"control","weight":1}]}`,
			store:      &mockStore{experiments: make(map[string]Experiment), createErr: ErrExperimentExists},
			wantStatus: http.StatusConflict,
		},
		{
			name:       "invalid json",
			body:       `{bad`,
			store:      newMockStore(),
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "validation error",
			body:       `{"slug":"new-exp","status":"draft","variants":[]}`,
			store:      &mockStore{experiments: make(map[string]Experiment), createErr: fmt.Errorf("experiment \"new-exp\" has no variants")},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(tt.store)
			req := httptest.NewRequest(http.MethodPost, "/admin/v1/experiments", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			srv.createExperiment(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d; body: %s", tt.wantStatus, w.Code, w.Body.String())
			}

			if tt.wantStatus == http.StatusCreated {
				var exp Experiment
				if err := json.NewDecoder(w.Body).Decode(&exp); err != nil {
					t.Fatalf("invalid json: %v", err)
				}
				if exp.Slug != "new-exp" {
					t.Fatalf("expected slug new-exp, got %s", exp.Slug)
				}
				if exp.Description != "test desc" {
					t.Errorf("Description = %q, want %q", exp.Description, "test desc")
				}
				if len(exp.Tags) != 1 || exp.Tags[0] != "checkout" {
					t.Errorf("Tags = %v, want [checkout]", exp.Tags)
				}
				if exp.Owner != "team-a" {
					t.Errorf("Owner = %q, want %q", exp.Owner, "team-a")
				}
				if exp.Hypothesis != "improves ctr" {
					t.Errorf("Hypothesis = %q, want %q", exp.Hypothesis, "improves ctr")
				}
			}
		})
	}
}

func TestUpdateExperiment(t *testing.T) {
	tests := []struct {
		name       string
		slug       string
		body       string
		store      *mockStore
		wantStatus int
	}{
		{
			name: "success",
			slug: "exp-a",
			body: `{"status":"running","variants":[{"name":"control","weight":1}]}`,
			store: &mockStore{experiments: map[string]Experiment{
				"exp-a": {Slug: "exp-a", Status: StatusDraft, Variants: []Variant{{Name: "control", Weight: 1}}},
			}},
			wantStatus: http.StatusOK,
		},
		{
			name:       "not found",
			slug:       "unknown",
			body:       `{"status":"running","variants":[{"name":"control","weight":1}]}`,
			store:      newMockStore(),
			wantStatus: http.StatusNotFound,
		},
		{
			name: "invalid json",
			slug: "exp-a",
			body: `{bad`,
			store: &mockStore{experiments: map[string]Experiment{
				"exp-a": {Slug: "exp-a", Status: StatusDraft},
			}},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "validation error",
			slug: "exp-a",
			body: `{"status":"running","variants":[]}`,
			store: &mockStore{
				experiments: map[string]Experiment{
					"exp-a": {Slug: "exp-a", Status: StatusDraft},
				},
				updateErr: fmt.Errorf("experiment \"exp-a\" has no variants"),
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(tt.store)
			mux := http.NewServeMux()
			mux.HandleFunc("PUT /admin/v1/experiments/{slug}", srv.updateExperiment)

			req := httptest.NewRequest(http.MethodPut, "/admin/v1/experiments/"+tt.slug, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d; body: %s", tt.wantStatus, w.Code, w.Body.String())
			}

			if tt.wantStatus == http.StatusOK {
				var exp Experiment
				if err := json.NewDecoder(w.Body).Decode(&exp); err != nil {
					t.Fatalf("invalid json: %v", err)
				}
				if exp.Slug != tt.slug {
					t.Fatalf("expected slug %s, got %s", tt.slug, exp.Slug)
				}
			}
		})
	}
}

func TestDeleteExperiment(t *testing.T) {
	tests := []struct {
		name       string
		slug       string
		store      *mockStore
		wantStatus int
	}{
		{
			name: "success",
			slug: "exp-a",
			store: &mockStore{experiments: map[string]Experiment{
				"exp-a": {Slug: "exp-a", Status: StatusRunning},
			}},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "not found",
			slug:       "unknown",
			store:      newMockStore(),
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestServer(tt.store)
			mux := http.NewServeMux()
			mux.HandleFunc("DELETE /admin/v1/experiments/{slug}", srv.deleteExperiment)

			req := httptest.NewRequest(http.MethodDelete, "/admin/v1/experiments/"+tt.slug, nil)
			w := httptest.NewRecorder()

			mux.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, w.Code)
			}

			if tt.wantStatus == http.StatusNoContent && w.Body.Len() != 0 {
				t.Fatalf("expected empty body, got %s", w.Body.String())
			}
		})
	}
}
