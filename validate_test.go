package pearcut

import "testing"

func TestExperimentValidate(t *testing.T) {
	tests := []struct {
		name    string
		exp     Experiment
		wantErr bool
	}{
		{
			name: "valid experiment",
			exp: Experiment{
				Slug:   "test-exp",
				Status: StatusRunning,
				Variants: []Variant{
					{Name: "control", Weight: 50},
					{Name: "treatment", Weight: 50},
				},
			},
		},
		{
			name:    "empty slug",
			exp:     Experiment{Status: StatusRunning, Variants: []Variant{{Name: "a", Weight: 1}}},
			wantErr: true,
		},
		{
			name:    "invalid status",
			exp:     Experiment{Slug: "test", Status: "bad", Variants: []Variant{{Name: "a", Weight: 1}}},
			wantErr: true,
		},
		{
			name:    "empty status",
			exp:     Experiment{Slug: "test", Status: "", Variants: []Variant{{Name: "a", Weight: 1}}},
			wantErr: true,
		},
		{
			name: "valid draft status",
			exp: Experiment{
				Slug:     "test",
				Status:   StatusDraft,
				Variants: []Variant{{Name: "a", Weight: 1}},
			},
		},
		{
			name: "valid paused status",
			exp: Experiment{
				Slug:     "test",
				Status:   StatusPaused,
				Variants: []Variant{{Name: "a", Weight: 1}},
			},
		},
		{
			name: "valid stopped status",
			exp: Experiment{
				Slug:     "test",
				Status:   StatusStopped,
				Variants: []Variant{{Name: "a", Weight: 1}},
			},
		},
		{
			name:    "no variants",
			exp:     Experiment{Slug: "test", Status: StatusRunning},
			wantErr: true,
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
		{
			name: "valid tags",
			exp: Experiment{
				Slug:     "test",
				Status:   StatusRunning,
				Variants: []Variant{{Name: "a", Weight: 1}},
				Tags:     []string{"checkout", "mobile"},
			},
		},
		{
			name: "empty tag rejected",
			exp: Experiment{
				Slug:     "test",
				Status:   StatusRunning,
				Variants: []Variant{{Name: "a", Weight: 1}},
				Tags:     []string{"checkout", ""},
			},
			wantErr: true,
		},
		{
			name: "nil tags valid",
			exp: Experiment{
				Slug:     "test",
				Status:   StatusRunning,
				Variants: []Variant{{Name: "a", Weight: 1}},
			},
		},
		{
			name: "valid targeting rule in",
			exp: Experiment{
				Slug:           "test",
				Status:         StatusRunning,
				Variants:       []Variant{{Name: "a", Weight: 1}},
				TargetingRules: []TargetingRule{{Attribute: "country", Operator: OperatorIn, Values: []string{"FR"}}},
			},
		},
		{
			name: "valid targeting rule not_in",
			exp: Experiment{
				Slug:           "test",
				Status:         StatusRunning,
				Variants:       []Variant{{Name: "a", Weight: 1}},
				TargetingRules: []TargetingRule{{Attribute: "country", Operator: OperatorNotIn, Values: []string{"CN"}}},
			},
		},
		{
			name: "targeting rule empty attribute",
			exp: Experiment{
				Slug:           "test",
				Status:         StatusRunning,
				Variants:       []Variant{{Name: "a", Weight: 1}},
				TargetingRules: []TargetingRule{{Attribute: "", Operator: OperatorIn, Values: []string{"FR"}}},
			},
			wantErr: true,
		},
		{
			name: "targeting rule invalid operator",
			exp: Experiment{
				Slug:           "test",
				Status:         StatusRunning,
				Variants:       []Variant{{Name: "a", Weight: 1}},
				TargetingRules: []TargetingRule{{Attribute: "country", Operator: "eq", Values: []string{"FR"}}},
			},
			wantErr: true,
		},
		{
			name: "targeting rule empty values",
			exp: Experiment{
				Slug:           "test",
				Status:         StatusRunning,
				Variants:       []Variant{{Name: "a", Weight: 1}},
				TargetingRules: []TargetingRule{{Attribute: "country", Operator: OperatorIn, Values: []string{}}},
			},
			wantErr: true,
		},
		{
			name: "nil targeting rules valid",
			exp: Experiment{
				Slug:     "test",
				Status:   StatusRunning,
				Variants: []Variant{{Name: "a", Weight: 1}},
			},
		},
		{
			name: "valid exclusion_percentage 0 (default, no exclusion)",
			exp: Experiment{
				Slug:                "test",
				Status:              StatusRunning,
				Variants:            []Variant{{Name: "a", Weight: 1}},
				ExclusionPercentage: 0,
			},
		},
		{
			name: "valid exclusion_percentage 50",
			exp: Experiment{
				Slug:                "test",
				Status:              StatusRunning,
				Variants:            []Variant{{Name: "a", Weight: 1}},
				ExclusionPercentage: 50,
			},
		},
		{
			name: "valid exclusion_percentage 100",
			exp: Experiment{
				Slug:                "test",
				Status:              StatusRunning,
				Variants:            []Variant{{Name: "a", Weight: 1}},
				ExclusionPercentage: 100,
			},
		},
		{
			name: "negative exclusion_percentage",
			exp: Experiment{
				Slug:                "test",
				Status:              StatusRunning,
				Variants:            []Variant{{Name: "a", Weight: 1}},
				ExclusionPercentage: -1,
			},
			wantErr: true,
		},
		{
			name: "exclusion_percentage over 100",
			exp: Experiment{
				Slug:                "test",
				Status:              StatusRunning,
				Variants:            []Variant{{Name: "a", Weight: 1}},
				ExclusionPercentage: 101,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.exp.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
