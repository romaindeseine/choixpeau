package main

import (
	"errors"
	"fmt"
	"testing"
)

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
		name       string
		store      ReadStore
		slug       string
		userID     string
		want       Assignment
		wantErr    error
		checkValue bool
	}{
		{
			name:    "experiment not found",
			store:   newMemStore(),
			slug:    "unknown",
			userID:  "user-1",
			wantErr: ErrExperimentNotFound,
		},
		{
			name: "experiment is draft",
			store: newMemStore(Experiment{
				Slug:     "draft-exp",
				Status:   StatusDraft,
				Variants: []Variant{{Name: "control", Weight: 100}},
			}),
			slug:    "draft-exp",
			userID:  "user-1",
			wantErr: ErrExperimentNotRunning,
		},
		{
			name: "experiment is paused",
			store: newMemStore(Experiment{
				Slug:     "paused-exp",
				Status:   StatusPaused,
				Variants: []Variant{{Name: "control", Weight: 100}},
			}),
			slug:    "paused-exp",
			userID:  "user-1",
			wantErr: ErrExperimentNotRunning,
		},
		{
			name: "experiment is stopped",
			store: newMemStore(Experiment{
				Slug:     "stopped-exp",
				Status:   StatusStopped,
				Variants: []Variant{{Name: "control", Weight: 100}},
			}),
			slug:    "stopped-exp",
			userID:  "user-1",
			wantErr: ErrExperimentNotRunning,
		},
		{
			name: "override hit",
			store: newMemStore(Experiment{
				Slug:      "override-exp",
				Seed:      "override-exp",
				Status:    StatusRunning,
				Variants:  []Variant{{Name: "control", Weight: 50}, {Name: "treatment", Weight: 50}},
				Overrides: map[string]string{"user-42": "treatment"},
			}),
			slug:       "override-exp",
			userID:     "user-42",
			want:       Assignment{Experiment: "override-exp", Variant: "treatment", UserID: "user-42"},
			checkValue: true,
		},
		{
			name:       "single variant always assigned",
			store:      newMemStore(Experiment{Slug: "single", Seed: "single", Status: StatusRunning, Variants: []Variant{{Name: "only", Weight: 100}}}),
			slug:       "single",
			userID:     "user-1",
			want:       Assignment{Experiment: "single", Variant: "only", UserID: "user-1"},
			checkValue: true,
		},
		{
			name:  "basic assignment returns valid variant",
			store: newMemStore(runningExp),
			slug:  "exp-1",
			// This user ID produces a known assignment via murmur3
			userID: "user-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := NewEngine(tt.store)
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

			if tt.checkValue && got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}

			if got.Experiment != tt.slug || got.UserID != tt.userID {
				t.Errorf("got experiment=%q user_id=%q, want experiment=%q user_id=%q", got.Experiment, got.UserID, tt.slug, tt.userID)
			}

			if got.Variant == "" {
				t.Error("got empty variant")
			}
		})
	}
}

func TestEngineAssignDeterminism(t *testing.T) {
	store := newMemStore(Experiment{
		Slug:     "det-exp",
		Seed:     "det-exp",
		Status:   StatusRunning,
		Variants: []Variant{{Name: "a", Weight: 50}, {Name: "b", Weight: 50}},
	})
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
		if got != first {
			t.Fatalf("iteration %d: got %+v, want %+v", i, got, first)
		}
	}
}

func TestEngineAssignDistribution(t *testing.T) {
	store := newMemStore(Experiment{
		Slug:     "dist-exp",
		Seed:     "dist-exp",
		Status:   StatusRunning,
		Variants: []Variant{{Name: "control", Weight: 50}, {Name: "treatment", Weight: 50}},
	})
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

func TestEngineAssignSeedOverride(t *testing.T) {
	baseExp := Experiment{
		Slug:     "seed-exp",
		Seed:     "seed-exp",
		Status:   StatusRunning,
		Variants: []Variant{{Name: "a", Weight: 50}, {Name: "b", Weight: 50}},
	}

	expWithSeed := baseExp
	expWithSeed.Seed = "different-seed"

	e1 := NewEngine(newMemStore(baseExp))
	e2 := NewEngine(newMemStore(expWithSeed))

	// Collect assignments for many users; with a different seed
	// at least some users should get different variants.
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
