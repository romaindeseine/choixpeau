package pearcut

import (
	"errors"
	"sync"
	"testing"
)

func newTestCachedStore(t *testing.T) *CachedStore {
	t.Helper()
	sqlite := newTestSQLiteStore(t)
	cs, err := NewCachedStore(sqlite)
	if err != nil {
		t.Fatal(err)
	}
	return cs
}

func TestCachedStoreGet(t *testing.T) {
	sqlite := newTestSQLiteStore(t)

	// Seed data before building cache.
	if err := sqlite.Create(testExperiment("exp-a")); err != nil {
		t.Fatal(err)
	}

	cs, err := NewCachedStore(sqlite)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		slug    string
		wantErr error
	}{
		{name: "hit after warm-up", slug: "exp-a", wantErr: nil},
		{name: "not found", slug: "nonexistent", wantErr: ErrExperimentNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cs.Get(tt.slug)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Get(%q) error = %v, want %v", tt.slug, err, tt.wantErr)
			}
			if err == nil && got.Slug != tt.slug {
				t.Errorf("Slug = %q, want %q", got.Slug, tt.slug)
			}
		})
	}
}

func TestCachedStoreList(t *testing.T) {
	sqlite := newTestSQLiteStore(t)

	running := StatusRunning
	draft := StatusDraft

	experiments := []Experiment{
		{Slug: "exp-a", Status: StatusRunning, Seed: "exp-a", Variants: []Variant{{Name: "v1", Weight: 1}}},
		{Slug: "exp-b", Status: StatusDraft, Seed: "exp-b", Variants: []Variant{{Name: "v1", Weight: 1}}},
		{Slug: "exp-c", Status: StatusRunning, Seed: "exp-c", Variants: []Variant{{Name: "v1", Weight: 1}}},
	}
	for _, exp := range experiments {
		if err := sqlite.Create(exp); err != nil {
			t.Fatalf("Create(%q) error = %v", exp.Slug, err)
		}
	}

	cs, err := NewCachedStore(sqlite)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		filter    ExperimentFilter
		wantSlugs []string
	}{
		{
			name:      "no filter",
			filter:    ExperimentFilter{},
			wantSlugs: []string{"exp-a", "exp-b", "exp-c"},
		},
		{
			name:      "filter by running status",
			filter:    ExperimentFilter{Status: &running},
			wantSlugs: []string{"exp-a", "exp-c"},
		},
		{
			name:      "filter by draft status",
			filter:    ExperimentFilter{Status: &draft},
			wantSlugs: []string{"exp-b"},
		},
		{
			name:      "filter by slugs",
			filter:    ExperimentFilter{Slugs: []string{"exp-a", "exp-c"}},
			wantSlugs: []string{"exp-a", "exp-c"},
		},
		{
			name:      "filter by status and slugs",
			filter:    ExperimentFilter{Status: &running, Slugs: []string{"exp-a", "exp-b"}},
			wantSlugs: []string{"exp-a"},
		},
		{
			name:      "no match",
			filter:    ExperimentFilter{Status: &draft, Slugs: []string{"exp-a"}},
			wantSlugs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cs.List(tt.filter)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			gotSlugs := make([]string, len(got))
			for i, exp := range got {
				gotSlugs[i] = exp.Slug
			}

			if len(gotSlugs) != len(tt.wantSlugs) {
				t.Fatalf("List() returned %v, want %v", gotSlugs, tt.wantSlugs)
			}
			for i := range gotSlugs {
				if gotSlugs[i] != tt.wantSlugs[i] {
					t.Errorf("List()[%d] = %q, want %q", i, gotSlugs[i], tt.wantSlugs[i])
				}
			}
		})
	}
}

func TestCachedStoreCreateUpdatesCache(t *testing.T) {
	cs := newTestCachedStore(t)

	exp := testExperiment("new-exp")
	if err := cs.Create(exp); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := cs.Get("new-exp")
	if err != nil {
		t.Fatalf("Get() after Create() error = %v", err)
	}
	if got.Slug != "new-exp" {
		t.Errorf("Slug = %q, want %q", got.Slug, "new-exp")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
}

func TestCachedStoreUpdateUpdatesCache(t *testing.T) {
	cs := newTestCachedStore(t)

	exp := testExperiment("update-exp")
	if err := cs.Create(exp); err != nil {
		t.Fatal(err)
	}

	exp.Status = StatusPaused
	if err := cs.Update(exp); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := cs.Get("update-exp")
	if err != nil {
		t.Fatalf("Get() after Update() error = %v", err)
	}
	if got.Status != StatusPaused {
		t.Errorf("Status = %q, want %q", got.Status, StatusPaused)
	}
}

func TestCachedStoreDeleteUpdatesCache(t *testing.T) {
	cs := newTestCachedStore(t)

	exp := testExperiment("delete-exp")
	if err := cs.Create(exp); err != nil {
		t.Fatal(err)
	}

	if err := cs.Delete("delete-exp"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := cs.Get("delete-exp")
	if !errors.Is(err, ErrExperimentNotFound) {
		t.Errorf("Get() after Delete() error = %v, want ErrExperimentNotFound", err)
	}
}

func TestCachedStoreConcurrency(t *testing.T) {
	cs := newTestCachedStore(t)

	// Seed some experiments.
	for i := 0; i < 5; i++ {
		slug := "conc-" + string(rune('a'+i))
		if err := cs.Create(testExperiment(slug)); err != nil {
			t.Fatal(err)
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			slug := "conc-" + string(rune('a'+(n%5)))
			cs.Get(slug)
			cs.List(ExperimentFilter{})
		}(i)
	}
	wg.Wait()
}
