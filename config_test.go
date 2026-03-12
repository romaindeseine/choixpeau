package main

import (
	"os"
	"path/filepath"
	"testing"
)

func validExperiment() Experiment {
	return Experiment{
		Slug:   "test-exp",
		Status: StatusRunning,
		Variants: []Variant{
			{Name: "control", Weight: 50},
			{Name: "treatment", Weight: 50},
		},
	}
}

func TestValidateSlug(t *testing.T) {
	tests := []struct {
		name    string
		exp     Experiment
		wantErr bool
	}{
		{
			name: "valid slug",
			exp:  validExperiment(),
		},
		{
			name:    "empty slug",
			exp:     Experiment{Slug: "", Status: StatusRunning, Variants: []Variant{{Name: "a", Weight: 1}}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSlug(tt.exp)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSlug() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateStatus(t *testing.T) {
	tests := []struct {
		name    string
		status  ExperimentStatus
		wantErr bool
	}{
		{"draft", StatusDraft, false},
		{"running", StatusRunning, false},
		{"paused", StatusPaused, false},
		{"stopped", StatusStopped, false},
		{"invalid", ExperimentStatus("invalid"), true},
		{"empty", ExperimentStatus(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exp := validExperiment()
			exp.Status = tt.status
			err := validateStatus(exp)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateVariantsNotEmpty(t *testing.T) {
	tests := []struct {
		name    string
		exp     Experiment
		wantErr bool
	}{
		{
			name: "has variants",
			exp:  validExperiment(),
		},
		{
			name:    "no variants",
			exp:     Experiment{Slug: "test", Status: StatusRunning, Variants: nil},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVariantsNotEmpty(tt.exp)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateVariantsNotEmpty() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateVariantName(t *testing.T) {
	tests := []struct {
		name    string
		exp     Experiment
		wantErr bool
	}{
		{
			name: "valid names",
			exp:  validExperiment(),
		},
		{
			name: "empty variant name",
			exp: Experiment{
				Slug:     "test",
				Status:   StatusRunning,
				Variants: []Variant{{Name: "", Weight: 50}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVariantName(tt.exp)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateVariantName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateVariantWeight(t *testing.T) {
	tests := []struct {
		name    string
		exp     Experiment
		wantErr bool
	}{
		{
			name: "positive weights",
			exp:  validExperiment(),
		},
		{
			name: "zero weight",
			exp: Experiment{
				Slug:     "test",
				Status:   StatusRunning,
				Variants: []Variant{{Name: "a", Weight: 0}},
			},
			wantErr: true,
		},
		{
			name: "negative weight",
			exp: Experiment{
				Slug:     "test",
				Status:   StatusRunning,
				Variants: []Variant{{Name: "a", Weight: -1}},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVariantWeight(tt.exp)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateVariantWeight() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUniqueVariants(t *testing.T) {
	tests := []struct {
		name    string
		exp     Experiment
		wantErr bool
	}{
		{
			name: "unique variants",
			exp:  validExperiment(),
		},
		{
			name: "duplicate variant names",
			exp: Experiment{
				Slug:   "test",
				Status: StatusRunning,
				Variants: []Variant{
					{Name: "control", Weight: 50},
					{Name: "control", Weight: 50},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUniqueVariants(tt.exp)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateUniqueVariants() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateOverrides(t *testing.T) {
	tests := []struct {
		name    string
		exp     Experiment
		wantErr bool
	}{
		{
			name: "no overrides",
			exp:  validExperiment(),
		},
		{
			name: "valid override",
			exp: Experiment{
				Slug:      "test",
				Status:    StatusRunning,
				Variants:  []Variant{{Name: "control", Weight: 50}, {Name: "treatment", Weight: 50}},
				Overrides: map[string]string{"user-1": "control"},
			},
		},
		{
			name: "override references unknown variant",
			exp: Experiment{
				Slug:      "test",
				Status:    StatusRunning,
				Variants:  []Variant{{Name: "control", Weight: 50}},
				Overrides: map[string]string{"user-1": "nonexistent"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOverrides(tt.exp)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateOverrides() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUniqueSlugs(t *testing.T) {
	tests := []struct {
		name    string
		exps    []Experiment
		wantErr bool
	}{
		{
			name: "unique slugs",
			exps: []Experiment{
				{Slug: "exp-1"},
				{Slug: "exp-2"},
			},
		},
		{
			name: "duplicate slugs",
			exps: []Experiment{
				{Slug: "exp-1"},
				{Slug: "exp-1"},
			},
			wantErr: true,
		},
		{
			name: "single experiment",
			exps: []Experiment{{Slug: "exp-1"}},
		},
		{
			name: "empty list",
			exps: []Experiment{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUniqueSlugs(tt.exps)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateUniqueSlugs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadExperiments(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantLen int
		wantErr bool
	}{
		{
			name: "valid config",
			yaml: `experiments:
  - slug: checkout-redesign
    status: running
    variants:
      - name: control
        weight: 50
      - name: new_checkout
        weight: 50
`,
			wantLen: 1,
		},
		{
			name: "seed defaults to slug",
			yaml: `experiments:
  - slug: my-exp
    status: draft
    variants:
      - name: a
        weight: 1
`,
			wantLen: 1,
		},
		{
			name: "explicit seed preserved",
			yaml: `experiments:
  - slug: my-exp
    status: draft
    seed: custom-seed
    variants:
      - name: a
        weight: 1
`,
			wantLen: 1,
		},
		{
			name:    "invalid yaml",
			yaml:    `not: [valid: yaml`,
			wantErr: true,
		},
		{
			name: "validation error propagated",
			yaml: `experiments:
  - slug: ""
    status: running
    variants:
      - name: a
        weight: 1
`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "experiments.yaml")
			if err := os.WriteFile(path, []byte(tt.yaml), 0644); err != nil {
				t.Fatal(err)
			}

			exps, err := loadExperiments(path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("loadExperiments() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(exps) != tt.wantLen {
				t.Errorf("loadExperiments() returned %d experiments, want %d", len(exps), tt.wantLen)
			}
		})
	}
}

func TestLoadExperimentsMissingFile(t *testing.T) {
	_, err := loadExperiments("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
