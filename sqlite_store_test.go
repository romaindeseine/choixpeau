package pearcut

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
		Overrides:   map[string]string{"user-42": "treatment"},
		Description: "test experiment description",
		Tags:        []string{"checkout", "mobile"},
		Owner:       "team-growth",
		Hypothesis:  "new variant improves conversion",
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
	if got.Description != exp.Description {
		t.Errorf("Description = %q, want %q", got.Description, exp.Description)
	}
	if len(got.Tags) != len(exp.Tags) {
		t.Fatalf("Tags count = %d, want %d", len(got.Tags), len(exp.Tags))
	}
	for i, tag := range got.Tags {
		if tag != exp.Tags[i] {
			t.Errorf("Tags[%d] = %q, want %q", i, tag, exp.Tags[i])
		}
	}
	if got.Owner != exp.Owner {
		t.Errorf("Owner = %q, want %q", got.Owner, exp.Owner)
	}
	if got.Hypothesis != exp.Hypothesis {
		t.Errorf("Hypothesis = %q, want %q", got.Hypothesis, exp.Hypothesis)
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
		opts      ListOptions
		wantSlugs []string
		wantTotal int
	}{
		{
			name:      "no filter",
			filter:    ExperimentFilter{},
			wantSlugs: []string{"exp-a", "exp-b", "exp-c"},
			wantTotal: 3,
		},
		{
			name:      "filter by running status",
			filter:    ExperimentFilter{Status: &running},
			wantSlugs: []string{"exp-a", "exp-c"},
			wantTotal: 2,
		},
		{
			name:      "filter by draft status",
			filter:    ExperimentFilter{Status: &draft},
			wantSlugs: []string{"exp-b"},
			wantTotal: 1,
		},
		{
			name:      "pagination page 1",
			opts:      ListOptions{Page: 1, PerPage: 2},
			wantSlugs: []string{"exp-a", "exp-b"},
			wantTotal: 3,
		},
		{
			name:      "pagination page 2",
			opts:      ListOptions{Page: 2, PerPage: 2},
			wantSlugs: []string{"exp-c"},
			wantTotal: 3,
		},
		{
			name:      "pagination beyond last page",
			opts:      ListOptions{Page: 10, PerPage: 2},
			wantSlugs: []string{},
			wantTotal: 3,
		},
		{
			name:      "search by slug",
			filter:    ExperimentFilter{Search: "exp-a"},
			wantSlugs: []string{"exp-a"},
			wantTotal: 1,
		},
		{
			name:      "search no match",
			filter:    ExperimentFilter{Search: "zzz"},
			wantSlugs: []string{},
			wantTotal: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.List(tt.filter, tt.opts)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			if got.Total != tt.wantTotal {
				t.Fatalf("List() total = %d, want %d", got.Total, tt.wantTotal)
			}

			gotSlugs := make([]string, len(got.Experiments))
			for i, exp := range got.Experiments {
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
	exp.Description = "updated description"
	exp.Tags = []string{"pricing"}
	exp.Owner = "team-platform"
	exp.Hypothesis = "updated hypothesis"

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
	if got.Description != "updated description" {
		t.Errorf("Description = %q, want %q", got.Description, "updated description")
	}
	if len(got.Tags) != 1 || got.Tags[0] != "pricing" {
		t.Errorf("Tags = %v, want [pricing]", got.Tags)
	}
	if got.Owner != "team-platform" {
		t.Errorf("Owner = %q, want %q", got.Owner, "team-platform")
	}
	if got.Hypothesis != "updated hypothesis" {
		t.Errorf("Hypothesis = %q, want %q", got.Hypothesis, "updated hypothesis")
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

func TestSQLiteStoreTargetingRulesRoundTrip(t *testing.T) {
	s := newTestSQLiteStore(t)
	exp := Experiment{
		Slug:     "targeted-exp",
		Status:   StatusRunning,
		Seed:     "targeted-exp",
		Variants: []Variant{{Name: "control", Weight: 100}},
		TargetingRules: []TargetingRule{
			{Attribute: "country", Operator: OperatorIn, Values: []string{"FR", "US"}},
			{Attribute: "plan", Operator: OperatorNotIn, Values: []string{"free"}},
		},
	}

	if err := s.Create(exp); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := s.Get("targeted-exp")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if len(got.TargetingRules) != 2 {
		t.Fatalf("TargetingRules count = %d, want 2", len(got.TargetingRules))
	}
	if got.TargetingRules[0].Attribute != "country" {
		t.Errorf("Rule[0].Attribute = %q, want %q", got.TargetingRules[0].Attribute, "country")
	}
	if got.TargetingRules[0].Operator != OperatorIn {
		t.Errorf("Rule[0].Operator = %q, want %q", got.TargetingRules[0].Operator, OperatorIn)
	}
	if len(got.TargetingRules[0].Values) != 2 {
		t.Errorf("Rule[0].Values count = %d, want 2", len(got.TargetingRules[0].Values))
	}
	if got.TargetingRules[1].Attribute != "plan" {
		t.Errorf("Rule[1].Attribute = %q, want %q", got.TargetingRules[1].Attribute, "plan")
	}
	if got.TargetingRules[1].Operator != OperatorNotIn {
		t.Errorf("Rule[1].Operator = %q, want %q", got.TargetingRules[1].Operator, OperatorNotIn)
	}
}

func TestSQLiteStoreDeleteNotFound(t *testing.T) {
	s := newTestSQLiteStore(t)

	err := s.Delete("nonexistent")
	if !errors.Is(err, ErrExperimentNotFound) {
		t.Errorf("Delete() error = %v, want ErrExperimentNotFound", err)
	}
}
