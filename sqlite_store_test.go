package choixpeau

import (
	"errors"
	"testing"
)

func newTestSQLiteStore(t *testing.T) *SQLiteStore {
	t.Helper()
	s, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func testExperiment(slug string) Experiment {
	return Experiment{
		Slug:   slug,
		Status: StatusRunning,
		Seed:   slug,
		Variants: []Variant{
			{Name: "control", Weight: 50},
			{Name: "treatment", Weight: 50},
		},
		Overrides: map[string]string{"user-42": "treatment"},
	}
}

func TestSQLiteStoreCreateAndGet(t *testing.T) {
	s := newTestSQLiteStore(t)
	exp := testExperiment("checkout-redesign")

	if err := s.Create(exp); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := s.Get("checkout-redesign")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Slug != exp.Slug {
		t.Errorf("Slug = %q, want %q", got.Slug, exp.Slug)
	}
	if got.Status != exp.Status {
		t.Errorf("Status = %q, want %q", got.Status, exp.Status)
	}
	if got.Seed != exp.Seed {
		t.Errorf("Seed = %q, want %q", got.Seed, exp.Seed)
	}
	if len(got.Variants) != len(exp.Variants) {
		t.Fatalf("Variants count = %d, want %d", len(got.Variants), len(exp.Variants))
	}
	for i, v := range got.Variants {
		if v.Name != exp.Variants[i].Name || v.Weight != exp.Variants[i].Weight {
			t.Errorf("Variant[%d] = %+v, want %+v", i, v, exp.Variants[i])
		}
	}
	if got.Overrides["user-42"] != "treatment" {
		t.Errorf("Override user-42 = %q, want %q", got.Overrides["user-42"], "treatment")
	}
	if got.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}
	if got.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should not be zero")
	}
}

func TestSQLiteStoreCreateDuplicate(t *testing.T) {
	s := newTestSQLiteStore(t)
	exp := testExperiment("dup-test")

	if err := s.Create(exp); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}

	err := s.Create(exp)
	if !errors.Is(err, ErrExperimentExists) {
		t.Errorf("second Create() error = %v, want ErrExperimentExists", err)
	}
}

func TestSQLiteStoreCreateSeedDefault(t *testing.T) {
	s := newTestSQLiteStore(t)
	exp := Experiment{
		Slug:     "no-seed",
		Status:   StatusRunning,
		Variants: []Variant{{Name: "a", Weight: 1}},
	}

	if err := s.Create(exp); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := s.Get("no-seed")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Seed != "no-seed" {
		t.Errorf("Seed = %q, want %q (should default to slug)", got.Seed, "no-seed")
	}
}

func TestSQLiteStoreCreateValidation(t *testing.T) {
	s := newTestSQLiteStore(t)
	exp := Experiment{Slug: "", Status: StatusRunning, Variants: []Variant{{Name: "a", Weight: 1}}}

	err := s.Create(exp)
	if err == nil {
		t.Fatal("Create() should fail validation for empty slug")
	}
}

func TestSQLiteStoreGetNotFound(t *testing.T) {
	s := newTestSQLiteStore(t)

	_, err := s.Get("nonexistent")
	if !errors.Is(err, ErrExperimentNotFound) {
		t.Errorf("Get() error = %v, want ErrExperimentNotFound", err)
	}
}

func TestSQLiteStoreList(t *testing.T) {
	s := newTestSQLiteStore(t)

	running := StatusRunning
	draft := StatusDraft

	experiments := []Experiment{
		{Slug: "exp-a", Status: StatusRunning, Seed: "exp-a", Variants: []Variant{{Name: "v1", Weight: 1}}},
		{Slug: "exp-b", Status: StatusDraft, Seed: "exp-b", Variants: []Variant{{Name: "v1", Weight: 1}}},
		{Slug: "exp-c", Status: StatusRunning, Seed: "exp-c", Variants: []Variant{{Name: "v1", Weight: 1}}},
	}
	for _, exp := range experiments {
		if err := s.Create(exp); err != nil {
			t.Fatalf("Create(%q) error = %v", exp.Slug, err)
		}
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
			got, err := s.List(tt.filter)
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

func TestSQLiteStoreUpdate(t *testing.T) {
	s := newTestSQLiteStore(t)
	exp := testExperiment("update-test")

	if err := s.Create(exp); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	created, _ := s.Get("update-test")

	exp.Status = StatusPaused
	exp.Variants = []Variant{{Name: "solo", Weight: 100}}
	exp.Overrides = map[string]string{"user-99": "solo"}

	if err := s.Update(exp); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := s.Get("update-test")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if got.Status != StatusPaused {
		t.Errorf("Status = %q, want %q", got.Status, StatusPaused)
	}
	if len(got.Variants) != 1 || got.Variants[0].Name != "solo" {
		t.Errorf("Variants = %+v, want [{solo 100}]", got.Variants)
	}
	if got.Overrides["user-99"] != "solo" {
		t.Errorf("Override user-99 = %q, want %q", got.Overrides["user-99"], "solo")
	}
	if _, ok := got.Overrides["user-42"]; ok {
		t.Error("old override user-42 should be removed")
	}
	if got.UpdatedAt.Before(created.CreatedAt) {
		t.Error("UpdatedAt should not be before CreatedAt")
	}
}

func TestSQLiteStoreUpdateNotFound(t *testing.T) {
	s := newTestSQLiteStore(t)
	exp := testExperiment("ghost")

	err := s.Update(exp)
	if !errors.Is(err, ErrExperimentNotFound) {
		t.Errorf("Update() error = %v, want ErrExperimentNotFound", err)
	}
}

func TestSQLiteStoreDelete(t *testing.T) {
	s := newTestSQLiteStore(t)
	exp := testExperiment("delete-me")

	if err := s.Create(exp); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := s.Delete("delete-me"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := s.Get("delete-me")
	if !errors.Is(err, ErrExperimentNotFound) {
		t.Errorf("Get() after Delete() error = %v, want ErrExperimentNotFound", err)
	}
}

func TestSQLiteStoreDeleteNotFound(t *testing.T) {
	s := newTestSQLiteStore(t)

	err := s.Delete("nonexistent")
	if !errors.Is(err, ErrExperimentNotFound) {
		t.Errorf("Delete() error = %v, want ErrExperimentNotFound", err)
	}
}
