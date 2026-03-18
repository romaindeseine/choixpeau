package pearcut

import (
	"errors"
	"sync"
	"testing"
)

func TestMemStoreGet(t *testing.T) {
	experiments := []Experiment{
		{Slug: "exp-a", Status: StatusRunning, Seed: "exp-a", Variants: []Variant{{Name: "v1", Weight: 1}}},
	}
	ms := NewMemStore(experiments)

	tests := []struct {
		name    string
		slug    string
		wantErr error
	}{
		{name: "hit", slug: "exp-a", wantErr: nil},
		{name: "not found", slug: "nonexistent", wantErr: ErrExperimentNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ms.Get(tt.slug)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Get(%q) error = %v, want %v", tt.slug, err, tt.wantErr)
			}
			if err == nil && got.Slug != tt.slug {
				t.Errorf("Slug = %q, want %q", got.Slug, tt.slug)
			}
		})
	}
}

func TestMemStoreList(t *testing.T) {
	experiments := []Experiment{
		{Slug: "exp-a", Status: StatusRunning, Seed: "exp-a", Variants: []Variant{{Name: "v1", Weight: 1}}},
		{Slug: "exp-b", Status: StatusDraft, Seed: "exp-b", Variants: []Variant{{Name: "v1", Weight: 1}}},
		{Slug: "exp-c", Status: StatusRunning, Seed: "exp-c", Variants: []Variant{{Name: "v1", Weight: 1}}},
	}
	ms := NewMemStore(experiments)

	tests := []struct {
		name      string
		slugs     []string
		status    ExperimentStatus
		wantSlugs map[string]bool
	}{
		{
			name:      "all running",
			status:    StatusRunning,
			wantSlugs: map[string]bool{"exp-a": true, "exp-c": true},
		},
		{
			name:      "all draft",
			status:    StatusDraft,
			wantSlugs: map[string]bool{"exp-b": true},
		},
		{
			name:      "specific slugs running",
			slugs:     []string{"exp-a", "exp-b"},
			status:    StatusRunning,
			wantSlugs: map[string]bool{"exp-a": true},
		},
		{
			name:      "no match",
			slugs:     []string{"exp-b"},
			status:    StatusRunning,
			wantSlugs: map[string]bool{},
		},
		{
			name:      "unknown slug ignored",
			slugs:     []string{"unknown", "exp-c"},
			status:    StatusRunning,
			wantSlugs: map[string]bool{"exp-c": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ms.List(tt.slugs, tt.status)
			if err != nil {
				t.Fatalf("List() error = %v", err)
			}

			if len(got) != len(tt.wantSlugs) {
				t.Fatalf("List() returned %d experiments, want %d", len(got), len(tt.wantSlugs))
			}
			for _, exp := range got {
				if !tt.wantSlugs[exp.Slug] {
					t.Errorf("unexpected experiment %q", exp.Slug)
				}
			}
		})
	}
}

func TestMemStoreSet(t *testing.T) {
	ms := NewMemStore(nil)

	exp := Experiment{Slug: "new-exp", Status: StatusRunning, Variants: []Variant{{Name: "v1", Weight: 1}}}
	ms.Set(exp)

	got, err := ms.Get("new-exp")
	if err != nil {
		t.Fatalf("Get() after Set() error = %v", err)
	}
	if got.Slug != "new-exp" {
		t.Errorf("Slug = %q, want %q", got.Slug, "new-exp")
	}
}

func TestMemStoreDelete(t *testing.T) {
	experiments := []Experiment{
		{Slug: "exp-a", Status: StatusRunning, Variants: []Variant{{Name: "v1", Weight: 1}}},
	}
	ms := NewMemStore(experiments)

	ms.Delete("exp-a")

	_, err := ms.Get("exp-a")
	if !errors.Is(err, ErrExperimentNotFound) {
		t.Errorf("Get() after Delete() error = %v, want ErrExperimentNotFound", err)
	}
}

func TestMemStoreConcurrency(t *testing.T) {
	experiments := []Experiment{
		{Slug: "conc-a", Status: StatusRunning, Variants: []Variant{{Name: "v1", Weight: 1}}},
		{Slug: "conc-b", Status: StatusRunning, Variants: []Variant{{Name: "v1", Weight: 1}}},
	}
	ms := NewMemStore(experiments)

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			if n%2 == 0 {
				ms.Get("conc-a")
				ms.List(nil, StatusRunning)
			} else {
				ms.Set(Experiment{Slug: "conc-a", Status: StatusRunning, Variants: []Variant{{Name: "v1", Weight: 1}}})
			}
		}(i)
	}
	wg.Wait()
}
