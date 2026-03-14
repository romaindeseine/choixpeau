package pearcut

import (
	"errors"
	"fmt"
	"testing"
)

func newTestStore(t *testing.T, experiments []Experiment) Store {
	t.Helper()
	s, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	for _, exp := range experiments {
		if err := s.Create(exp); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func TestEngineAssign(t *testing.T) {
	runningExp := Experiment{
		Slug:   "exp-1",
		Seed:   "exp-1",
		Status: StatusRunning,
		Variants: []Variant{
			{Name: "control", Weight: 50},
			{Name: "treatment", Weight: 50},
		},
	}

	tests := []struct {
		name        string
		exps        []Experiment
		slug        string
		userID      string
		wantVariant string
		wantErr     error
	}{
		{
			name:    "experiment not found",
			exps:    nil,
			slug:    "unknown",
			userID:  "user-1",
			wantErr: ErrExperimentNotFound,
		},
		{
			name: "experiment is draft",
			exps: []Experiment{{
				Slug:     "draft-exp",
				Status:   StatusDraft,
				Variants: []Variant{{Name: "control", Weight: 100}},
			}},
			slug:    "draft-exp",
			userID:  "user-1",
			wantErr: ErrExperimentNotRunning,
		},
		{
			name: "experiment is paused",
			exps: []Experiment{{
				Slug:     "paused-exp",
				Status:   StatusPaused,
				Variants: []Variant{{Name: "control", Weight: 100}},
			}},
			slug:    "paused-exp",
			userID:  "user-1",
			wantErr: ErrExperimentNotRunning,
		},
		{
			name: "experiment is stopped",
			exps: []Experiment{{
				Slug:     "stopped-exp",
				Status:   StatusStopped,
				Variants: []Variant{{Name: "control", Weight: 100}},
			}},
			slug:    "stopped-exp",
			userID:  "user-1",
			wantErr: ErrExperimentNotRunning,
		},
		{
			name: "override hit",
			exps: []Experiment{{
				Slug:      "override-exp",
				Seed:      "override-exp",
				Status:    StatusRunning,
				Variants:  []Variant{{Name: "control", Weight: 50}, {Name: "treatment", Weight: 50}},
				Overrides: map[string]string{"user-42": "treatment"},
			}},
			slug:        "override-exp",
			userID:      "user-42",
			wantVariant: "treatment",
		},
		{
			name:        "single variant always assigned",
			exps:        []Experiment{{Slug: "single", Seed: "single", Status: StatusRunning, Variants: []Variant{{Name: "only", Weight: 100}}}},
			slug:        "single",
			userID:      "user-1",
			wantVariant: "only",
		},
		{
			name:   "basic assignment returns valid variant",
			exps:   []Experiment{runningExp},
			slug:   "exp-1",
			userID: "user-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t, tt.exps)
			e := NewEngine(store)
			got, err := e.Assign(tt.slug, tt.userID)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("got error %v, want %v", err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if got.Experiment != tt.slug {
				t.Errorf("got experiment %q, want %q", got.Experiment, tt.slug)
			}

			if tt.wantVariant != "" && got.Variant != tt.wantVariant {
				t.Errorf("got variant %q, want %q", got.Variant, tt.wantVariant)
			}

			if got.Variant == "" {
				t.Error("got empty variant")
			}
		})
	}
}

func TestEngineAssignDeterminism(t *testing.T) {
	store := newTestStore(t, []Experiment{{
		Slug:     "det-exp",
		Seed:     "det-exp",
		Status:   StatusRunning,
		Variants: []Variant{{Name: "a", Weight: 50}, {Name: "b", Weight: 50}},
	}})
	e := NewEngine(store)

	first, err := e.Assign("det-exp", "user-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i := 0; i < 100; i++ {
		got, err := e.Assign("det-exp", "user-123")
		if err != nil {
			t.Fatalf("unexpected error on iteration %d: %v", i, err)
		}
		if got.Variant != first.Variant {
			t.Fatalf("iteration %d: got %q, want %q", i, got.Variant, first.Variant)
		}
	}
}

func TestEngineAssignDistribution(t *testing.T) {
	store := newTestStore(t, []Experiment{{
		Slug:     "dist-exp",
		Seed:     "dist-exp",
		Status:   StatusRunning,
		Variants: []Variant{{Name: "control", Weight: 50}, {Name: "treatment", Weight: 50}},
	}})
	e := NewEngine(store)

	counts := map[string]int{}
	n := 10000
	for i := 0; i < n; i++ {
		a, err := e.Assign("dist-exp", fmt.Sprintf("user-%d", i))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		counts[a.Variant]++
	}

	for _, variant := range []string{"control", "treatment"} {
		ratio := float64(counts[variant]) / float64(n)
		if ratio < 0.45 || ratio > 0.55 {
			t.Errorf("variant %q got %.2f%% of traffic, expected ~50%%", variant, ratio*100)
		}
	}
}

func TestEngineBulkAssign(t *testing.T) {
	allExps := []Experiment{
		{Slug: "running-1", Seed: "running-1", Status: StatusRunning, Variants: []Variant{{Name: "control", Weight: 50}, {Name: "treatment", Weight: 50}}},
		{Slug: "running-2", Seed: "running-2", Status: StatusRunning, Variants: []Variant{{Name: "a", Weight: 100}}},
		{Slug: "draft-exp", Seed: "draft-exp", Status: StatusDraft, Variants: []Variant{{Name: "control", Weight: 100}}},
		{Slug: "paused-exp", Seed: "paused-exp", Status: StatusPaused, Variants: []Variant{{Name: "control", Weight: 100}}},
	}

	tests := []struct {
		name      string
		exps      []Experiment
		slugs     []string
		wantSlugs []string
	}{
		{
			name:      "all running experiments",
			exps:      allExps,
			slugs:     nil,
			wantSlugs: []string{"running-1", "running-2"},
		},
		{
			name:      "specific running experiments",
			exps:      allExps,
			slugs:     []string{"running-1"},
			wantSlugs: []string{"running-1"},
		},
		{
			name:      "non-running experiments filtered out",
			exps:      allExps,
			slugs:     []string{"running-1", "draft-exp", "paused-exp"},
			wantSlugs: []string{"running-1"},
		},
		{
			name:      "unknown experiment ignored",
			exps:      allExps,
			slugs:     []string{"unknown", "running-2"},
			wantSlugs: []string{"running-2"},
		},
		{
			name:      "no running experiments",
			exps:      []Experiment{{Slug: "draft", Seed: "draft", Status: StatusDraft, Variants: []Variant{{Name: "c", Weight: 100}}}},
			slugs:     nil,
			wantSlugs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t, tt.exps)
			e := NewEngine(store)
			assignments, err := e.BulkAssign("user-1", tt.slugs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(assignments) != len(tt.wantSlugs) {
				t.Fatalf("got %d assignments, want %d", len(assignments), len(tt.wantSlugs))
			}

			got := make(map[string]bool, len(assignments))
			for _, a := range assignments {
				got[a.Experiment] = true
			}
			for _, slug := range tt.wantSlugs {
				if !got[slug] {
					t.Errorf("missing assignment for %q", slug)
				}
			}
		})
	}
}

func TestEngineAssignSeedOverride(t *testing.T) {
	baseExp := Experiment{
		Slug:     "seed-exp",
		Seed:     "seed-exp",
		Status:   StatusRunning,
		Variants: []Variant{{Name: "a", Weight: 50}, {Name: "b", Weight: 50}},
	}

	expWithSeed := baseExp
	expWithSeed.Seed = "different-seed"

	store1 := newTestStore(t, []Experiment{baseExp})
	store2 := newTestStore(t, []Experiment{expWithSeed})
	e1 := NewEngine(store1)
	e2 := NewEngine(store2)

	diffs := 0
	n := 100
	for i := 0; i < n; i++ {
		uid := fmt.Sprintf("user-%d", i)
		a1, _ := e1.Assign("seed-exp", uid)
		a2, _ := e2.Assign("seed-exp", uid)
		if a1.Variant != a2.Variant {
			diffs++
		}
	}

	if diffs == 0 {
		t.Error("different seeds produced identical assignments for all users")
	}
}
